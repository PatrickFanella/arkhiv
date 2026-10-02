FROM golang:1.26.6-alpine AS build
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN go build -o /out/api ./cmd/app && go build -o /out/arkhiv ./cmd/arkhiv
FROM alpine:3.22
RUN apk add --no-cache ca-certificates poppler-utils postgresql17-client util-linux
WORKDIR /app
COPY --from=build /out/ /app/
COPY migrations/ /app/migrations/
COPY collections/ /app/collections/
ENV MIGRATIONS_DIR=/app/migrations ARKHIV_ARCHIVE=/data/archive
EXPOSE 8080
CMD ["/app/api"]
