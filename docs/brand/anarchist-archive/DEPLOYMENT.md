# Anarchist collection identity

This is a named collection export for the retained Anarchist Archive frontend. Arkhiv’s default `web/index.html`, pack, public catalog and engine selection retain the Arkhiv identity. The narrow deployment applies `identity.json` to the current frontend HTML and copies this directory’s public assets. It does not replace the engine, documents, collection membership, rights metadata or source attribution.

After building or assembling the retained frontend, run from the repository root:

```sh
python3 web/scripts/apply_collection_identity.py --site-root web/dist --identity web/public/brand/anarchist-archive/identity.json
```

Run it once on a fresh artifact; it refuses an already-applied identity. The application’s existing collection/profile and API pins stay independent.
