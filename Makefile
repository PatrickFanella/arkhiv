SHELL := /usr/bin/env bash
.DEFAULT_GOAL := help

.PHONY: help verify deps fmt-check typecheck lint test build compose-config open-pilot-check

help:
	@awk 'BEGIN {FS = ":.*##"; printf "Commands:\n"} /^[a-zA-Z0-9_-]+:.*##/ {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

deps:
	pnpm --dir web install --frozen-lockfile

verify: deps fmt-check typecheck lint test build compose-config open-pilot-check ## Run all checks

fmt:
	@find backend -name '*.go' -not -path '*/vendor/*' -print0 | xargs -0 -r gofmt -w

fmt-check:
	@test -z "$$(find backend -name '*.go' -not -path '*/vendor/*' -print0 | xargs -0 -r gofmt -l)" || (echo 'Run make fmt'; exit 1)

typecheck:
	pnpm --dir web run typecheck

lint:
	cd backend && go vet ./...
	pnpm --dir web run lint

test:
	cd backend && go test ./...
	pnpm --dir web run test

build:
	cd backend && go build -o bin/arkhiv ./cmd/app
	pnpm --dir web run build

compose-config:
	docker compose config --quiet

open-pilot-check:
	@test -f .gitea/ISSUE_TEMPLATE/open-pilot-task.yaml
	@grep -q 'Test Command' .gitea/ISSUE_TEMPLATE/open-pilot-task.yaml

.PHONY: fmt-check typecheck dev-api dev-web
dev-api: ## Start the Go service
	cd backend && go run ./cmd/app

dev-web: ## Start Vite with the API proxy
	pnpm --dir web dev
