FROM python:3.13-slim AS dependencies
COPY --from=ghcr.io/astral-sh/uv:0.12.22 /uv /usr/local/bin/uv
WORKDIR /app
COPY pyproject.toml uv.lock ./
RUN uv sync --frozen --no-dev && useradd --uid 1000 --create-home curation \
    && mkdir /app/data && chown curation:curation /app/data
COPY arkhiv_curation ./arkhiv_curation
COPY LICENSE SOURCE.json ./

FROM dependencies AS verify
COPY tests ./tests
RUN uv sync --frozen && uv run --frozen pytest -q \
    && uv run --frozen ruff check . && uv run --frozen ruff format --check .

FROM dependencies AS runtime
USER curation
ENV ARKHIV_CURATION_DB=/app/data/curation.sqlite
EXPOSE 18764
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s CMD ["/app/.venv/bin/python", "-c", "import json,urllib.request; r=json.load(urllib.request.urlopen('http://127.0.0.1:18764/api/health')); assert r['worker_alive']"]
CMD ["/app/.venv/bin/python", "-m", "arkhiv_curation", "--host", "0.0.0.0"]
