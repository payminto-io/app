.PHONY: dev dev-down backend frontend checkout landing mcp migrate test lint smoke-local

dev:
	docker compose -f docker-compose.dev.yml up --build

dev-down:
	docker compose -f docker-compose.dev.yml down -v

backend:
	cd backend && set -a && . ./.env && set +a && go run cmd/server/main.go

frontend:
	cd frontend && npm run dev -- -p 3003

checkout:
	cd checkout && PAYMINTO_API_URL=http://localhost:8090/api/v1 npm run dev

landing:
	cd landing && npm run dev -- -p 3001

mcp:
	cd mcp-server && npm run dev

migrate-up:
	cd backend && go run cmd/migrate/main.go up

migrate-down:
	cd backend && go run cmd/migrate/main.go down

test:
	cd backend && go test ./...
	cd frontend && npm test
	cd contracts && forge test

lint:
	cd backend && golangci-lint run
	cd frontend && npm run lint

smoke-local:
	./scripts/smoke-local.sh
