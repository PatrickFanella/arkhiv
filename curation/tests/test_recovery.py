import sqlite3

import pytest
from fastapi.testclient import TestClient

from arkhiv_curation.app import create_app
from arkhiv_curation.recovery import checkpoint


def test_online_checkpoint_and_new_path_restore_preserve_review_and_progress(tmp_path):
    source = tmp_path / "live.sqlite"
    app = create_app(str(source), start_worker=False)
    with TestClient(app) as client:
        cid = client.post("/api/collections", json={"name": "Recovery qualification"}).json()["id"]
        sid = client.post(
            f"/api/collections/{cid}/searches",
            json={
                "name": "Saved search",
                "query": "zines",
                "enabled": False,
            },
        ).json()["id"]
        with app.state.database.connect() as db:
            db.execute("UPDATE searches SET cursor='retained-checkpoint' WHERE id=?", (sid,))
            db.execute(
                "INSERT INTO items(collection_id,identifier,title,creator,description,media_type,"
                "year,subjects,source_collections,discovered_at,status,notes) "
                "VALUES (?,'reviewed','Zine','','','texts','','','',1,'accepted','Page evidence')",
                (cid,),
            )
        backup = tmp_path / "backup.sqlite"
        receipt = checkpoint(source, backup)
        assert receipt["integrity"] == "ok"
        assert receipt["table_counts"]["items"] == 1
        assert backup.stat().st_mode & 0o777 == 0o600
        # Subsequent decisions in the live workspace must not change its checkpoint.
        client.patch(f"/api/collections/{cid}/items/reviewed", json={"status": "rejected"})
        restored = tmp_path / "restored.sqlite"
        checkpoint(backup, restored)
    recovered = create_app(str(restored), start_worker=False)
    with TestClient(recovered) as client:
        item = client.get(f"/api/collections/{cid}/items?status=accepted").json()["items"][0]
        assert item["notes"] == "Page evidence"
        assert (
            client.get(f"/api/collections/{cid}/searches").json()[0]["cursor"]
            == "retained-checkpoint"
        )
        assert len(client.get(f"/api/collections/{cid}/export").json()["items"]) == 1


def test_checkpoint_refuses_overwrite_and_unrelated_databases(tmp_path):
    source = tmp_path / "source.sqlite"
    create_app(str(source), start_worker=False)
    protected = tmp_path / "existing.sqlite"
    protected.write_bytes(b"existing recovery data")
    with pytest.raises(ValueError, match="already exists"):
        checkpoint(source, protected)
    assert protected.read_bytes() == b"existing recovery data"
    unrelated = tmp_path / "unrelated.sqlite"
    with sqlite3.connect(unrelated) as db:
        db.execute("CREATE TABLE other(id INTEGER)")
    destination = tmp_path / "invalid.sqlite"
    with pytest.raises(ValueError, match="complete curation workspace"):
        checkpoint(unrelated, destination)
    assert not destination.exists()
