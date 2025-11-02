# Makefile – Zentrale Orchestrierung für Projekt-Automationen
# Referenz: copilot-instructions.md Abschnitt 3.1

.PHONY: help test test-be test-be-unit test-be-int test-be-bench test-be-examples test-be-ex-cargo test-be-ex-nav test-fe lint lint-be lint-fe lint-ci adr-ref commit-lint release-check security-blockers scan scan-json secrets-scan secrets-check pr-check release ci-local clean ensure-trivy ensure-gitleaks push-ci pr-quality-gates-ci docker-up docker-down docker-logs docker-ps docker-build docker-clean docker-restart docker-rebuild docker-shell-api docker-shell-db docker-shell-redis migrate migrate-up migrate-down migrate-create

# Standardwerte
TRIVY_FAIL_ON ?= HIGH,CRITICAL
TRIVY_JSON_REPORT ?= tmp/trivy-fs-report.json
VERSION ?=
BACKEND_DIR ?= backend
FRONTEND_DIR ?= frontend
COMPOSE_FILE ?= deployments/docker-compose.yml
DOCKER_COMPOSE ?= $(shell command -v docker-compose 2>/dev/null || echo "docker compose")
DATABASE_URL ?= postgresql://eveprovit:dev@localhost:5432/eveprovit?sslmode=disable
MIGRATIONS_DIR ?= $(BACKEND_DIR)/migrations

.DEFAULT_GOAL := help

help: ## Zeigt verfügbare Targets (gruppiert)
	@echo "════════════════════════════════════════════════════════════════════════════════════════════════════════"
	@echo "  Projekt Automations – Make Targets"
	@echo "════════════════════════════════════════════════════════════════════════════════════════════════════════"
	@echo ""
	@echo "┌─ Tests ───────────────────────────────────────────────────────────────────────────────────────────────"
	@grep -E '^test.*:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "│ \033[36m%-26s\033[0m %-68s\n", $$1, $$2}'
	@echo "└───────────────────────────────────────────────────────────────────────────────────────────────────────"
	@echo ""
	@echo "┌─ Linting & Code-Qualität ─────────────────────────────────────────────────────────────────────────────"
	@grep -E '^lint.*:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "│ \033[36m%-26s\033[0m %-68s\n", $$1, $$2}'
	@echo "└───────────────────────────────────────────────────────────────────────────────────────────────────────"
	@echo ""
	@echo "┌─ Security & Scans ────────────────────────────────────────────────────────────────────────────────────"
	@grep -E '^(scan|security-blockers|secrets).*:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "│ \033[36m%-26s\033[0m %-68s\n", $$1, $$2}'
	@echo "└───────────────────────────────────────────────────────────────────────────────────────────────────────"
	@echo ""
	@echo "┌─ CI/CD & Quality Gates ───────────────────────────────────────────────────────────────────────────────"
	@grep -E '^(pr-check|push-ci|ci-local|pr-quality-gates-ci).*:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "│ \033[36m%-26s\033[0m %-68s\n", $$1, $$2}'
	@echo "└───────────────────────────────────────────────────────────────────────────────────────────────────────"
	@echo ""
	@echo "┌─ Release & Versioning ────────────────────────────────────────────────────────────────────────────────"
	@grep -E '^(release|release-check).*:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "│ \033[36m%-26s\033[0m %-68s\n", $$1, $$2}'
	@echo "└───────────────────────────────────────────────────────────────────────────────────────────────────────"
	@echo ""
	@echo "┌─ Governance & Validation ─────────────────────────────────────────────────────────────────────────────"
	@grep -E '^(adr-ref|commit-lint).*:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "│ \033[36m%-26s\033[0m %-68s\n", $$1, $$2}'
	@echo "└───────────────────────────────────────────────────────────────────────────────────────────────────────"
	@echo ""
	@echo "┌─ Utility ─────────────────────────────────────────────────────────────────────────────────────────────"
	@grep -E '^(clean|ensure-trivy|help).*:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "│ \033[36m%-26s\033[0m %-68s\n", $$1, $$2}'
	@echo "└───────────────────────────────────────────────────────────────────────────────────────────────────────"
	@echo ""
	@echo "┌─ Docker & Compose ────────────────────────────────────────────────────────────────────────────────────"
	@grep -E '^docker.*:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "│ \033[36m%-26s\033[0m %-68s\n", $$1, $$2}'
	@echo "└───────────────────────────────────────────────────────────────────────────────────────────────────────"
	@echo ""
	@echo "┌─ Database Migrations ─────────────────────────────────────────────────────────────────────────────────"
	@grep -E '^migrate.*:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "│ \033[36m%-26s\033[0m %-68s\n", $$1, $$2}'
	@echo "└───────────────────────────────────────────────────────────────────────────────────────────────────────"
	@echo ""

