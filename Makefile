.PHONY: help build build-prod rebuild run dev stop rerun test deployprod deployprod-status demo-url snapshot neo4j-up neo4j-down neo4j-seed neo4j-logs clean

PORT ?= 8090
PROD_SSH ?= prod
PROD_IP ?= 82.156.88.47
PROD_DIR ?= /opt/kg-viewer
PROD_SERVICE ?= kg-viewer
PROD_PORT ?= 8090
PROD_USER ?= kgviewer
PROD_ENV ?= .env.prod
PROD_BIN ?= bin/kg-viewer-linux-amd64
DEMO_HOURS ?= 72

help:
	@echo "kg-viewer — 3D knowledge-graph viewer (Neo4j + Three.js)"
	@echo ""
	@echo "Targets:"
	@echo "  make run         Run the compiled server binary (port $(PORT))"
	@echo "  make dev         Run via go run (no separate build step)"
	@echo "  make build       Build binary to bin/kg-viewer"
	@echo "  make deployprod  Cross-build, upload to prod (ssh prod), run on port 8090"
	@echo "  make deployprod-status  Show prod service status and listening port"
	@echo "  make demo-url    Print a time-limited demo URL (needs DEMO_SECRET in .env.prod)"
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

# Linux/amd64 release binary for the production server (ssh prod).
build-prod:
	mkdir -p bin && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o $(PROD_BIN) ./cmd/server

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

# Repeatable production deploy: ssh prod, no domain, direct port $(PROD_PORT).
deployprod: build-prod
	ssh $(PROD_SSH) 'install -d -m 0755 $(PROD_DIR); if ! id -u $(PROD_USER) >/dev/null 2>&1; then useradd --system --no-create-home --home $(PROD_DIR) --shell /usr/sbin/nologin $(PROD_USER); fi'
	@if [ -f $(PROD_ENV) ]; then \
		echo ">> Uploading prod env from $(PROD_ENV)"; \
		rsync -az $(PROD_ENV) $(PROD_SSH):$(PROD_DIR)/.env; \
	else \
		echo ">> $(PROD_ENV) not found, keeping remote .env (embedded snapshot fallback)"; \
	fi
	rsync -az $(PROD_BIN) $(PROD_SSH):$(PROD_DIR)/kg-viewer
	rsync -az deploy/kg-viewer.service $(PROD_SSH):/etc/systemd/system/$(PROD_SERVICE).service
	ssh $(PROD_SSH) 'chmod 0755 $(PROD_DIR)/kg-viewer; chown -R $(PROD_USER):$(PROD_USER) $(PROD_DIR); systemctl daemon-reload; systemctl enable $(PROD_SERVICE) >/dev/null 2>&1; systemctl restart $(PROD_SERVICE); systemctl is-active $(PROD_SERVICE)'
	ssh $(PROD_SSH) 'curl -fsS --max-time 5 http://127.0.0.1:$(PROD_PORT)/kg-viewer -o /dev/null && ss -ltn | grep -q ":$(PROD_PORT) "'
	@echo "Deployed: http://<prod-ip>:$(PROD_PORT)/kg-viewer"

deployprod-status:
	ssh $(PROD_SSH) 'systemctl status $(PROD_SERVICE) --no-pager -l | sed -n "1,20p"; ss -ltn | grep ":$(PROD_PORT) " || true'

demo-url:
	@set -a; [ -f $(PROD_ENV) ] && . ./$(PROD_ENV); set +a; \
	if [ -z "$$DEMO_SECRET" ]; then \
		echo "DEMO_SECRET is not set in $(PROD_ENV)"; exit 1; \
	fi; \
	exp=$$(( $$(date +%s) + $$(DEMO_HOURS) * 3600 )); \
	sig=$$(printf '%s' "$$exp" | openssl dgst -sha256 -hmac "$$DEMO_SECRET" | awk '{print $$NF}'); \
	echo "Demo URL (valid for $(DEMO_HOURS)h):"; \
	echo "http://$(PROD_IP):$(PROD_PORT)/kg-viewer?token=$$exp.$$sig"

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
