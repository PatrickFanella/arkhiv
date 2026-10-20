import json
from pathlib import Path
import tempfile
import unittest

from enrich_source_artist import IMAGE_HOSTS, Fetcher, commons_url, add_archive, archive_description, build_bundle, validate_url

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

    def test_commons_lead_image_and_license_page_fallbacks(self):
        calls = []

        def fetch(url):
            calls.append(url)
            if "musicbrainz.org/" in url:
                return {"id": ID, "relations": [{"type": "wikidata", "url": {"resource": "https://www.wikidata.org/wiki/Q123"}}]}
            if "wikidata.org/" in url:
                return {"entities": {"Q123": {"sitelinks": {"enwiki": {"title": "Fixture artist"}}, "claims": {}}}}
            if "en.wikipedia.org/" in url:
                return {"extract": "Bio.", "originalimage": {"source": "https://upload.wikimedia.org/wikipedia/commons/thumb/a/ab/Lead_%28live%29.jpg/1920px-Lead_%28live%29.jpg"}}
            return {"query": {"pages": {"1": {"imageinfo": [{"thumburl": "https://upload.wikimedia.org/wikipedia/commons/a/ab/Lead_(live).jpg", "descriptionurl": "https://commons.wikimedia.org/wiki/File:Lead_(live).jpg", "extmetadata": {"Artist": {"value": "Photographer"}, "LicenseShortName": {"value": "GFDL 1.2"}, "LicenseUrl": {"value": "https://www.gnu.org/licenses/old-licenses/fdl-1.2.html"}}}]}}}}

        bundle = build_bundle(ID, fetch)
        self.assertIn("File%3ALead_%28live%29.jpg", calls[-1])
        self.assertEqual(bundle["photo"]["license_url"], "https://commons.wikimedia.org/wiki/File:Lead_(live).jpg#Licensing")
        self.assertEqual(bundle["photo"]["license"], "GFDL 1.2")

    def test_locally_hosted_lead_image_is_not_used(self):
        def fetch(url):
            if "musicbrainz.org/" in url:
                return {"id": ID, "relations": [{"type": "wikidata", "url": {"resource": "https://www.wikidata.org/wiki/Q123"}}]}
            if "wikidata.org/" in url:
                return {"entities": {"Q123": {"sitelinks": {"enwiki": {"title": "Fixture artist"}}, "claims": {}}}}
            if "en.wikipedia.org/" in url:
                return {"extract": "Bio.", "originalimage": {"source": "https://upload.wikimedia.org/wikipedia/en/a/ab/Album_cover.jpg"}}
            raise AssertionError("Commons queried for a non-Commons image: " + url)

        self.assertNotIn("photo", build_bundle(ID, fetch))

    def test_no_identity_guess_or_uncredited_photo(self):
        with self.assertRaises(ValueError):
            build_bundle("artist name", lambda _: {})
        with self.assertRaises(ValueError):
            build_bundle(ID, lambda _: {"id": "other"})
        self.assertNotIn("biography", build_bundle(ID, lambda _: {"id": ID}))
        for url in ["http://musicbrainz.org/ws/2", "https://localhost/", "https://musicbrainz.org.evil.invalid/", "https://user:secret@musicbrainz.org/", "https://musicbrainz.org:444/"]:
            with self.assertRaises(ValueError):
                validate_url(url)

    def test_archive_collection_text_and_image(self):
        description = "Ween<br>rn rn Ween formed in New Hope, Pennsylvania, in 1984 and recorded for Twin/Tone, Elektra and Sanctuary before starting its own label, playing long improvised sets across the country. Download free music from Ween. Concerts in the Live Music Archive are available for download and streaming in formats including flac, mp3, and ogg vorbis."
        response = {"metadata": {"identifier": "Ween", "mediatype": "collection", "title": "Ween", "collection": ["etree"], "description": description},
                    "files": [{"name": "act.jpg", "format": "JPEG"}, {"name": "Ween_itemimage.jpg", "format": "Item Image"}]}
        images = []

        def image(url):
            images.append(url)
            return b"jpeg bytes"

        bundle = add_archive({}, "Ween", lambda url: response, image)
        self.assertEqual(images, ["https://archive.org/download/Ween/Ween_itemimage.jpg"])
        self.assertEqual(bundle["archive_collection"], "Ween")
        self.assertTrue(bundle["biography"]["text"].startswith("Ween formed in New Hope"))
        self.assertNotIn("Download free music", bundle["biography"]["text"])
        self.assertNotIn(" rn ", bundle["biography"]["text"])
        self.assertEqual(bundle["biography"]["license"], "Internet Archive collection description")
        self.assertEqual(bundle["photo"]["data_url"], bundle["photo"]["url"])
        self.assertEqual(bundle["photo"]["license_url"], "")
        kept = add_archive({"photo": {"url": "commons"}, "biography": {"text": "Wikipedia"}}, "Ween", lambda url: response, image)
        self.assertEqual((kept["photo"], kept["biography"]), ({"url": "commons"}, {"text": "Wikipedia"}))
        self.assertNotIn("photo", add_archive({}, "Ween", lambda url: response, image, use_image=False))
        for bad in ({"identifier": "Other", "mediatype": "collection", "title": "Ween", "collection": ["etree"]}, {"identifier": "Ween", "mediatype": "etree", "title": "Ween", "collection": ["etree"]}):
            with self.assertRaises(ValueError):
                add_archive({}, "Ween", lambda url: {"metadata": bad}, image)

    def test_generated_collection_text_is_not_a_biography(self):
        self.assertIsNone(archive_description("Plan J Download free music from Plan J. Concerts in the Live Music Archive are available for download and streaming in formats including flac, mp3, and ogg vorbis.", "Plan J"))
        self.assertIsNone(archive_description("Free Stockholm Syndrome concerts. Trade-friendly Live Music Archive concerts are available for download and/or streaming in formats including mp3, flac, and ogg vorbis. Limited Flag: NoSBD", "Stockholm Syndrome"))

    def test_collection_text_drops_taping_notes_and_joins_wrapped_lines(self):
        text = archive_description("Escaper is a Brooklyn-based space rock band named an \"On the\n\nVerge\" artist by Relix Magazine,\nwith funky dance parties at festivals across the northeast and long improvised sets.\n\nThe band allows audience taping; soundboard patches on request.", "Escaper")
        self.assertEqual(text, "Escaper is a Brooklyn-based space rock band named an \"On the Verge\" artist by Relix Magazine, with funky dance parties at festivals across the northeast and long improvised sets.")
        self.assertIsNone(archive_description("On May 30, 2019, permission was granted to the LMA: Just wanted to give the go ahead and allow the LMA permission to create a section for the band and its many recordings from shows.", "Band"))

    def test_commons_thumbnail_host_and_tracking_parameters(self):
        self.assertEqual(commons_url("https://thumb.wikimedia.org/wikipedia/commons/thumb/7/7d/K.jpg/1280px-K.jpg?utm_source=commons.wikimedia.org"), "https://upload.wikimedia.org/wikipedia/commons/thumb/7/7d/K.jpg/1280px-K.jpg")
        self.assertEqual(commons_url("https://upload.wikimedia.org/wikipedia/commons/7/7d/K.jpg?utm_content=original"), "https://upload.wikimedia.org/wikipedia/commons/7/7d/K.jpg")
        self.assertEqual(commons_url("https://thumb.wikimedia.org/other/K.jpg"), "https://thumb.wikimedia.org/other/K.jpg")

    def test_commons_photo_bytes_and_image_hosts(self):
        def fetch(url):
            if "musicbrainz.org/" in url:
                return {"id": ID, "relations": [{"type": "wikidata", "url": {"resource": "https://www.wikidata.org/wiki/Q123"}}]}
            if "wikidata.org/" in url:
                return {"entities": {"Q123": {"claims": {"P18": [{"mainsnak": {"datavalue": {"value": "Fixture.jpg"}}}]}}}}
            return {"query": {"pages": {"1": {"imageinfo": [{"thumburl": "https://upload.wikimedia.org/fixture.jpg", "descriptionurl": "https://commons.wikimedia.org/wiki/File:Fixture.jpg", "extmetadata": {"Artist": {"value": "Photographer"}, "LicenseShortName": {"value": "CC BY 4.0"}, "LicenseUrl": {"value": "https://creativecommons.org/licenses/by/4.0/"}}}]}}}}

        bundle = build_bundle(ID, fetch, lambda url: b"\xff\xd8")
        self.assertEqual(bundle["photo"]["data_url"], "https://upload.wikimedia.org/fixture.jpg")
        self.assertEqual(bundle["photo"]["data_base64"], "/9g=")
        validate_url("https://ia800602.us.archive.org/33/items/Ween/Ween_itemimage.jpg", IMAGE_HOSTS, nodes=True)
        for url in ["https://archive.org.evil.invalid/x.jpg", "https://upload.wikimedia.org.evil.invalid/x.jpg", "http://archive.org/x.jpg"]:
            with self.assertRaises(ValueError):
                validate_url(url, IMAGE_HOSTS, nodes=True)

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