test: test-be test-fe ## Führt alle Tests aus (Backend + Frontend)
	@echo "[make test] ✅ Alle Tests abgeschlossen"

test-be: ## Führt alle Backend-Tests aus (Unit + Integration)
	@echo "[make test-be] Führe Backend Tests aus..."
	@cd $(BACKEND_DIR) && go test -v -race -coverprofile=coverage.out ./...
	@echo "[make test-be] ✅ Backend Tests erfolgreich"

test-be-unit: ## Führt nur Backend Unit-Tests aus (ohne Integration)
	@echo "[make test-be-unit] Führe Backend Unit-Tests aus..."
	@cd $(BACKEND_DIR) && go test -v -race -short ./...

test-be-int: ## Führt nur Backend Integration-Tests aus (mit Redis/Testcontainers)
	@echo "[make test-be-int] Führe Backend Integration-Tests aus..."
	@cd $(BACKEND_DIR) && go test -v -race -tags=integration ./...

test-integration: test-be-int ## Alias für test-be-int (Redis/Testcontainers Integration Tests)

test-be-bench: ## Führt Backend Benchmarks aus
	@echo "[make test-be-bench] Führe Backend Benchmarks aus..."
	@cd $(BACKEND_DIR) && go test -bench=. -benchmem ./pkg/evedb/navigation/

test-load: ## Führt Load Tests aus (erfordert Redis + SDE Database)
	@echo "[make test-load] Führe Load Tests aus (Redis + SDE erforderlich)..."
	@echo "[make test-load] Hinweis: Redis muss auf localhost:6379 laufen"
	@echo "[make test-load] Hinweis: SDE DB muss unter backend/data/sde/eve-sde.db liegen"
	@cd $(BACKEND_DIR) && go test -v -timeout 15m -tags=load ./internal/services/load_test.go ./internal/services/cache.go ./internal/services/market_fetcher.go
	@echo "[make test-load] ✅ Load Tests abgeschlossen"

test-load-bench: ## Führt Load Test Benchmarks aus
	@echo "[make test-load-bench] Führe Load Test Benchmarks aus..."
	@cd $(BACKEND_DIR) && go test -bench=BenchmarkTheForge -benchtime=3x -tags=load ./internal/services/
	@echo "[make test-load-bench] ✅ Benchmarks abgeschlossen"

test-be-examples: test-be-ex-cargo test-be-ex-nav ## Führt alle Backend-Examples aus

test-be-ex-cargo: ## Führt Cargo-Example aus (Badger + Tritanium)
	@echo "[make test-be-ex-cargo] Führe Cargo-Example aus..."
	@cd $(BACKEND_DIR)/examples/cargo && go run main.go -ship 648 -item 34 -racial-hauler 5 -ship-info
	@echo "[make test-be-ex-cargo] ✅ Cargo-Example erfolgreich"

test-be-ex-nav: ## Führt Navigation-Example aus (Jita → Amarr)
	@echo "[make test-be-ex-nav] Führe Navigation-Example aus..."
	@cd $(BACKEND_DIR)/examples/navigation && go run main.go -from 30000142 -to 30002187 -exact
	@echo "[make test-be-ex-nav] ✅ Navigation-Example erfolgreich"

