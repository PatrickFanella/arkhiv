import json
from pathlib import Path
import tempfile
import unittest

from enrich_source_artist import Fetcher, build_bundle, validate_url

ID = "00000000-0000-0000-0000-000000000001"


class ArtistEnrichmentTests(unittest.TestCase):
    def test_verified_identity_links_biography_and_credited_photo(self):
        calls = []

        def fetch(url):
            calls.append(url)
            if "musicbrainz.org/" in url:
                return {"id": ID, "relations": [{"type": "wikidata", "url": {"resource": "https://www.wikidata.org/wiki/Q123"}}]}
            if "wikidata.org/" in url:
                return {"entities": {"Q123": {"sitelinks": {"enwiki": {"title": "Fixture artist"}}, "claims": {"P18": [{"mainsnak": {"datavalue": {"value": "Fixture.jpg"}}}]}}}}
            if "en.wikipedia.org/" in url:
                return {"extract": "A sourced fixture biography."}
            return {"query": {"pages": {"1": {"imageinfo": [{"thumburl": "https://upload.wikimedia.org/fixture.jpg", "descriptionurl": "https://commons.wikimedia.org/wiki/File:Fixture.jpg", "extmetadata": {"Artist": {"value": "<a>Fixture photographer</a>"}, "LicenseShortName": {"value": "CC BY-SA 4.0"}, "LicenseUrl": {"value": "https://creativecommons.org/licenses/by-sa/4.0/"}}}]}}}}

        bundle = build_bundle(ID, fetch)
        self.assertEqual(bundle["biography"]["text"], "A sourced fixture biography.")
        self.assertEqual(bundle["photo"]["credit"], "Fixture photographer")
        self.assertEqual(len(calls), 4)
        self.assertTrue(all("/artist?query=" not in call for call in calls))

    def test_no_identity_guess_or_uncredited_photo(self):
        with self.assertRaises(ValueError):
            build_bundle("artist name", lambda _: {})
        with self.assertRaises(ValueError):
            build_bundle(ID, lambda _: {"id": "other"})
        self.assertNotIn("biography", build_bundle(ID, lambda _: {"id": ID}))
        for url in ["http://musicbrainz.org/ws/2", "https://localhost/", "https://musicbrainz.org.evil.invalid/", "https://user:secret@musicbrainz.org/", "https://musicbrainz.org:444/"]:
            with self.assertRaises(ValueError):
                validate_url(url)

    def test_cached_snapshot_preserves_original_bytes_and_timestamp(self):
        import base64
        import hashlib
        import time
        with tempfile.TemporaryDirectory() as directory:
            url = "https://musicbrainz.org/ws/2/artist/" + ID
            raw = b'{ "id": "' + ID.encode() + b'" }'
            source = {"url": url, "raw_base64": base64.b64encode(raw).decode(), "fetched_at": "2026-10-19T00:00:00Z"}
            key = hashlib.sha256(url.encode()).hexdigest()
            Path(directory, key + ".json").write_text(json.dumps({"epoch": time.time(), "source": source}))
            fetch = Fetcher(directory)
            self.assertEqual(fetch(url)["id"], ID)
            self.assertEqual(fetch.sources, [source])


if __name__ == "__main__":
    unittest.main()
