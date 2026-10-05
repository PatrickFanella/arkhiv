import json

import httpx
import pytest
from fastapi.testclient import TestClient

from arkhiv_curation.app import create_app
from arkhiv_curation.archive import ArchiveClient, query_for


class Archive:
    def __init__(self):
        self.calls = []
        self.pages = {}
        self.fail = None

    def fetch(self, query, cursor):
        self.calls.append((query, cursor))
        if self.fail:
            raise self.fail
        return self.pages.get(cursor, {"items": [], "cursor": None})


@pytest.fixture
def workspace(tmp_path):
    archive = Archive()
    app = create_app(str(tmp_path / "test.sqlite"), archive=archive, start_worker=False)
    with TestClient(app) as client:
        collection = client.post("/api/collections", json={"name": "Print culture"}).json()
        search = client.post(
            f"/api/collections/{collection['id']}/searches",
            json={
                "name": "Magazines",
                "query": 'subject:"little magazines"',
                "media_type": "texts",
                "year_from": 1900,
                "year_to": 1999,
            },
        ).json()
        yield client, app, archive, collection["id"], search["id"]


def record(identifier="first", **values):
    return {
        "identifier": identifier,
        "title": "A little magazine",
        "mediatype": "texts",
        "description": "<p>Archive description</p>",
        **values,
    }


def run(client, app, search):
    response = client.post(f"/api/searches/{search}/run", json={})
    assert response.status_code == 202
    assert app.state.worker.run_one()
    return response.json()["run_id"]


def test_resume_deduplicate_and_preserve_decisions(workspace):
    client, app, archive, collection, search = workspace
    app.state.worker.pages_per_run = 1
    archive.pages = {
        None: {"items": [record()], "cursor": "page2"},
        "page2": {"items": [record(), record("second")], "cursor": None},
    }
    run(client, app, search)
    response = client.patch(
        f"/api/collections/{collection}/items/first",
        json={"status": "rejected", "notes": "Wrong topic"},
    )
    assert response.status_code == 200
    run(client, app, search)
    assert [cursor for _, cursor in archive.calls] == [None, "page2"]
    pending = client.get(f"/api/collections/{collection}/items").json()
    assert pending["total"] == 1
    assert pending["items"][0]["identifier"] == "second"
    rejected = client.get(f"/api/collections/{collection}/items?status=rejected").json()
    assert rejected["items"][0]["notes"] == "Wrong topic"
    searches = client.get(f"/api/collections/{collection}/searches").json()
    assert searches[0]["cursor"] is None
    # A fresh pass still leaves the rejection and note intact.
    run(client, app, search)
    assert client.get(f"/api/collections/{collection}/items?status=rejected").json()["total"] == 1


def test_overlapping_searches_and_collection_scoping(workspace):
    client, app, archive, collection, search = workspace
    archive.pages = {None: {"items": [record()], "cursor": None}}
    second = client.post(
        f"/api/collections/{collection}/searches", json={"name": "Alternative", "query": "magazine"}
    ).json()["id"]
    run(client, app, search)
    run(client, app, second)
    result = client.get(f"/api/collections/{collection}/items").json()
    assert result["total"] == 1
    assert len(result["items"][0]["matches"]) == 2
    other = client.post("/api/collections", json={"name": "Other"}).json()["id"]
    assert (
        client.patch(
            f"/api/collections/{other}/items/first", json={"status": "accepted"}
        ).status_code
        == 404
    )
    assert client.get(f"/api/collections/{other}/items").json()["total"] == 0


def test_failure_keeps_cursor_and_records_error(workspace):
    client, app, archive, collection, search = workspace
    app.state.worker.pages_per_run = 1
    archive.pages = {None: {"items": [record()], "cursor": "saved"}}
    run(client, app, search)
    archive.fail = httpx.ReadTimeout("timeout")
    run(client, app, search)
    history = client.get(f"/api/collections/{collection}/runs").json()
    assert history[0]["state"] == "failed"
    assert "Could not reach Archive" in history[0]["error"]
    assert client.get(f"/api/collections/{collection}/searches").json()[0]["cursor"] == "saved"
    archive.fail = None
    archive.pages["saved"] = {"items": [record("second")], "cursor": None}
    run(client, app, search)
    assert archive.calls[-1][1] == "saved"
    assert client.get(f"/api/collections/{collection}/items").json()["total"] == 2