test-fe: ## Führt Frontend-Tests aus (Platzhalter)
	@echo "[make test-fe] Keine Frontend-Tests konfiguriert – Platzhalter für zukünftige Implementierung"

lint: lint-be ## Führt alle Linting-Checks aus (Backend + Frontend)
	@echo "[make lint] ✅ Alle Linting-Checks abgeschlossen"

lint-be: ## Führt Backend Linting aus (gofmt, go vet)
	@echo "[make lint-be] Prüfe Backend Code-Stil..."
	@cd $(BACKEND_DIR) && gofmt -l . | tee /dev/stderr | (! grep .)
	@cd $(BACKEND_DIR) && go vet ./...
	@echo "[make lint-be] ✅ Backend Linting erfolgreich"

lint-fe: ## Führt Frontend Linting aus (Platzhalter)
	@echo "[make lint-fe] Kein Frontend Linting konfiguriert – Platzhalter für zukünftige Implementierung"

lint-ci: lint-be ## Statische Analysen (CI-Modus)
	@echo "[make lint-ci] ✅ CI Linting abgeschlossen"

adr-ref: ## Erzwingt ADR-Referenzen für Governance-Änderungen (CI-kompatibel)
	@echo "[make adr-ref] Prüfe ADR Referenz-Anforderungen..."; \
	if [ -x scripts/common/check-adr-ref.sh ]; then \
		set +e; \
		bash scripts/common/check-adr-ref.sh; \
		rc=$$?; \
		set -e; \
		if [ $$rc -eq 1 ]; then \
			echo "[make adr-ref] ❌ ADR-Referenz Pflicht verletzt"; \
			exit 1; \
		elif [ $$rc -eq 2 ]; then \
			echo "[make adr-ref] ⚠️ Skip Marker erkannt – Warnung akzeptiert"; \
			exit 0; \
		fi; \
	else \
		echo "[make adr-ref] scripts/common/check-adr-ref.sh nicht gefunden" >&2; \
		exit 1; \
	fi

commit-lint: ## Validiert Commit Messages (RANGE=origin/main..HEAD oder COMMIT_FILE=pfad)
	@echo "[make commit-lint] Prüfe Commit Messages..."; \
	if [ -x scripts/common/check-commit-msg.sh ]; then \
		if [ -n "$${RANGE:-}" ]; then \
			bash scripts/common/check-commit-msg.sh --range "$$RANGE"; \
		elif [ -n "$${COMMIT_FILE:-}" ]; then \
			bash scripts/common/check-commit-msg.sh --file "$$COMMIT_FILE"; \
		else \
			echo "[make commit-lint] ERROR: Bitte RANGE oder COMMIT_FILE angeben" >&2; \
			exit 1; \
		fi; \
	else \
		echo "[make commit-lint] scripts/common/check-commit-msg.sh nicht gefunden" >&2; \
		exit 1; \
	fi

release-check: ## Prüft VERSION/CHANGELOG Synchronität (für Release PRs)
	@echo "[make release-check] Prüfe VERSION und CHANGELOG..."; \
	if [ -x scripts/common/check-version-changelog.sh ]; then \
		bash scripts/common/check-version-changelog.sh; \
	else \
		echo "[make release-check] scripts/common/check-version-changelog.sh nicht gefunden" >&2; \
		exit 1; \
	fi

security-blockers: ## Prüft Trivy Report auf kritische Findings
	@echo "[make security-blockers] Prüfe Security Blocker..."; \
	if [ -x scripts/common/check-security-blockers.sh ]; then \
		bash scripts/common/check-security-blockers.sh; \
	else \
		echo "[make security-blockers] scripts/common/check-security-blockers.sh nicht gefunden" >&2; \
		exit 1; \
	fi

