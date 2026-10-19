"""Build a retained artist bundle from an explicitly verified MusicBrainz ID.

No name search, audio requests, or browser-triggered provider calls. Import the
result with vhness import-source-artist and explicit operator review notes.
"""
import argparse
import base64
import datetime as dt
import fcntl
import hashlib
import html
import json
import os
from pathlib import Path
import re
import time
import urllib.error
import urllib.parse
import urllib.request

MBID = re.compile(r"[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}")
AGENT = "ArkhivArtistEnrichment/1.0 (https://git.subcult.tv/subculture-collective/arkhiv)"
HOSTS = {"musicbrainz.org", "www.wikidata.org", "en.wikipedia.org", "commons.wikimedia.org"}


class Redirects(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        validate_url(newurl)
        return super().redirect_request(req, fp, code, msg, headers, newurl)


def validate_url(url):
    parsed = urllib.parse.urlsplit(url)
    if parsed.scheme != "https" or parsed.hostname not in HOSTS or parsed.username or parsed.port:
        raise ValueError("Unexpected artist metadata endpoint")


class Fetcher:
    def __init__(self, cache, refresh=False):
        self.cache = Path(cache)
        self.cache.mkdir(parents=True, exist_ok=True)
        self.refresh = refresh
        self.sources = []

    def __call__(self, url):
        validate_url(url)
        key = hashlib.sha256(url.encode()).hexdigest()
        cached = self.cache / (key + ".json")
        if not self.refresh and cached.exists():
            saved = json.loads(cached.read_text())
            if time.time() - saved["epoch"] < 7 * 86400:
                self.sources.append(saved["source"])
                return json.loads(base64.b64decode(saved["source"]["raw_base64"]))
        # Serialize outbound metadata calls across invocations on this host.
        # This is deliberately slower than MusicBrainz's one-call/second limit.
        with (self.cache / "provider.lock").open("a+") as lock:
            fcntl.flock(lock, fcntl.LOCK_EX)
            pacing = self.cache / "next-request"
            next_request = float(pacing.read_text()) if pacing.exists() else 0
            wait = next_request - time.time()
            if wait > 2:
                raise RuntimeError(f"Provider cooldown remains active for {wait:.0f} seconds")
            if wait > 0:
                time.sleep(wait)
            request = urllib.request.Request(url, headers={"User-Agent": AGENT, "Accept": "application/json"})
            try:
                with urllib.request.build_opener(Redirects()).open(request, timeout=25) as response:
                    raw = response.read((2 << 20) + 1)
            except urllib.error.HTTPError as error:
                if error.code in (429, 503):
                    retry = error.headers.get("Retry-After", "60")
                    seconds = max(60, int(retry)) if retry.isdigit() else 300
                    pacing.write_text(str(time.time() + seconds))
                raise
            finally:
                if not pacing.exists() or float(pacing.read_text()) < time.time():
                    pacing.write_text(str(time.time() + 1.1))
        if len(raw) > 2 << 20:
            raise ValueError("Artist metadata response exceeds 2 MiB")
        data = json.loads(raw)
        source = {"url": url, "raw_base64": base64.b64encode(raw).decode(), "fetched_at": dt.datetime.now(dt.timezone.utc).isoformat()}
        self.sources.append(source)
        cached.write_text(json.dumps({"epoch": time.time(), "source": source}))
        return data


def plain(value):
    return " ".join(html.unescape(re.sub(r"<[^>]*>", " ", value)).split())


def build_bundle(mbid, fetch):
    if not MBID.fullmatch(mbid):
        raise ValueError("Expected a reviewed MusicBrainz artist ID")
    artist = fetch("https://musicbrainz.org/ws/2/artist/" + mbid + "?inc=aliases%2Burl-rels&fmt=json")
    if artist.get("id") != mbid:
        raise ValueError("MusicBrainz response identity differs from reviewed artist ID")
    bundle = {"musicbrainz_id": mbid}
    wikidata = next((relation.get("url", {}).get("resource", "") for relation in artist.get("relations", []) if relation.get("type") == "wikidata"), "")
    match = re.fullmatch(r"https?://(?:www\.)?wikidata.org/wiki/(Q[0-9]+)", wikidata)
    if not match:
        return bundle
    entity = fetch("https://www.wikidata.org/wiki/Special:EntityData/" + match[1] + ".json").get("entities", {}).get(match[1], {})
    title = entity.get("sitelinks", {}).get("enwiki", {}).get("title")
    if title:
        summary = fetch("https://en.wikipedia.org/api/rest_v1/page/summary/" + urllib.parse.quote(title.replace(" ", "_"), safe=""))
        if summary.get("type") != "disambiguation" and summary.get("extract"):
            bundle["biography"] = {"text": summary["extract"][:16000], "source": "https://en.wikipedia.org/wiki/" + urllib.parse.quote(title.replace(" ", "_"), safe=""), "license": "CC BY-SA 4.0"}
    images = entity.get("claims", {}).get("P18", [])
    images = sorted(images, key=lambda claim: claim.get("rank") != "preferred")
    filename = next((claim.get("mainsnak", {}).get("datavalue", {}).get("value") for claim in images if claim.get("rank") != "deprecated" and isinstance(claim.get("mainsnak", {}).get("datavalue", {}).get("value"), str)), None)
    if filename:
        query = urllib.parse.urlencode({"action": "query", "format": "json", "titles": "File:" + filename, "prop": "imageinfo", "iiprop": "url|extmetadata", "iiurlwidth": "720"})
        pages = fetch("https://commons.wikimedia.org/w/api.php?" + query).get("query", {}).get("pages", {})
        info = next((page["imageinfo"][0] for page in pages.values() if page.get("imageinfo")), {})
        meta = info.get("extmetadata", {})
        credit = plain(meta.get("Artist", {}).get("value", ""))
        license_name = plain(meta.get("LicenseShortName", {}).get("value", ""))
        license_url = meta.get("LicenseUrl", {}).get("value", "").replace("http://", "https://", 1)
        image_url = info.get("thumburl", info.get("url", ""))
        if credit and license_name and license_url and image_url and info.get("descriptionurl"):
            bundle["photo"] = {"url": image_url, "source": info["descriptionurl"], "credit": credit, "license": license_name, "license_url": license_url}
    return bundle


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--mbid", required=True, help="Explicitly verified MusicBrainz artist ID; no name matching")
    parser.add_argument("--output", type=Path, required=True, help="Private operator bundle; do not commit")
    parser.add_argument("--cache", type=Path, default=Path.home() / ".cache/arkhiv/artist-enrichment")
    parser.add_argument("--refresh", action="store_true")
    args = parser.parse_args()
    os.umask(0o077)
    fetcher = Fetcher(args.cache, args.refresh)
    bundle = build_bundle(args.mbid, fetcher)
    bundle["sources"] = fetcher.sources
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(bundle, indent=2) + "\n")
    print("Retained artist bundle saved; review the identity and sources before import.")


if __name__ == "__main__":
    main()
