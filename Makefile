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

# Colima exposes Docker at a per-user socket; the reaper must mount the VM's own socket.
test-integration:
	cd backend && \
	if [ -z "$$DOCKER_HOST" ] && [ -S "$$HOME/.colima/default/docker.sock" ]; then \
		export DOCKER_HOST=unix://$$HOME/.colima/default/docker.sock TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock; \
	fi; \
	go test -tags=integration -count=1 -timeout 45m ./...

lint:
	cd backend && golangci-lint run
	cd frontend && npm run lint

smoke-local:
	./scripts/smoke-local.sh
