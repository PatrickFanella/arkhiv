.PHONY: verify test test-integration
verify:
	cd backend && test -z "$$(gofmt -l cmd internal)" && go vet ./... && go test -race ./... && go build ./cmd/vhness
	pnpm --dir web install --frozen-lockfile
	pnpm --dir web typecheck
	pnpm --dir web lint
	pnpm --dir web test
	pnpm --dir web build
test:
	cd backend && go test -race ./...
	pnpm --dir web test
test-integration:
	test -n "$(TEST_DATABASE_URL)"
	cd backend && go test -race ./internal/catalog ./internal/worker -count=1
