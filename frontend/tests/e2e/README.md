# E2E Tests

Playwright suite against the local Docker stack (frontend `:9000`, backend `:9001`).
Design: `../../../../docs/superpowers/specs/2026-05-25-e2e-test-suite-design.md`.

## Projects
- **setup** (`auth.setup.ts`) — validates the cached real session
  (`playwright/.auth/user.json`). Fail-fast with a re-capture hint if missing/expired.
- **public** (`public/*.spec.ts`) — unauthenticated use cases: API integration,
  page smoke tests, login-UI initiation, and the logged-out "trading is inert" policy.
  CI-capable, no login required.
- **auth** (`auth/*.spec.ts`) — authenticated use cases: character page, the **full
  trading workflow** (region → staleness → refresh → calculate → route cards → set
  waypoint [intercepted] → filter effect → pagination), and logout. Reuses the
  captured session; depends on `setup`.

> The trading workflow is only operable when authenticated — logged out, the selects
> and calculate button stay disabled by design. Hence the full workflow lives in the
> `auth` project, while `public` only asserts the page loads and is inert.

## Run
```bash
# The stack starts automatically via webServer (make -C .. docker-up), or start it yourself.
npx playwright test --project=public   # no login needed (CI)
npx playwright test                     # full suite (needs a captured session)
```
First run only: `npx playwright install chromium`.

## Capturing the session (one-time / when expired)
The `auth` project needs `playwright/.auth/user.json` (gitignored — real session
cookies). Capture it with the headed helper, which opens the app, waits for a real
EVE SSO login, then saves the storage state:
```bash
node tests/e2e/capture-session.mjs
```
Verify with `npx playwright test --project=setup`.

If only the short-lived access token expired (the `setup` project fails but the
saved file is recent), renew the session non-interactively via the refresh token
(~30d) — no browser login needed:
```bash
node tests/e2e/refresh-session.mjs
```
If that reports the refresh token is also expired, fall back to `capture-session.mjs`.

## CI
Run only `--project=public` in CI (no session secret needed, deterministic). The
`auth`/`setup` projects are local/nightly because the captured session expires
(~30d refresh-token lifetime) and cannot be obtained non-interactively.
