"""Read-only combined-core client and retained curation evidence."""

import hashlib
import json
import re
from urllib.parse import quote, urlsplit

import httpx
from pydantic import BaseModel, ConfigDict, Field, field_validator

ID = r"[A-Za-z0-9][A-Za-z0-9._-]{0,199}"
ITEM_ID = r"(?:legacy-video:)?" + ID
SCHEMA = "arkhiv.curation.v1"
DENIED = {
    key: False
    for key in ("read", "playback", "download", "acquisition", "relay", "schedule", "public")
}


class CoreError(Exception):
    pass


class Representation(BaseModel):
    model_config = ConfigDict(extra="allow")
    id: str = Field(pattern="^" + ID + "$")
    media: str = Field(pattern="^(print|audio|video)$")
    source_item_id: str = Field(pattern="^" + ID + "$")
    snapshot_sha256: str = Field(pattern=r"^[0-9a-f]{64}$")
    file: str = Field(min_length=1, max_length=1000)
    extension: dict
    collections: list[str]

    @field_validator("file")
    @classmethod
    def safe_file(cls, value):
        if (
            value.startswith("/")
            or "\\" in value
            or any(part in {"", ".", ".."} for part in value.split("/"))
        ):
            raise ValueError("Invalid source file")
        return value

    @field_validator("extension")
    @classmethod
    def typed_extension(cls, value, info):
        media = info.data["media"]
        if media not in value or not isinstance(value[media], dict):
            raise ValueError("Missing media evidence")
        return value


class CatalogItem(BaseModel):
    model_config = ConfigDict(extra="allow")
    id: str = Field(pattern="^" + ITEM_ID + "$")
    title: str = Field(min_length=1, max_length=4000)
    description: str = ""
    date: str = ""
    creator: str = ""
    collections: list[str]
    representations: list[Representation] = Field(min_length=1, max_length=64)


class CoreClient:
    def __init__(self, base_url="", token="", transport=None):
        self.base_url = base_url.rstrip("/")
        parsed = urlsplit(self.base_url)
        if base_url and (
            parsed.scheme not in {"http", "https"}
            or not parsed.netloc
            or parsed.username
            or parsed.password
            or parsed.query
            or parsed.fragment
        ):
            raise ValueError(
                "ARKHIV_CURATION_CORE_URL must be an HTTP(S) API base URL without credentials or query parameters."
            )
        self.token = token
        self.transport = transport

    def get(self, path, params=None):
        if not self.base_url:
            raise CoreError(
                "The combined catalog has not been connected. Your saved searches still work."
            )
        try:
            with (
                httpx.Client(
                    timeout=10, follow_redirects=False, trust_env=False, transport=self.transport
                ) as client,
                client.stream(
                    "GET",
                    self.base_url + path,
                    params=params,
                    headers={"Authorization": "Bearer " + self.token} if self.token else {},
                ) as response,
            ):
                response.raise_for_status()
                chunks, size = [], 0
                for chunk in response.iter_bytes():
                    size += len(chunk)
                    if size > 8 * 1024 * 1024:
                        raise CoreError("The combined catalog response is too large.")
                    chunks.append(chunk)
                return json.loads(b"".join(chunks))
        except (httpx.HTTPError, ValueError) as exc:
            raise CoreError(
                "Could not read the combined catalog. Check its connection and access."
            ) from exc

    def browse(self, q="", media="", collection="", page=1):
        try:
            result = self.get(
                "/api/catalog/v1/items",
                {"q": q, "media": media, "collection": collection, "page": page},
            )
            items = [CatalogItem.model_validate(item).model_dump() for item in result["items"]]
            total = int(result["total"])
            if total < 0 or len(items) > 24 or result["page"] != page:
                raise ValueError("Invalid pagination")
            return {"items": items, "total": total, "page": page, "page_size": 24}
        except (KeyError, TypeError, ValueError) as exc:
            raise CoreError("The combined catalog returned incompatible item evidence.") from exc

    def item(self, item_id):
        if not re.fullmatch(ITEM_ID, item_id):
            raise CoreError("Invalid catalog identifier.")
        try:
            item = CatalogItem.model_validate(
                self.get("/api/catalog/v1/items/" + quote(item_id, safe=""))
            )
            if item.id != item_id:
                raise ValueError("Wrong item")
            return item.model_dump()
        except (TypeError, ValueError) as exc:
            raise CoreError("The combined catalog returned incompatible item evidence.") from exc


def evidence_hash(item):
    return hashlib.sha256(
        json.dumps(item, ensure_ascii=False, sort_keys=True, separators=(",", ":")).encode()
    ).hexdigest()
