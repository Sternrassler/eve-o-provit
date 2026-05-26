# eve-o-provit

Full-Stack Trading Profit Optimizer. Backend Go/Fiber (`backend/`, go 1.24),
Frontend Next.js 16 (`frontend/`). Detail-Architektur, API-Endpunkte, DB-Schema
und OAuth2-Flow: [→ ../docs/eve-o-provit.md](../docs/eve-o-provit.md).

## Commands

`make help` listet alle Targets (gruppiert). Wichtigste:

- `make test` — Backend + Frontend.
- `make test-be-int` — Backend-Integrationstests; **erfordert Docker** (Redis via Testcontainers, Build-Tag `-tags=integration`).
- `make test-fe-e2e` — Playwright E2E (headless).
- `make lint` — Backend (`gofmt`-Check + `go vet`) + Frontend (ESLint).
- `make pr-check` — lokales PR-Gate: `lint + test + scan + secrets-check`.

## Gotchas

- `make scan`/`secrets-check` brauchen **Trivy** bzw. **Gitleaks** (`make ensure-trivy` / `ensure-gitleaks` installieren sie bei Bedarf).
- Conventional Commits erzwungen via `.githooks` (`git config core.hooksPath .githooks`).
- Release: SemVer lebt nur in `CHANGELOG.md` + git-Tag (`vX.Y.Z`) — keine `VERSION`-Datei. `make release-check` prüft die CHANGELOG-Konsistenz; `make release VERSION=x.y.z` transformiert den `[Unreleased]`-Block.
- Deployment via Docker Compose unter `deployments/` — Frontend :9000, Backend :9001, Postgres :5432, Redis :6379.