def test_schedule_pause_overlap_and_interrupted_recovery(workspace):
    client, app, _archive, collection, search = workspace
    worker = app.state.worker
    worker.schedule(now=10**12)
    first = worker.enqueue(search)
    assert worker.enqueue(search) == first
    # Edits cannot change a queued/running query and invalidate its cursor.
    assert (
        client.put(
            f"/api/searches/{search}", json={"name": "Changed", "query": "other"}
        ).status_code
        == 409
    )
    with app.state.database.connect() as db:
        db.execute("UPDATE runs SET state='running' WHERE id=?", (first,))
    worker.recover()
    assert worker.run_one()
    search_data = client.get(f"/api/collections/{collection}/searches").json()[0]
    assert (
        client.put(f"/api/searches/{search}", json=search_data | {"enabled": False}).status_code
        == 200
    )
    worker.schedule(now=10**12)
    assert not worker.run_one()
    assert client.get(f"/api/collections/{collection}/runs").json()[0]["state"] == "completed"


def test_exports_only_accepted_and_protect_csv(workspace):
    client, app, archive, collection, search = workspace
    archive.pages = {
        None: {
            "items": [record(title="=SUM(1,2)"), record("pending"), record("passed")],
            "cursor": None,
        }
    }
    run(client, app, search)
    client.patch(
        f"/api/collections/{collection}/items/first", json={"status": "accepted", "notes": "Keep"}
    )
    client.patch(f"/api/collections/{collection}/items/passed", json={"status": "rejected"})
    exported = client.get(f"/api/collections/{collection}/export").json()
    assert [i["identifier"] for i in exported["items"]] == ["first"]
    assert exported["items"][0]["notes"] == "Keep"
    csv = client.get(f"/api/collections/{collection}/export?format=csv").text
    assert "'=SUM" in csv
    assert "passed" not in csv and "pending" not in csv
    inquiry = client.get(f"/api/collections/{collection}/export?format=inquiry").text
    assert "not a claim that I uploaded" in inquiry
    assert "https://archive.org/details/first" in inquiry
    assert "https://archive.org/details/pending" not in inquiry


def test_validation_query_filters_and_browser_write_boundary(workspace):
    client, _app, _archive, collection, _search = workspace
    assert client.post("/api/collections", json={"name": "  "}).status_code == 422
    assert (
        client.post(
            f"/api/collections/{collection}/searches",
            json={
                "name": "Invalid",
                "query": "test",
                "year_from": 2000,
                "year_to": 1900,
            },
        ).status_code
        == 422
    )
    assert (
        client.post(
            "/api/collections", json={"name": "Bad"}, headers={"Origin": "https://other.example"}
        ).status_code
        == 403
    )
    assert (
        client.post(
            "/api/collections", content='{"name":"Bad"}', headers={"Content-Type": "text/plain"}
        ).status_code
        == 415
    )
    assert client.get(f"/api/collections/{collection}/items?page=0").status_code == 422
    assert client.get("/api/collections/999/items").status_code == 404
    query = query_for(
        {"query": "a OR b", "media_type": "texts", "source_collection": 'a"b', "year_from": 1900}
    )
    assert query.startswith("(a OR b) AND NOT mediatype:collection")
    assert 'collection:"a\\"b"' in query and "year:[1900 TO *]" in query


def test_repeated_cursor_fails_instead_of_endless_scan(workspace):
    client, app, archive, collection, search = workspace
    archive.pages = {
        None: {"items": [record()], "cursor": "same"},
        "same": {"items": [record("second")], "cursor": "same"},
    }
    run(client, app, search)
    history = client.get(f"/api/collections/{collection}/runs").json()
    assert history[0]["state"] == "failed"
    assert "repeated" in history[0]["error"]
    assert client.get(f"/api/collections/{collection}/items").json()["total"] == 1


