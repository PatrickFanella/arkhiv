import copy

import httpx
from fastapi.testclient import TestClient

from arkhiv_curation.app import create_app
from arkhiv_curation.core import CoreClient, evidence_hash


def catalog_item():
    return {
        "id": "mixed",
        "title": "Reviewed collection item",
        "creator": "Credited creator",
        "description": "Source metadata",
        "date": "1977-05-08",
        "collections": ["original"],
        "representations": [
            {
                "id": "scan",
                "media": "print",
                "source_item_id": "zine",
                "snapshot_sha256": "a" * 64,
                "file": "scan.pdf",
                "collections": ["original"],
                "extension": {"print": {"document_id": "zine", "variant": "scan", "pages": 12}},
            },
            {
                "id": "tape",
                "media": "audio",
                "source_item_id": "tape-source",
                "snapshot_sha256": "b" * 64,
                "file": "track.flac",
                "collections": ["original"],
                "extension": {
                    "audio": {
                        "recording_id": "tape-source",
                        "credits": {"taper": "Original taper"},
                        "tracks": [
                            {
                                "id": "one",
                                "file": "track.flac",
                                "title": "Song",
                                "set": "II",
                                "set_reviewed": True,
                            }
                        ],
                    }
                },
            },
        ],
    }


def workspace(tmp_path):
    evidence = catalog_item()
    calls = []

    def respond(request):
        calls.append(request)
        assert request.method == "GET"
        if request.url.path == "/api/catalog/v1/items":
            return httpx.Response(
                200, json={"items": [evidence], "total": 1, "page": int(request.url.params["page"])}
            )
        return httpx.Response(200, json=evidence)

    core = CoreClient("http://core.test", transport=httpx.MockTransport(respond))
    app = create_app(str(tmp_path / "curation.sqlite"), start_worker=False, core=core)
    return app, evidence, calls, core


def attach(client, collection, representations):
    return client.post(
        f"/api/collections/{collection}/core/attachments",
        json={
            "item_id": "mixed",
            "representations": representations,
            "notes": "Reviewed source and credits",
        },
    )


def test_mixed_curation_manifest_retains_evidence_without_access(tmp_path):
    app, _evidence, calls, core = workspace(tmp_path)
    with TestClient(app) as client:
        cid = client.post("/api/collections", json={"name": "Mixed curation"}).json()["id"]
        browse = client.get("/api/core/catalog?media=audio&q=reviewed&page=2")
        assert browse.status_code == 200
        assert calls[-1].url.params["media"] == "audio"
        assert calls[-1].url.params["q"] == "reviewed"
        assert attach(client, cid, ["scan", "tape"]).status_code == 201
        assert client.get(f"/api/collections/{cid}/items").json()["total"] == 2
        assert client.get(f"/api/collections/{cid}/manifest").status_code == 409
        assert (
            client.put(
                f"/api/collections/{cid}/core", json={"target_collection": "target"}
            ).status_code
            == 200
        )
        client.patch(f"/api/collections/{cid}/items/zine", json={"status": "accepted"})
        assert client.get(f"/api/collections/{cid}/manifest").json()["mappings"] == []
        client.patch(
            f"/api/collections/{cid}/items/tape-source",
            json={"status": "accepted", "notes": "Credits checked"},
        )
        result = client.get(f"/api/collections/{cid}/manifest").json()
        assert result["schema"] == "arkhiv.curation.v1"
        assert result["target_collection"] == "target"
        assert not any(result["access"].values())
        mapping = result["mappings"][0]
        assert not any(mapping["access"].values())
        assert mapping["catalog_item"]["representations"][1]["extension"]["audio"]["credits"] == {
            "taper": "Original taper"
        }
        assert mapping["evidence_sha256"] == evidence_hash(core.item("mixed"))
        assert result["unresolved_items"] == []
        # Withdrawal from the local selection immediately removes its mapping.
        client.patch(f"/api/collections/{cid}/items/tape-source", json={"status": "rejected"})
        assert client.get(f"/api/collections/{cid}/manifest").json()["mappings"] == []
    restarted = create_app(app.state.database.path, start_worker=False, core=core)
    with TestClient(restarted) as client:
        assert len(client.get(f"/api/collections/{cid}/core").json()["attachments"]) == 1
        assert (
            client.get(f"/api/collections/{cid}/items?status=rejected").json()["items"][0]["notes"]
            == "Credits checked"
        )


