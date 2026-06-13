# eve-o-provit

Full-Stack Trading Profit Optimizer mit drei Komponenten: Backend Go/Fiber
(`backend/`, go 1.25 / toolchain 1.26), Frontend Next.js 16 (`frontend/`) und
Flutter-Android-Client (`app/`, Galaxy-Tab-Ziel). Detail-Architektur,
API-Endpunkte, DB-Schema und OAuth2-Flow:
[→ ../docs/eve-o-provit.md](../docs/eve-o-provit.md).

## Layout

- `backend/cmd/` Entrypoints (`api`, `test-sde`), `backend/internal/` Services + Handler (database, esi, handlers, metrics, models, services, version), `backend/pkg/` shared, `backend/migrations/` + `backend/sql/` DB.
- `frontend/` Next.js 16.
- `app/` Flutter-Client — **eigenes Makefile**, nicht das Root-Makefile.

## Commands

`make help` listet alle Targets (gruppiert). Wichtigste:

- `make test` — Backend + Frontend.
- `make test-be-int` — Backend-Integrationstests; **erfordert Docker** (Redis via Testcontainers, Build-Tag `-tags=integration`).
- `make test-fe-e2e` — Playwright E2E (headless).
- `make lint` — Backend (`gofmt`-Check + `go vet`) + Frontend (ESLint).
- `make pr-check` — lokales PR-Gate: `lint + test + scan + secrets-check`.
- DB-Migrationen: `make migrate-up` (ausstehende ausführen) · `make migrate-create NAME=...` (neue anlegen) · `make test-migrations` (Migrations-Integrationstests via Testcontainers).
- Docker-Dev: `make docker-up` (Services + SDE, Image-Rebuild) · `make docker-logs` · `make docker-shell-api|db|redis`.
- Swagger/OpenAPI: `make swagger` regeneriert die Spec in `backend/docs` aus den swag-Annotationen.

### Flutter-App (`app/`)

Nicht über das Root-Makefile, sondern `cd app && make ...`:

- `make test` — Widget-/Unit-Tests · `make analyze` — `flutter analyze`.
- `make android-install` — Release-APK bauen + via `adb` aufs Gerät spielen (braucht `EVE_MOBILE_CLIENT_ID` aus dem Deploy-Env).

## Gotchas

- `make scan`/`secrets-check` brauchen **Trivy** bzw. **Gitleaks** (`make ensure-trivy` / `ensure-gitleaks` installieren sie bei Bedarf).
- **Swagger-Drift-Gate** in CI (`ci.yml`): führt `make swagger` aus und bricht via `git diff` auf `backend/docs` ab. Nach Änderungen an swag-Annotationen `make swagger` laufen lassen und das Ergebnis committen, sonst rote CI.
- Conventional Commits erzwungen via `.githooks` (`git config core.hooksPath .githooks`).
- Release: SemVer lebt nur in `CHANGELOG.md` + git-Tag (`vX.Y.Z`) — keine `VERSION`-Datei. `make release-check` prüft die CHANGELOG-Konsistenz; `make release VERSION=x.y.z` transformiert den `[Unreleased]`-Block.
- **Deploy ist tag-getrieben** (kein release-please): Push eines `v*`-Tags triggert `deploy.yml` (baut + pusht Docker-Images), nachgelagert läuft `smoke-test.yml`.
- Deployment via Docker Compose unter `deployments/` — Frontend :9000, Backend :9001, Postgres :5432, Redis :6379.
- Backend exponiert Prometheus-Metriken unter `/metrics` (Port 9001, Namespace `eveoprovit_`, nicht rate-limited).
