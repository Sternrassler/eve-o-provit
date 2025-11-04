# E2E Tests mit Playwright

End-to-End Tests für EVE-O-Provit Frontend mit echtem EVE SSO OAuth Flow.

## Voraussetzungen

1. **Docker Services laufen:**

   ```bash
   cd /home/ix/vscode/eve-o-provit
   make docker-rebuild
   ```

2. **EVE Test Account Credentials:**

   Erstelle `/home/ix/vscode/eve-o-provit/.env` (neben docker-compose.yml):

   ```bash
   # EVE Online Test Account
   EVE_TEST_CHARACTER=YourCharacterName
   EVE_TEST_USERNAME=your-eve-email@example.com
   EVE_TEST_PASSWORD=your-eve-password
   ```

3. **Playwright installiert:**

   ```bash
   cd frontend
   npm install
   npx playwright install chromium
   ```

## Tests ausführen

### Headless (CI-Mode)

```bash
cd frontend
npm run test:e2e
```

### Headed (Browser sichtbar)

```bash
npm run test:e2e:headed
```

### Debug Mode (mit Playwright Inspector)

```bash
npm run test:e2e:debug
```

### UI Mode (interaktive Test-UI)

```bash
npm run test:e2e:ui
```

## Test-Struktur

```txt
tests/
├── e2e/
│   ├── auth.spec.ts          # EVE SSO Login Flow
│   └── market.spec.ts         # Market Features (TODO)
└── screenshots/               # Debug Screenshots (gitignored)
    ├── 01-character-selection.png
    ├── 02-after-character-click.png
    ├── 03-login-form.png
    ├── 04-credentials-filled.png
    ├── 05-authorize-page.png
    └── 06-back-to-app.png
```

## EVE SSO Login Flow (zweistufig)

1. **App:** Login Button klicken → Redirect zu EVE SSO
2. **EVE SSO (Stufe 1):** Character aus Account wählen
3. **EVE SSO (Stufe 2):** Username/Password eingeben
4. **EVE SSO (Optional):** Authorize (nur beim ersten Mal)
5. **App:** Redirect zurück → Character Info anzeigen

## Selektoren experimentell finden

Für manuelle Exploration:

```bash
npm run test:e2e:debug
```

Dann im Test `page.pause()` nutzen - Playwright Inspector öffnet sich und zeigt:

- DOM-Struktur
- Verfügbare Selektoren
- Interaction Recording

## Troubleshooting

### Services laufen nicht

```bash
make docker-ps  # Status prüfen
make docker-logs  # Logs anzeigen
```

### Screenshots fehlen

```bash
mkdir -p tests/screenshots
```

### Browser findet Element nicht

- Timeout erhöhen: `{ timeout: 15000 }`
- Screenshot machen: `await page.screenshot({ path: 'debug.png' })`
- Playwright Inspector nutzen: `npm run test:e2e:debug`

### EVE SSO ändert Struktur

Screenshots in `tests/screenshots/` prüfen und Selektoren anpassen.

## CI Integration (TODO)

GitHub Actions Workflow für automatische E2E Tests bei PRs:

```yaml
# .github/workflows/e2e-tests.yml
name: E2E Tests
on: [pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: make docker-rebuild
      - run: cd frontend && npm ci
      - run: cd frontend && npx playwright install --with-deps chromium
      - run: cd frontend && npm run test:e2e
        env:
          EVE_TEST_CHARACTER: ${{ secrets.EVE_TEST_CHARACTER }}
          EVE_TEST_USERNAME: ${{ secrets.EVE_TEST_USERNAME }}
          EVE_TEST_PASSWORD: ${{ secrets.EVE_TEST_PASSWORD }}
```

## Weitere Tests (geplant)

- [ ] Market Orders anzeigen
- [ ] Character Switching
- [ ] Logout Flow
- [ ] Token Refresh
- [ ] Navigation Calculator
- [ ] Cargo Calculator
