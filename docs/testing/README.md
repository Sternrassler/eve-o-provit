# Testing Guide

## Test-Typen

### Unit Tests (Backend)

```bash
make test-be-unit        # Go unit tests
make test-be-coverage    # Mit Coverage Report
```

### Integration Tests (Backend)

```bash
make test-migrations     # Database Migrations (Testcontainers)
make test-all           # Alle Backend Tests
```

Siehe [migrations.md](migrations.md) für Details zu Migration Tests.

### E2E Tests (Frontend)

Playwright suite with three projects (`public` / `setup` / `auth`). The `public`
project needs no login; the `auth` project reuses a one-time captured EVE SSO
session. Full guide:

→ [`frontend/tests/e2e/README.md`](../../frontend/tests/e2e/README.md)

```bash
cd frontend
npx playwright test --project=public   # no login (CI-capable)
npx playwright test                     # full suite (needs a captured session)
```

Coverage: API integration, public page smoke + login-UI, logged-out trading
policy, and the authenticated character/trading-workflow/logout flows.

## Debugging

**Backend Tests:**

```bash
go test -v ./internal/services/... -run TestSpecific
```

**E2E Tests:**

```bash
cd frontend
npx playwright test --project=public --debug
npx playwright show-trace test-results/*/trace.zip
```

## CI Integration

Tests laufen automatisch in GitHub Actions:

- ✅ Unit Tests (Backend)
- ✅ Linting (Go, TypeScript)
- ✅ Security Scans (CodeQL, Gitleaks)
- E2E `public` project is CI-capable (no login); `auth` runs locally/nightly

## Weiterführende Docs

- [migrations.md](migrations.md) - Migration Testing mit Testcontainers
- [frontend/tests/e2e/README.md](../../frontend/tests/e2e/README.md) - E2E Setup & Workflows
- [Playwright Docs](https://playwright.dev/docs/intro)