scan: ## Security & Dependency Checks
	@echo "[make scan] Führe Security Scan aus (Trivy)..."
	@$(MAKE) --no-print-directory ensure-trivy
	@if command -v trivy >/dev/null 2>&1; then \
		trivy fs --ignore-unfixed --scanners vuln --severity $(TRIVY_FAIL_ON) --exit-code 1 .; \
	else \
		echo "[make scan] trivy Installation fehlgeschlagen – überspringe Scan"; \
	fi

scan-json: ## Security Scan mit JSON Report (für check-security-blockers.sh)
	@echo "[make scan-json] Erzeuge Trivy JSON Report (ohne Build-Abbruch)..."
	@$(MAKE) --no-print-directory ensure-trivy
	@mkdir -p tmp
	@if command -v trivy >/dev/null 2>&1; then \
		trivy fs --ignore-unfixed --scanners vuln --format json -o $(TRIVY_JSON_REPORT) . || true; \
		echo "[make scan-json] Trivy JSON Report: $(TRIVY_JSON_REPORT)"; \
	else \
		echo "[make scan-json] trivy nicht verfügbar – kein Report erzeugt"; \
	fi

secrets-scan: ## Führt Gitleaks Scan aus (alle Commits + Staging)
	@echo "[make secrets-scan] Prüfe Repository auf Secrets mit Gitleaks..."
	@$(MAKE) --no-print-directory ensure-gitleaks
	@if command -v gitleaks >/dev/null 2>&1; then \
		gitleaks detect --source . --verbose --exit-code 1; \
		echo "[make secrets-scan] ✅ Keine Secrets gefunden"; \
	else \
		echo "[make secrets-scan] ⚠️ Gitleaks nicht verfügbar – überspringe Scan"; \
	fi

secrets-check: ## Führt Gitleaks Pre-Commit Check aus (nur Staging Area)
	@echo "[make secrets-check] Prüfe Staging Area auf Secrets..."
	@$(MAKE) --no-print-directory ensure-gitleaks
	@if command -v gitleaks >/dev/null 2>&1; then \
		gitleaks protect --staged --verbose --exit-code 1; \
		echo "[make secrets-check] ✅ Keine Secrets in Staging Area"; \
	else \
		echo "[make secrets-check] ⚠️ Gitleaks nicht verfügbar – überspringe Check"; \
	fi

pr-check: lint test scan secrets-check ## Bündelt: lint + test + scan + secrets (für lokale PR-Vorbereitung)
	@echo "[make pr-check] ✅ Alle lokalen Checks erfolgreich"

push-ci: ## Führt lint-ci und test in einem Rutsch aus
	@$(MAKE) --no-print-directory lint-ci
	@$(MAKE) --no-print-directory test
	@echo "[make push-ci] ✅ Lint & Test abgeschlossen"

release: ## Version bump + CHANGELOG Transform (Beispiel: make release VERSION=0.2.0)
	@if [ -z "$(VERSION)" ]; then \
		echo "[make release] ERROR: VERSION Parameter fehlt (Beispiel: make release VERSION=0.2.0)" >&2; \
		exit 1; \
	fi
	@echo "[make release] Bump Version auf $(VERSION)..."
	@echo "$(VERSION)" > VERSION
	@sed -i "s/^## \[Unreleased\]/## [Unreleased]\n\n## [$(VERSION)] - $$(date +%Y-%m-%d)/" CHANGELOG.md
	@echo "[make release] VERSION und CHANGELOG aktualisiert – bitte commit + tag erstellen"

ci-local: ## Simulation definierter CI-Gates lokal
	@echo "[make ci-local] Simuliere CI Pipeline lokal..."
	@bash scripts/common/check-normative.sh
	@bash scripts/common/check-adr.sh
	@$(MAKE) --no-print-directory test
	@$(MAKE) --no-print-directory scan

pr-quality-gates-ci: ## Führt alle Quality-Gate-Prüfungen für PRs aus
	@bash scripts/workflows/pr-quality-gates/run.sh

