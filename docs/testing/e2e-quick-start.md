# E2E Test Quick Start Guide

## Setup (Einmalig)

1. **Dependencies installieren:**

   ```bash
   cd /home/ix/vscode/eve-o-provit
   make fe-install
   ```

2. **EVE Test Account Credentials setzen:**

   Erstelle `.env` im Root-Verzeichnis:

   ```bash
   # EVE Online Test Account (für auth.spec.ts)
   EVE_TEST_CHARACTER=YourCharacterName
   EVE_TEST_USERNAME=your-eve-email@example.com
   EVE_TEST_PASSWORD=your-eve-password
   ```

3. **Docker Services starten:**

   ```bash
   make docker-rebuild
   ```

## Tests ausführen

### Alle E2E Tests (Headless)

```bash
make test-e2e
# oder
cd frontend && npm run test:e2e
```

### Browser sichtbar (Debugging)

```bash
make fe-test-e2e-headed
# oder
cd frontend && npm run test:e2e:headed
```

### Interaktive UI (Beste Developer Experience)

```bash
make fe-test-e2e-ui
# oder
cd frontend && npm run test:e2e:ui
```

### Einzelne Test-Datei ausführen

```bash
cd frontend
npx playwright test tests/e2e/home.spec.ts
```

### Einzelnen Test ausführen (grep)

```bash
cd frontend
npx playwright test -g "Home page loads"
```

## Test-Dateien

```txt
tests/e2e/
├── home.spec.ts        # ✅ Implementiert (9 tests)
├── auth.spec.ts        # 🚧 Teilweise (needs credentials)
├── character.spec.ts   # 🚧 Teilweise (requires auth)
├── navigation.spec.ts  # ✅ Implementiert (8 tests)
├── cargo.spec.ts       # ✅ Implementiert (8 tests)
├── market.spec.ts      # ✅ Implementiert (8 tests)
└── api.spec.ts         # ✅ Implementiert (13 tests)
```

**Total:** ~54 Tests implementiert

## Debugging

### Screenshots anschauen

```bash
ls -lh frontend/tests/screenshots/
```

### Trace Viewer (bei Fehlern)

```bash
cd frontend
npx playwright show-trace test-results/*/trace.zip
```

### Browser Inspector (Pause auf Fehler)

```bash
cd frontend
npx playwright test --debug
```

## Typische Workflows

### Workflow 1: Feature entwickeln → Tests schreiben → Ausführen

```bash
# 1. Services starten
make docker-up

# 2. Frontend dev server (separates Terminal)
make fe-dev

# 3. E2E Tests im UI Mode (separates Terminal)
make fe-test-e2e-ui

# Dann: Tests interaktiv auswählen und debuggen
```

### Workflow 2: PR vorbereiten (alle Tests grün)

```bash
# Backend Tests
make test-all

# Linting
make lint

# E2E Tests
make test-e2e

# Alles OK → PR erstellen
```

### Workflow 3: EVE SSO Login debuggen

```bash
# 1. Credentials in .env setzen
# 2. auth.spec.ts skip entfernen
# 3. Headed mode mit Pause
cd frontend
npx playwright test auth.spec.ts --headed --debug

# Playwright Inspector öffnet sich
# → Step-by-step durch Login-Flow
```

## Bekannte Einschränkungen

### Auth Tests (auth.spec.ts, character.spec.ts)

- ⚠️ Benötigen gültige EVE Account Credentials
- ⚠️ EVE SSO Selektoren können sich ändern (CCP Updates)
- ⚠️ Aktuell `test.skip()` markiert bis Frontend Auth fertig
- ✅ Helper-Functions bereits implementiert (`tests/helpers/auth.ts`)

### Frontend Features noch nicht implementiert

- Navigation Calculator (UI fehlt noch)
- Cargo Calculator (UI fehlt noch)
- Market Analysis (UI fehlt noch)
- Character Page (UI fehlt noch)

→ Tests sind **spezifikativ** (definieren erwartetes Verhalten)
→ Tests schlagen fehl bis Features implementiert sind (TDD)

## CI Integration (geplant)

```yaml
# .github/workflows/e2e-tests.yml
name: E2E Tests
on: [pull_request]
jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: make docker-rebuild
      - run: make fe-install
      - run: make test-e2e
        env:
          EVE_TEST_CHARACTER: ${{ secrets.EVE_TEST_CHARACTER }}
          EVE_TEST_USERNAME: ${{ secrets.EVE_TEST_USERNAME }}
          EVE_TEST_PASSWORD: ${{ secrets.EVE_TEST_PASSWORD }}
```

## Nächste Schritte

1. **Frontend Features implementieren:**
   - Navigation Calculator UI
   - Cargo Calculator UI
   - Market Analysis UI
   - Character Page UI

2. **Auth Tests aktivieren:**
   - `.env` mit Test-Account setzen
   - `test.skip()` entfernen
   - EVE SSO Selektoren validieren

3. **CI Pipeline:**
   - GitHub Actions Workflow erstellen
   - Secrets in GitHub Repository setzen
   - Auto-Run bei PRs

4. **Test-Erweiterungen:**
   - Visual Regression Tests (Screenshots vergleichen)
   - Performance Tests (Lighthouse CI)
   - Accessibility Tests (axe-core)

## Hilfe & Debugging

### Test schlägt fehl - Was tun?

1. **Screenshot anschauen:**

   ```bash
   open frontend/tests/screenshots/<test-name>.png
   ```

2. **Browser sichtbar machen:**

   ```bash
   npx playwright test <test-name> --headed
   ```

3. **Trace anschauen:**

   ```bash
   npx playwright show-trace test-results/*/trace.zip
   ```

4. **Inspector nutzen:**

   ```bash
   npx playwright test <test-name> --debug
   ```

### Services laufen nicht

```bash
make docker-ps       # Status prüfen
make docker-logs     # Logs anschauen
make docker-restart  # Neustart
```

### Playwright Browser fehlt

```bash
cd frontend
npx playwright install chromium
```

## Ressourcen

- [Playwright Docs](https://playwright.dev/docs/intro)
- [Playwright Best Practices](https://playwright.dev/docs/best-practices)
- [Test Generator](https://playwright.dev/docs/codegen) - `npx playwright codegen`
- [VS Code Extension](https://marketplace.visualstudio.com/items?itemName=ms-playwright.playwright)
