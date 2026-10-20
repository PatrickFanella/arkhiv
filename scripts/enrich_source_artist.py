"""Build a retained artist bundle from an explicitly verified identity.

The identity is a reviewed MusicBrainz ID, the Internet Archive collection that
holds the artist's recordings, or both. No name search, audio requests, or
browser-triggered provider calls. The photograph's bytes are included so the
engine can store its own copy. Import the result with vhness
import-source-artist and explicit operator review notes.
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
HOSTS = {"musicbrainz.org", "www.wikidata.org", "en.wikipedia.org", "commons.wikimedia.org", "archive.org"}
IMAGE_HOSTS = {"upload.wikimedia.org", "archive.org"}
# archive.org download links redirect to the storage node holding the item.
ARCHIVE_NODE = re.compile(r"[a-z0-9-]+\.(?:[a-z0-9-]+\.)?archive\.org")
ARCHIVE_ID = re.compile(r"[A-Za-z0-9][A-Za-z0-9_.-]{0,99}")
MAX_IMAGE = 12 << 20
DESCRIPTION_LICENSE = "Internet Archive collection description"
IMAGE_LICENSE = "Internet Archive collection image"
# Live Music Archive collection pages often carry only this generated text.
BOILERPLATE = [re.compile(pattern, re.I) for pattern in (
    r"Download free music(?: from [^.]*)?\.?",
    r"(?:Free|Trade-friendly) [^.]{0,200}?concerts?\.",
    r"(?:Trade-friendly )?(?:Live Music Archive )?concerts (?:in the Live Music Archive )?are available for download and(?:/or)? streaming in formats including [^.]*\.",
    r"Limited Flag:.*$",
    r"Stream ?only\.?$",
)]
# Paragraphs about taping, trading or archive permission are not biography.
NOT_BIOGRAPHY = re.compile(r"(?i)\b(?:permission|tap(?:e|es|ed|er|ers|ing)\b|taping|soundboard|non-?commercial|trad(?:e|ing)\b|audience record|recording policy|live music archive|\bLMA\b|etree|archive\.org|flac|mp3|shn\b)")
COMMONS_UPLOAD = re.compile(r"https://upload\.wikimedia\.org/wikipedia/commons/(?:thumb/)?[0-9a-f]/[0-9a-f]{2}/([^/?#]+)")


class Redirects(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        validate_url(newurl)
        return super().redirect_request(req, fp, code, msg, headers, newurl)


def validate_url(url, hosts=HOSTS, nodes=False):
    parsed = urllib.parse.urlsplit(url)
    host_ok = parsed.hostname in hosts or (nodes and ARCHIVE_NODE.fullmatch(parsed.hostname or ""))
    if parsed.scheme != "https" or not host_ok or parsed.username or parsed.port:
        raise ValueError("Unexpected artist metadata endpoint")


class ImageRedirects(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        validate_url(newurl, IMAGE_HOSTS, nodes=True)
        return super().redirect_request(req, fp, code, msg, headers, newurl)


class Fetcher:
    def __init__(self, cache, refresh=False):
        self.cache = Path(cache)
        self.cache.mkdir(parents=True, exist_ok=True)
        self.refresh = refresh
        self.sources = []

    def _request(self, url, opener, accept, limit):
        # Serialize outbound calls per provider across invocations on this
        # host. MusicBrainz stays slower than its one-call/second limit.
        host = urllib.parse.urlsplit(url).hostname
        interval = 1.1 if host == "musicbrainz.org" else 0.4
        with (self.cache / "provider.lock").open("a+") as lock:
            fcntl.flock(lock, fcntl.LOCK_EX)
            pacing = self.cache / ("next-request" if host == "musicbrainz.org" else "next-request-" + host)
            next_request = float(pacing.read_text()) if pacing.exists() else 0
            wait = next_request - time.time()
            if wait > 2:
                raise RuntimeError(f"Provider cooldown remains active for {wait:.0f} seconds")
            if wait > 0:
                time.sleep(wait)
            request = urllib.request.Request(url, headers={"User-Agent": AGENT, "Accept": accept})
            try:
                with opener.open(request, timeout=25) as response:
                    return response.read(limit + 1)
            except urllib.error.HTTPError as error:
                if error.code in (429, 503):
                    retry = error.headers.get("Retry-After", "60")
                    seconds = max(60, int(retry)) if retry.isdigit() else 300
                    pacing.write_text(str(time.time() + seconds))
                raise
            finally:
                if not pacing.exists() or float(pacing.read_text()) < time.time():
                    pacing.write_text(str(time.time() + interval))

    def image(self, url):
        """Return photograph bytes. They go into the bundle, not the sources."""
        validate_url(url, IMAGE_HOSTS)
        cached = self.cache / (hashlib.sha256(url.encode()).hexdigest() + ".image")
        if not self.refresh and cached.exists() and time.time() - cached.stat().st_mtime < 7 * 86400:
            return cached.read_bytes()
        raw = self._request(url, urllib.request.build_opener(ImageRedirects()), "image/jpeg,image/png,image/gif", MAX_IMAGE)
        if len(raw) > MAX_IMAGE:
            raise ValueError("Artist image exceeds 12 MiB")
        cached.write_bytes(raw)
        return raw

    def __call__(self, url):
        validate_url(url)
        key = hashlib.sha256(url.encode()).hexdigest()
        cached = self.cache / (key + ".json")
        if not self.refresh and cached.exists():
            saved = json.loads(cached.read_text())
            if time.time() - saved["epoch"] < 7 * 86400:
                self.sources.append(saved["source"])
                return json.loads(base64.b64decode(saved["source"]["raw_base64"]))
        raw = self._request(url, urllib.request.build_opener(Redirects()), "application/json", 2 << 20)
        if len(raw) > 2 << 20:
            raise ValueError("Artist metadata response exceeds 2 MiB")
        data = json.loads(raw)
        source = {"url": url, "raw_base64": base64.b64encode(raw).decode(), "fetched_at": dt.datetime.now(dt.timezone.utc).isoformat()}
        self.sources.append(source)
        cached.write_text(json.dumps({"epoch": time.time(), "source": source}))
        return data


def plain(value):
    return " ".join(html.unescape(re.sub(r"<[^>]*>", " ", value)).split())


def archive_description(value, title):
    """Collection page text without generated download notices, or None."""
    if isinstance(value, list):
        value = "\n\n".join(str(part) for part in value)
    text = re.sub(r"(?i)<br\s*/?>|</p>|</div>|\r?\n", "\n", str(value or ""))
    paragraphs = []
    for block in re.split(r"\n\s*\n|\n", html.unescape(re.sub(r"<[^>]*>", " ", text))):
        block = " ".join(word for word in block.split() if word != "rn")
        for pattern in BOILERPLATE:
            block = pattern.sub("", block)
        block = " ".join(block.split())
        if not block or block.casefold() == title.casefold():
            continue
        # Hard-wrapped text: continue a paragraph that stopped mid-sentence.
        if paragraphs and not re.search(r"[.!?:\"”’')\]]$", paragraphs[-1]):
            paragraphs[-1] += " " + block
        else:
            paragraphs.append(block)
    paragraphs = [block for block in paragraphs if not NOT_BIOGRAPHY.search(block) and not re.fullmatch(r"\S+", block)]
    if paragraphs:
        # Drop a repeated name or "BIO" label before the text itself.
        name = re.escape(title)
        first = re.sub(rf"(?i)^{name}\s+(?={name}\b)", "", paragraphs[0])
        paragraphs[0] = re.sub(rf"(?i)^(?:{name}\s+)?bio(?:graphy)?\b[\s:.-]*", "", first) or first
    text = "\n\n".join(paragraphs).strip()
    return text[:16000] if len(text.split()) >= 25 else None


def add_archive(bundle, collection, fetch, fetch_image, use_image=True, use_description=True):
    if not ARCHIVE_ID.fullmatch(collection):
        raise ValueError("Expected an Internet Archive collection identifier")
    response = fetch("https://archive.org/metadata/" + collection)
    meta = response.get("metadata", {})
    parents = meta.get("collection", [])
    parents = [parents] if isinstance(parents, str) else parents
    if meta.get("identifier") != collection or meta.get("mediatype") != "collection" or "etree" not in parents or not meta.get("title"):
        raise ValueError("Archive response is not the reviewed Live Music Archive collection")
    bundle["archive_collection"] = collection
    page = "https://archive.org/details/" + collection
    if use_description and "biography" not in bundle:
        text = archive_description(meta.get("description"), str(meta["title"]))
        if text:
            bundle["biography"] = {"text": text, "source": page, "license": DESCRIPTION_LICENSE}
    if use_image and "photo" not in bundle and fetch_image:
        name = next((f["name"] for f in response.get("files", []) if f.get("format") == "Item Image" and re.fullmatch(r"[^/]+\.(?:jpe?g|png|gif)", f.get("name", ""), re.I)), None)
        if name:
            url = page.replace("/details/", "/download/") + "/" + urllib.parse.quote(name)
            data = fetch_image(url)
            bundle["photo"] = {"url": url, "source": page, "credit": str(meta["title"])[:1900] + " collection, Live Music Archive", "license": IMAGE_LICENSE,
                               "license_url": "", "data_url": url, "data_base64": base64.b64encode(data).decode()}
    return bundle


def build_bundle(mbid, fetch, fetch_image=None):
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
    summary = {}
    if title:
        summary = fetch("https://en.wikipedia.org/api/rest_v1/page/summary/" + urllib.parse.quote(title.replace(" ", "_"), safe=""))
        if summary.get("type") != "disambiguation" and summary.get("extract"):
            bundle["biography"] = {"text": summary["extract"][:16000], "source": "https://en.wikipedia.org/wiki/" + urllib.parse.quote(title.replace(" ", "_"), safe=""), "license": "CC BY-SA 4.0"}
    images = entity.get("claims", {}).get("P18", [])
    images = sorted(images, key=lambda claim: claim.get("rank") != "preferred")
    filename = next((claim.get("mainsnak", {}).get("datavalue", {}).get("value") for claim in images if claim.get("rank") != "deprecated" and isinstance(claim.get("mainsnak", {}).get("datavalue", {}).get("value"), str)), None)
    if not filename and summary.get("type") != "disambiguation":
        # Without a Wikidata image, use the article's lead image only when it is
        # a Commons file; locally hosted (often non-free) images are skipped.
        lead = (summary.get("originalimage") or {}).get("source", "")
        match = COMMONS_UPLOAD.match(lead)
        filename = urllib.parse.unquote(match[1]) if match else None
    if filename:
        query = urllib.parse.urlencode({"action": "query", "format": "json", "titles": "File:" + filename, "prop": "imageinfo", "iiprop": "url|extmetadata", "iiurlwidth": "1280"})
        pages = fetch("https://commons.wikimedia.org/w/api.php?" + query).get("query", {}).get("pages", {})
        info = next((page["imageinfo"][0] for page in pages.values() if page.get("imageinfo")), {})
        meta = info.get("extmetadata", {})
        credit = plain(meta.get("Artist", {}).get("value", ""))
        license_name = plain(meta.get("LicenseShortName", {}).get("value", ""))
        license_url = meta.get("LicenseUrl", {}).get("value", "").replace("http://", "https://", 1)
        image_url = info.get("thumburl", info.get("url", ""))
        description = info.get("descriptionurl", "")
        if urllib.parse.urlsplit(image_url).hostname != "upload.wikimedia.org":
            image_url = info.get("url", "")
        # Public-domain files have no license URL, and GFDL/FAL link other hosts;
        # the file page's licensing section states the terms for both.
        if description and urllib.parse.urlsplit(license_url).hostname not in ("creativecommons.org", "commons.wikimedia.org"):
            license_url = description + "#Licensing"
        if credit and license_name and urllib.parse.urlsplit(image_url).hostname == "upload.wikimedia.org" and description.startswith("https://commons.wikimedia.org/"):
            bundle["photo"] = {"url": image_url, "source": description, "credit": credit[:2000], "license": license_name[:200], "license_url": license_url}
            if fetch_image:
                bundle["photo"]["data_url"] = image_url
                bundle["photo"]["data_base64"] = base64.b64encode(fetch_image(image_url)).decode()
    return bundle


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--mbid", help="Explicitly verified MusicBrainz artist ID; no name matching")
    parser.add_argument("--archive-collection", help="Live Music Archive collection holding the artist's recordings")
    parser.add_argument("--no-archive-image", action="store_true", help="Skip the collection image, for example a shared placeholder")
    parser.add_argument("--no-archive-description", action="store_true", help="Skip the collection page text")
    parser.add_argument("--output", type=Path, required=True, help="Private operator bundle; do not commit")
    parser.add_argument("--cache", type=Path, default=Path.home() / ".cache/arkhiv/artist-enrichment")
    parser.add_argument("--refresh", action="store_true")
    args = parser.parse_args()
    os.umask(0o077)
    if not args.mbid and not args.archive_collection:
        parser.error("--mbid or --archive-collection is required")
    fetcher = Fetcher(args.cache, args.refresh)
    bundle = build_bundle(args.mbid, fetcher, fetcher.image) if args.mbid else {}
    if args.archive_collection:
        add_archive(bundle, args.archive_collection, fetcher, fetcher.image, not args.no_archive_image, not args.no_archive_description)
    bundle["sources"] = fetcher.sources
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(bundle, indent=2) + "\n")
    print("Retained artist bundle saved; review the identity and sources before import.")


if __name__ == "__main__":
    main()
