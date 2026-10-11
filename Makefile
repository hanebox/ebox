.PHONY: dev build test lint clean migrate

dev:
	@scripts/dev.sh

build:
	@scripts/build.sh

migrate:
	cd backend && go run . artisan migrate

test:
	cd backend && go test ./...
	cd web && bun run test

lint:
	cd backend && go vet ./...
	cd web && bun run lint && bun run typecheck

clean:
	rm -rf dist
	rm -rf backend/web/dist
	cd web && rm -rf dist node_modules