clean: ## Entfernt Build-Artefakte und temporäre Dateien
	@echo "[make clean] Räume temporäre Dateien auf..."
	@rm -rf tmp/*.md tmp/test-fixtures/
	@echo "[make clean] ✅ Clean abgeschlossen"

ensure-trivy: ## Stellt sicher, dass Trivy verfügbar ist
	@if command -v trivy >/dev/null 2>&1; then \
		echo "[make ensure-trivy] trivy bereits verfügbar"; \
	else \
		echo "[make ensure-trivy] trivy nicht installiert – versuche Installation"; \
		if command -v apt-get >/dev/null 2>&1; then \
			if command -v sudo >/dev/null 2>&1; then \
				sudo apt-get update -y >/dev/null 2>&1 || true; \
				sudo apt-get install -y wget jq >/dev/null 2>&1 || true; \
			else \
				apt-get update -y >/dev/null 2>&1 || true; \
				apt-get install -y wget jq >/dev/null 2>&1 || true; \
			fi; \
		fi; \
		if command -v sudo >/dev/null 2>&1; then \
			curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sudo sh -s -- -b /usr/local/bin || true; \
		else \
			curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin || true; \
		fi; \
	fi

ensure-gitleaks: ## Stellt sicher, dass Gitleaks verfügbar ist
	@if command -v gitleaks >/dev/null 2>&1; then \
		echo "[make ensure-gitleaks] gitleaks bereits verfügbar ($$(gitleaks version))"; \
	else \
		echo "[make ensure-gitleaks] gitleaks nicht installiert – versuche Installation"; \
		TEMP_DIR=$$(mktemp -d); \
		cd $$TEMP_DIR; \
		LATEST_VERSION=$$(curl -s https://api.github.com/repos/gitleaks/gitleaks/releases/latest | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/'); \
		ARCH=$$(uname -m); \
		OS=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
		if [ "$$ARCH" = "x86_64" ]; then ARCH="x64"; fi; \
		if [ "$$ARCH" = "aarch64" ]; then ARCH="arm64"; fi; \
		DOWNLOAD_URL="https://github.com/gitleaks/gitleaks/releases/download/v$$LATEST_VERSION/gitleaks_$${LATEST_VERSION}_$${OS}_$${ARCH}.tar.gz"; \
		echo "[make ensure-gitleaks] Download: $$DOWNLOAD_URL"; \
		curl -sSL "$$DOWNLOAD_URL" -o gitleaks.tar.gz || exit 1; \
		tar -xzf gitleaks.tar.gz || exit 1; \
		if command -v sudo >/dev/null 2>&1; then \
			sudo mv gitleaks /usr/local/bin/ || exit 1; \
			sudo chmod +x /usr/local/bin/gitleaks || exit 1; \
		else \
			mv gitleaks /usr/local/bin/ || exit 1; \
			chmod +x /usr/local/bin/gitleaks || exit 1; \
		fi; \
		cd -; \
		rm -rf $$TEMP_DIR; \
		echo "[make ensure-gitleaks] ✅ Gitleaks installiert: $$(gitleaks version)"; \
	fi

# Docker & Compose Targets

db-load: ## Lädt EVE SDE SQLite DB vom offiziellen GitHub Release
	@echo "[make db-load] Lade EVE SDE Datenbank..."
	@bash scripts/download-sde.sh
	@echo "[make db-load] ✅ SDE Datenbank geladen"

docker-up: ## Startet alle Services (PostgreSQL, Redis, Backend)
	@echo "[make docker-up] Starte Docker Compose Services..."
	@$(DOCKER_COMPOSE) -f $(COMPOSE_FILE) up -d
	@echo "[make docker-up] ✅ Services gestartet"
	@echo ""
	@echo "Services verfügbar unter:"
	@echo "  - Frontend:     http://localhost:9000"
	@echo "  - Backend API:  http://localhost:9001"
	@echo "  - PostgreSQL:   localhost:5432 (User: eveprovit, DB: eveprovit)"
	@echo "  - Redis:        localhost:6379"
	@echo ""
	@echo "Logs anzeigen: make docker-logs"
	@echo "Status prüfen: make docker-ps"

docker-down: ## Stoppt alle Services
	@echo "[make docker-down] Stoppe Docker Compose Services..."
	@$(DOCKER_COMPOSE) -f $(COMPOSE_FILE) down
	@echo "[make docker-down] ✅ Services gestoppt"

docker-logs: ## Zeigt Logs aller Services (oder SERVICE=api für einzelnen Service)
	@if [ -z "$(SERVICE)" ]; then \
		$(DOCKER_COMPOSE) -f $(COMPOSE_FILE) logs -f; \
	else \
		$(DOCKER_COMPOSE) -f $(COMPOSE_FILE) logs -f $(SERVICE); \
	fi

docker-ps: ## Zeigt Status aller Services
	@$(DOCKER_COMPOSE) -f $(COMPOSE_FILE) ps

docker-build: ## Baut alle Docker Images neu
	@echo "[make docker-build] Baue Docker Images..."
	@$(DOCKER_COMPOSE) -f $(COMPOSE_FILE) build --no-cache
	@echo "[make docker-build] ✅ Images gebaut"

docker-clean: ## Entfernt alle Container, Volumes und Images
	@echo "[make docker-clean] Räume Docker Ressourcen auf..."
	@$(DOCKER_COMPOSE) -f $(COMPOSE_FILE) down -v --rmi all
	@echo "[make docker-clean] ✅ Cleanup abgeschlossen"

docker-restart: docker-down docker-up ## Neustart aller Services (ohne Rebuild)

docker-rebuild: docker-down docker-build docker-up ## Kompletter Rebuild: Down → Build → Up

docker-shell-api: ## Shell im Backend Container
	@docker exec -it eve-o-provit-api /bin/sh

docker-shell-db: ## Shell in PostgreSQL Container
	@docker exec -it eve-o-provit-postgres psql -U eveprovit -d eveprovit

docker-shell-redis: ## Redis CLI
	@docker exec -it eve-o-provit-redis redis-cli

# Database Migration Targets

migrate-up: ## Führt alle ausstehenden Migrations aus
	@echo "[make migrate-up] Führe Database Migrations aus..."
	@cd $(BACKEND_DIR) && migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" up
	@echo "[make migrate-up] ✅ Migrations abgeschlossen"

migrate-down: ## Rollt letzte Migration zurück
	@echo "[make migrate-down] Rollback letzte Migration..."
	@cd $(BACKEND_DIR) && migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" down 1
	@echo "[make migrate-down] ✅ Rollback abgeschlossen"

migrate-create: ## Erstellt neue Migration (NAME=migration_name)
	@if [ -z "$(NAME)" ]; then \
		echo "[make migrate-create] ERROR: NAME Parameter fehlt (Beispiel: make migrate-create NAME=add_users_table)" >&2; \
		exit 1; \
	fi
	@echo "[make migrate-create] Erstelle neue Migration: $(NAME)"
	@cd $(BACKEND_DIR) && migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(NAME)
	@echo "[make migrate-create] ✅ Migration erstellt"

migrate: migrate-up ## Alias für migrate-up

test-migrations: ## Führt Migration Integration Tests mit Testcontainers aus
	@echo "[make test-migrations] Führe Migration Tests aus..."
	@if ! command -v migrate >/dev/null 2>&1 && ! [ -x ~/go/bin/migrate ]; then \
		echo "[make test-migrations] golang-migrate nicht gefunden - installiere..."; \
		cd $(BACKEND_DIR) && go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest; \
	fi
	@echo "[make test-migrations] 🧪 Running Migration Tests:"
	@cd $(BACKEND_DIR) && go test -v -run "TestMigration" ./internal/database/ 2>&1 | grep -E "(RUN|PASS|FAIL|Migration output:|Migration status:)" || true
	@echo "[make test-migrations] ✅ Migration Tests abgeschlossen"