def test_client_binds_checkpoint_to_query_and_checks_echo(monkeypatch):
    requests = []

    def handler(request):
        requests.append(request)
        query = request.url.params["q"]
        return httpx.Response(
            200,
            json={
                "responseHeader": {"params": {"qin": query}},
                "response": {"docs": [{"identifier": "first"}], "numFound": 2},
            },
        )

    original = httpx.Client
    monkeypatch.setattr(
        httpx, "Client", lambda **kwargs: original(**kwargs, transport=httpx.MockTransport(handler))
    )
    archive = ArchiveClient()
    first = archive.fetch("topic", None)
    archive.fetch("topic", first["cursor"])
    assert requests[0].url.params["sort[]"] == "identifier asc"
    assert "identifier:[first TO *] AND NOT identifier:first" in requests[1].url.params["q"]
    assert requests[1].url.params["page"] == "1"
    with pytest.raises(ValueError, match="incompatible"):
        archive.fetch("other topic", first["cursor"])
    with pytest.raises(ValueError, match="incompatible"):
        archive.fetch("topic", "old-provider-cursor")


def test_client_rejects_results_for_another_query(monkeypatch):
    original = httpx.Client
    handler = lambda request: httpx.Response(
        200,
        json={
            "responseHeader": {"params": {"qin": "another query"}},
            "response": {"docs": [{"identifier": "wrong"}], "numFound": 1},
        },
    )
    monkeypatch.setattr(
        httpx, "Client", lambda **kwargs: original(**kwargs, transport=httpx.MockTransport(handler))
    )
    with pytest.raises(ValueError, match="different query"):
        ArchiveClient().fetch("requested query", None)


def test_decisions_survive_app_restart(workspace):
    client, app, archive, collection, search = workspace
    archive.pages = {None: {"items": [record()], "cursor": None}}
    run(client, app, search)
    client.patch(f"/api/collections/{collection}/items/first", json={"status": "accepted"})
    restarted = create_app(app.state.database.path, archive=archive, start_worker=False)
    with TestClient(restarted) as second:
        result = second.get(f"/api/collections/{collection}/items?status=accepted").json()
        assert result["total"] == 1
        assert (
            json.loads(second.get(f"/api/collections/{collection}/export").text)["items"][0][
                "identifier"
            ]
            == "first"
        )


def test_restart_scan_preserves_review_decisions(workspace):
    client, app, archive, collection, search = workspace
    app.state.worker.pages_per_run = 1
    archive.pages = {None: {"items": [record()], "cursor": "expired"}}
    run(client, app, search)
    client.patch(f"/api/collections/{collection}/items/first", json={"status": "accepted"})
    response = client.post(f"/api/searches/{search}/restart", json={})
    assert response.status_code == 202
    assert client.post(f"/api/searches/{search}/restart", json={}).status_code == 409
    assert app.state.worker.run_one()
    assert archive.calls[-1][1] is None
    assert client.get(f"/api/collections/{collection}/items?status=accepted").json()["total"] == 1


def test_interrupted_run_keeps_page_accounting(workspace):
    client, app, archive, collection, search = workspace
    run_id = app.state.worker.enqueue(search)
    with app.state.database.connect() as db:
        db.execute("UPDATE runs SET state='running',scanned=100,added=80 WHERE id=?", (run_id,))
        db.execute("UPDATE searches SET cursor='checkpoint' WHERE id=?", (search,))
    archive.pages = {"checkpoint": {"items": [record()], "cursor": None}}
    app.state.worker.recover()
    assert app.state.worker.run_one()
    result = client.get(f"/api/collections/{collection}/runs").json()[0]
    assert result["scanned"] == 101 and result["added"] == 81


def test_query_edit_resets_cursor_without_discarding_items(workspace):
    client, app, archive, collection, search = workspace
    app.state.worker.pages_per_run = 1
    archive.pages = {None: {"items": [record()], "cursor": "saved"}}
    run(client, app, search)
    value = client.get(f"/api/collections/{collection}/searches").json()[0]
    assert (
        client.put(f"/api/searches/{search}", json=value | {"query": "new topic"}).status_code
        == 200
    )
    assert client.get(f"/api/collections/{collection}/searches").json()[0]["cursor"] is None
    assert client.get(f"/api/collections/{collection}/items").json()["total"] == 1
