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

```bash
make test-e2e            # Headless
make fe-test-e2e-headed  # Browser sichtbar
make fe-test-e2e-ui      # Interactive UI Mode
```

**Setup:**

1. Environment setzen:
   ```bash
   # .env (Root)
   EVE_TEST_CHARACTER=YourCharacterName
   EVE_TEST_USERNAME=your-eve-email@example.com
   EVE_TEST_PASSWORD=your-eve-password
   ```

2. Services starten:
   ```bash
   make docker-up
   ```

3. Tests ausführen:
   ```bash
   make test-e2e
   ```

**Test Coverage:**

- ✅ Landing Page (9 tests)
- ✅ API Integration (13 tests)
- 🚧 Auth Flow (requires credentials)
- 🚧 Character Features (requires auth)

Siehe [e2e-quick-start.md](e2e-quick-start.md) für Details.

## Debugging

**Backend Tests:**

```bash
go test -v ./internal/services/... -run TestSpecific
```

**E2E Tests:**

```bash
cd frontend
npx playwright test tests/e2e/home.spec.ts --debug
npx playwright show-trace test-results/*/trace.zip
```

**Screenshots:**

```bash
ls -lh frontend/tests/screenshots/
```

## CI Integration

Tests laufen automatisch in GitHub Actions:

- ✅ Unit Tests (Backend)
- ✅ Linting (Go, TypeScript)
- ✅ Security Scans (CodeQL, Gitleaks)
- ⏸️ E2E Tests (deaktiviert bis Auth fertig)

## Weiterführende Docs

- [migrations.md](migrations.md) - Migration Testing mit Testcontainers
- [e2e-quick-start.md](e2e-quick-start.md) - E2E Setup & Workflows
- [Playwright Docs](https://playwright.dev/docs/intro)