def test_attachment_preserves_decisions_and_checks_fresh_source(tmp_path):
    app, evidence, _calls, core = workspace(tmp_path)
    with TestClient(app) as client:
        cid = client.post("/api/collections", json={"name": "Print"}).json()["id"]
        attach(client, cid, ["scan"])
        client.patch(
            f"/api/collections/{cid}/items/zine",
            json={"status": "rejected", "notes": "Keep rejection"},
        )
        assert attach(client, cid, ["scan"]).status_code == 201
        assert client.get(f"/api/collections/{cid}/items?status=rejected").json()["total"] == 1
        client.patch(f"/api/collections/{cid}/items/zine", json={"status": "accepted"})
        client.put(f"/api/collections/{cid}/core", json={"target_collection": "print"})
        original = copy.deepcopy(evidence)
        evidence["representations"][0]["snapshot_sha256"] = "c" * 64
        assert client.get(f"/api/collections/{cid}/manifest").status_code == 409
        evidence.clear()
        evidence.update(original)
        core.base_url = "http://replacement.test"
        assert client.get(f"/api/collections/{cid}/manifest").status_code == 409
        core.base_url = "http://core.test"
        other = client.post("/api/collections", json={"name": "Other"}).json()["id"]
        assert client.get(f"/api/collections/{other}/core").json()["attachments"] == []
        assert attach(client, cid, ["missing"]).status_code == 422
        assert attach(client, cid, ["scan", "scan"]).status_code == 422
        assert (
            client.post(
                f"/api/collections/{cid}/core/attachments",
                json={"item_id": "mixed", "representations": ["scan"], "notes": " "},
            ).status_code
            == 422
        )
        assert client.get("/api/core/catalog?page=0").status_code == 422
        assert client.get("/api/core/catalog?media=movies").status_code == 422
        assert (
            client.post(
                f"/api/collections/{cid}/core/attachments",
                json={"item_id": "mixed", "representations": ["scan"], "notes": "Reviewed"},
                headers={"Origin": "https://evil.test"},
            ).status_code
            == 403
        )


def test_disabled_connection_and_incompatible_response(tmp_path):
    app = create_app(str(tmp_path / "local.sqlite"), start_worker=False)
    with TestClient(app) as client:
        assert client.get("/api/core/status").json()["configured"] is False
        assert client.get("/api/core/catalog").status_code == 503
        assert client.post("/api/collections", json={"name": "Still local"}).status_code == 201
    app, evidence, _calls, _core = workspace(tmp_path)
    evidence["representations"][0]["file"] = "../restricted.pdf"
    with TestClient(app) as client:
        assert client.get("/api/core/catalog").status_code == 503


def test_native_video_projection_can_be_curated(tmp_path):
    app, evidence, _calls, _core = workspace(tmp_path)
    evidence["id"] = "legacy-video:episode-id"
    evidence["representations"] = [
        {
            "id": "video",
            "media": "video",
            "source_item_id": "video-source",
            "snapshot_sha256": "c" * 64,
            "file": "episode.mp4",
            "collections": ["original"],
            "legacy_episode_id": "episode-id",
            "extension": {"video": {"episode_id": "episode-id", "captions": []}},
        }
    ]
    with TestClient(app) as client:
        cid = client.post("/api/collections", json={"name": "Video selection"}).json()["id"]
        assert client.get("/api/core/catalog?media=video").status_code == 200
        response = client.post(
            f"/api/collections/{cid}/core/attachments",
            json={
                "item_id": evidence["id"],
                "representations": ["video"],
                "notes": "Reviewed episode metadata",
            },
        )
        assert response.status_code == 201
        client.patch(f"/api/collections/{cid}/items/video-source", json={"status": "accepted"})
        client.put(f"/api/collections/{cid}/core", json={"target_collection": "selected-videos"})
        exported = client.get(f"/api/collections/{cid}/manifest").json()
        assert (
            exported["mappings"][0]["catalog_item"]["representations"][0]["legacy_episode_id"]
            == "episode-id"
        )
        assert not any(exported["access"].values())
