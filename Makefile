.PHONY: help build rebuild run dev stop rerun test snapshot neo4j-up neo4j-down neo4j-seed neo4j-logs clean

PORT ?= 8090

help:
	@echo "kg-viewer — 3D knowledge-graph viewer (Neo4j + Three.js)"
	@echo ""
	@echo "Targets:"
	@echo "  make run         Run the compiled server binary (port $(PORT))"
	@echo "  make dev         Run via go run (no separate build step)"
	@echo "  make build       Build binary to bin/kg-viewer"
	@echo "  make rebuild     Alias for build (recompile)"
	@echo "  make stop        Kill any running kg-viewer on port $(PORT)"
	@echo "  make rerun       Rebuild → stop → run"
	@echo "  make snapshot    Refresh snapshot.json from Neo4j (run while M3 is online)"
	@echo "  make test        Run unit tests"
	@echo "  make neo4j-up    Start local Neo4j (docker compose)"
	@echo "  make neo4j-seed  Seed sample graph data (idempotent)"
	@echo "  make neo4j-down  Stop Neo4j"
	@echo "  make neo4j-logs  Tail Neo4j logs"
	@echo "  make clean       Remove build artifacts"

build:
	mkdir -p bin && go build -o bin/kg-viewer ./cmd/server

rebuild: build

snapshot:
	go run ./cmd/snapshot

run: build
	@set -a; [ -f .env ] && . ./.env; set +a; KG_VIEWER_PORT=$(PORT) ./bin/kg-viewer

dev:
	@set -a; [ -f .env ] && . ./.env; set +a; KG_VIEWER_PORT=$(PORT) go run ./cmd/server

stop:
	@PID=$$(lsof -ti:$(PORT) 2>/dev/null); \
	if [ -n "$$PID" ]; then \
		echo "Stopping kg-viewer (PID $$PID) on port $(PORT)"; \
		kill $$PID 2>/dev/null; sleep 1; \
		kill -9 $$PID 2>/dev/null || true; \
	else echo "No process on port $(PORT)"; fi

rerun: rebuild stop run

test:
	go test -count=1 -race ./...

# --- Neo4j (local docker compose) ---
NEO4J_DEPLOY ?= deploy/neo4j

neo4j-up:
	docker compose -f $(NEO4J_DEPLOY)/docker-compose.yml up -d

neo4j-down:
	docker compose -f $(NEO4J_DEPLOY)/docker-compose.yml down

neo4j-seed:
	docker compose -f $(NEO4J_DEPLOY)/docker-compose.yml exec -T neo4j cypher-shell -u neo4j -p neo4j-healthdinner-2026 -f /seeds/seed.cypher

neo4j-logs:
	docker compose -f $(NEO4J_DEPLOY)/docker-compose.yml logs -f

clean:
	rm -rf bin/
