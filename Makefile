# Makefile – Zentrale Orchestrierung für Projekt-Automationen
# Referenz: copilot-instructions.md Abschnitt 3.1

.PHONY: help test test-backend test-backend-unit test-backend-integration test-backend-bench test-frontend lint lint-backend lint-frontend lint-ci adr-ref commit-lint release-check security-blockers scan scan-json pr-check release ci-local clean ensure-trivy push-ci pr-quality-gates-ci

# Standardwerte
TRIVY_FAIL_ON ?= HIGH,CRITICAL
TRIVY_JSON_REPORT ?= tmp/trivy-fs-report.json
VERSION ?=
BACKEND_DIR ?= backend
FRONTEND_DIR ?= frontend

help: ## Zeigt verfügbare Targets
	@echo "Projekt Automations – Make Targets"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

test: test-backend ## Führt alle Tests aus (Backend + Frontend)
	@echo "[make test] ✅ Alle Tests abgeschlossen"

test-backend: ## Führt alle Backend-Tests aus (Unit + Integration)
	@echo "[make test-backend] Führe Backend Tests aus..."
	@cd $(BACKEND_DIR) && go test -v -race -coverprofile=coverage.out ./...
	@echo "[make test-backend] ✅ Backend Tests erfolgreich"

test-backend-unit: ## Führt nur Backend Unit-Tests aus (ohne Integration)
	@echo "[make test-backend-unit] Führe Backend Unit-Tests aus..."
	@cd $(BACKEND_DIR) && go test -v -race -short ./...

test-backend-integration: ## Führt nur Backend Integration-Tests aus
	@echo "[make test-backend-integration] Führe Backend Integration-Tests aus..."
	@cd $(BACKEND_DIR) && go test -v -race -run Integration ./...

test-backend-bench: ## Führt Backend Benchmarks aus
	@echo "[make test-backend-bench] Führe Backend Benchmarks aus..."
	@cd $(BACKEND_DIR) && go test -bench=. -benchmem ./pkg/evedb/navigation/

test-frontend: ## Führt Frontend-Tests aus (Platzhalter)
	@echo "[make test-frontend] Keine Frontend-Tests konfiguriert – Platzhalter für zukünftige Implementierung"

lint: lint-backend ## Führt alle Linting-Checks aus (Backend + Frontend)
	@echo "[make lint] ✅ Alle Linting-Checks abgeschlossen"

lint-backend: ## Führt Backend Linting aus (gofmt, go vet)
	@echo "[make lint-backend] Prüfe Backend Code-Stil..."
	@cd $(BACKEND_DIR) && gofmt -l . | tee /dev/stderr | (! grep .)
	@cd $(BACKEND_DIR) && go vet ./...
	@echo "[make lint-backend] ✅ Backend Linting erfolgreich"

lint-frontend: ## Führt Frontend Linting aus (Platzhalter)
	@echo "[make lint-frontend] Kein Frontend Linting konfiguriert – Platzhalter für zukünftige Implementierung"

lint-ci: lint-backend ## Statische Analysen (CI-Modus)
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

pr-check: lint test scan ## Bündelt: lint + test + scan (für lokale PR-Vorbereitung)
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


