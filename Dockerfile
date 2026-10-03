FROM node:24-bookworm-slim AS frontend
RUN npm install --global pnpm@10.33.0 --force
WORKDIR /src/web
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

FROM golang:1.26-bookworm AS backend
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -o /out/vhness ./cmd/vhness

FROM debian:bookworm-slim
RUN apt-get update && apt-get install --yes --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=backend /out/vhness /usr/local/bin/vhness
COPY --from=frontend /src/web/dist /app/web
COPY migrations/ /app/migrations/
ENV API_ADDR=0.0.0.0:8080 VHNESS_WEB=/app/web VHNESS_MIGRATIONS=/app/migrations VHNESS_MEDIA=/media
USER 1000:1000
EXPOSE 8080
HEALTHCHECK --interval=20s --timeout=5s --start-period=20s --retries=3 CMD ["vhness", "healthcheck"]
ENTRYPOINT ["vhness"]
CMD ["serve"]
