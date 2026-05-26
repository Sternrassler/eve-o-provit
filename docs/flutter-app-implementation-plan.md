# Flutter Tablet-App — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Flutter Android-tablet client for eve-o-provit (core trading workflow + character), consuming the existing Go/Fiber API, with a mobile EVE-SSO Bearer-token flow.

**Architecture:** Additive Go backend extension (mobile token endpoints + multi-audience JWT validation + Bearer middleware; web cookie path unchanged) plus a new Flutter app in `eve-o-provit/app/`. Design direction C (neutral shadcn base + one blue accent, light+dark), modern two-pane (List-Detail) at a single ~840dp breakpoint tuned to the Galaxy Tab.

**Tech Stack:** Go/Fiber + jwx/v2 (backend); Flutter + Material 3, flutter_riverpod, go_router, dio, flutter_secure_storage, flutter_web_auth_2 (app).

**Spec:** `docs/flutter-app-design.md`

> Plan location note: stored as a plain doc (not `docs/superpowers/plans/`) — eve-o-provit deliberately removed `docs/superpowers/`.

---

## Phase 0 — Manual prerequisite (you, in the EVE Developer portal)

> **This is a manual step for the human owner. The implementation cannot proceed past Phase 3 (auth) without the resulting `client_id`.** EVE SSO applications allow only ONE callback URL each, so the mobile app needs its own application.

- [ ] **Step 1: Register a second EVE application**
  - Go to <https://developers.eveonline.com/> → **Manage Applications** → **Create New Application**.
  - **Name:** `eve-o-provit-mobile` (any name).
  - **Connection Type:** *Authentication & API Access* (so scopes can be requested).
  - **Permissions / Scopes:** add exactly the six the web app uses:
    `esi-location.read_location.v1`, `esi-location.read_ship_type.v1`, `esi-skills.read_skills.v1`, `esi-clones.read_clones.v1`, `esi-assets.read_assets.v1`, `esi-ui.write_waypoint.v1`.
  - **Callback URL:** `eveoprovit://callback`
  - Save.

- [ ] **Step 2: Record the credentials**
  - Copy the new application's **Client ID** → this is `EVE_MOBILE_CLIENT_ID`.
  - The mobile flow is a public client (PKCE), so no client secret is used.
  - Hand these to the implementer: `EVE_MOBILE_CLIENT_ID=<…>` and confirm callback `eveoprovit://callback`.

- [ ] **Step 3: Confirm the existing web app is untouched** — the web application (`client_id 0828b4bcd…`, callback `localhost:9000/callback`) keeps working unchanged.

---

## Phase 1 — Backend: multi-audience validation + Bearer middleware (Go, TDD)

**Files:**
- Modify: `backend/pkg/evesso/jwt.go` (validator accepts multiple client IDs)
- Modify: `backend/pkg/evesso/jwt_test.go`
- Modify: `backend/pkg/evesso/middleware.go` (accept Bearer in addition to cookie)
- Modify: `backend/pkg/evesso/middleware_test.go` (create if absent)

### Task 1.1: TokenValidator accepts multiple audiences

- [ ] **Step 1: Write the failing test** — add to `backend/pkg/evesso/jwt_test.go`:

```go
func TestValidate_AcceptsSecondaryClientID(t *testing.T) {
	// Build a validator that trusts BOTH the web and mobile client IDs.
	keys := testKeySet(t) // existing test helper that returns the signing jwk.Set
	v := NewTokenValidatorWithKeySet("web-client", keys)
	v.AddAcceptedClientID("mobile-client")

	// Token issued to the mobile client (aud = [mobile-client, "EVE Online"]).
	token := signTestToken(t, map[string]any{
		"sub": "CHARACTER:EVE:12345",
		"aud": []string{"mobile-client", audienceEVE},
		"iss": "login.eveonline.com",
		"name": "Ix Sternrassler",
	})

	info, err := v.Validate(context.Background(), token)
	require.NoError(t, err)
	require.Equal(t, 12345, info.CharacterID)
}
```
> If `testKeySet`/`signTestToken` helpers don't exist with these names, reuse the helpers already present in `jwt_test.go` (check the existing passing tests) and adapt the call sites — keep the assertion semantics.

- [ ] **Step 2: Run it, verify it fails**

Run: `cd backend && TZ=UTC go test ./pkg/evesso/ -run TestValidate_AcceptsSecondaryClientID -v`
Expected: FAIL — `AddAcceptedClientID` undefined.

- [ ] **Step 3: Implement multi-audience support** in `backend/pkg/evesso/jwt.go`:

```go
// TokenValidator field change: keep clientID for back-compat, add accepted set.
type TokenValidator struct {
	clientID        string
	acceptedClients []string // includes clientID; any match satisfies audience
	keys            jwk.Set
}
```
Update both constructors to seed `acceptedClients: []string{clientID}`. Add:

```go
// AddAcceptedClientID lets the validator accept tokens whose audience contains an
// additional client ID (e.g. the mobile app), alongside the primary one.
func (v *TokenValidator) AddAcceptedClientID(id string) {
	if id == "" {
		return
	}
	v.acceptedClients = append(v.acceptedClients, id)
}
```
Replace the audience check in `Validate`:

```go
	aud := tok.Audience()
	if !containsString(aud, audienceEVE) || !audienceHasAcceptedClient(aud, v.acceptedClients) {
		return nil, fmt.Errorf("invalid audience: %v", aud)
	}
```
Add helper:

```go
func audienceHasAcceptedClient(aud, accepted []string) bool {
	for _, id := range accepted {
		if containsString(aud, id) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run tests, verify pass**

Run: `cd backend && TZ=UTC go test ./pkg/evesso/ -v`
Expected: PASS (new test + all existing).

- [ ] **Step 5: Commit**

```bash
git add backend/pkg/evesso/jwt.go backend/pkg/evesso/jwt_test.go
git commit -m "feat(auth): TokenValidator accepts multiple client-id audiences"
```

### Task 1.2: Bearer-aware auth middleware

- [ ] **Step 1: Write the failing test** in `backend/pkg/evesso/middleware_test.go`:

```go
func TestAuthMiddleware_AcceptsBearer(t *testing.T) {
	v := NewTokenValidatorWithKeySet("mobile-client", testKeySet(t))
	token := signTestToken(t, map[string]any{
		"sub": "CHARACTER:EVE:777", "aud": []string{"mobile-client", audienceEVE},
		"iss": "login.eveonline.com",
	})
	app := fiber.New()
	app.Get("/p", NewAuthMiddleware(v), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"cid": c.Locals("character_id")})
	})
	req := httptest.NewRequest("GET", "/p", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
}
```

- [ ] **Step 2: Run it, verify it fails** — `TZ=UTC go test ./pkg/evesso/ -run TestAuthMiddleware_AcceptsBearer -v` → FAIL (401, middleware only reads cookie).

- [ ] **Step 3: Implement** — in `backend/pkg/evesso/middleware.go`, add a token extractor and use it in both middlewares:

```go
// bearerOrCookie returns the access token from the Authorization: Bearer header
// (mobile clients) or the eve_access_token cookie (web), in that order.
func bearerOrCookie(c *fiber.Ctx) string {
	if h := c.Get("Authorization"); len(h) > 7 && strings.EqualFold(h[:7], "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return c.Cookies("eve_access_token")
}
```
Replace `accessToken := c.Cookies("eve_access_token")` with `accessToken := bearerOrCookie(c)` in both `NewAuthMiddleware` and `NewOptionalAuthMiddleware`. Add `"strings"` to imports.

- [ ] **Step 4: Run tests, verify pass** — `TZ=UTC go test ./pkg/evesso/ -v` → PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/pkg/evesso/middleware.go backend/pkg/evesso/middleware_test.go
git commit -m "feat(auth): accept Authorization: Bearer in addition to cookie"
```

### Task 1.3: Mobile callback + refresh endpoints (token in JSON body)

**Files:** Modify `backend/pkg/evesso/auth_handler.go` (+ `auth_handler_test.go`); modify `backend/cmd/api/main.go` (wire routes + config).

- [ ] **Step 1: Write the failing test** in `auth_handler_test.go` — assert a mobile handler returns tokens in the body (reuse the existing test's ExchangeCode stubbing pattern; check how `HandleCallback` is currently tested and mirror it):

```go
func TestHandleMobileCallback_ReturnsTokensInBody(t *testing.T) {
	// Arrange a handler with a stubbed token exchange returning access+refresh.
	// (Mirror the stubbing used by the existing HandleCallback test.)
	h := newTestMobileHandler(t, "mobile-client", "eveoprovit://callback")
	body := `{"code":"abc","code_verifier":"v"}`
	req := httptest.NewRequest("POST", "/auth/mobile/callback", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := h.app.Test(req)
	require.Equal(t, 200, resp.StatusCode)
	var out map[string]any
	json.NewDecoder(resp.Body).Decode(&out)
	require.NotEmpty(t, out["access_token"])
	require.NotEmpty(t, out["refresh_token"])
	require.NotNil(t, out["character"])
}
```
> If the existing tests inject the token exchange via a package var or interface, reuse that seam. If `ExchangeCode` is a free function, introduce a small injectable function field on `AuthHandler` (`exchange func(...) (*TokenResponse, error)`) defaulting to `ExchangeCode`, and set a stub in the test. Keep the web `HandleCallback` using the same default.

- [ ] **Step 2: Run it, verify it fails** — FAIL (`HandleMobileCallback` undefined).

- [ ] **Step 3: Implement** — extend `AuthHandler` with mobile creds and two handlers:

```go
// Add fields:  mobileClientID, mobileRedirectURI string
// New constructor option or setter:
func (h *AuthHandler) WithMobile(clientID, redirectURI string) *AuthHandler {
	h.mobileClientID = clientID
	h.mobileRedirectURI = redirectURI
	return h
}

type mobileTokenResponse struct {
	AccessToken  string            `json:"access_token"`
	RefreshToken string            `json:"refresh_token"`
	ExpiresIn    int               `json:"expires_in"`
	Character    characterResponse `json:"character"`
}

// HandleMobileCallback: POST /auth/mobile/callback — like HandleCallback but uses
// the mobile client_id/redirect_uri and returns tokens in the JSON body (no cookies).
func (h *AuthHandler) HandleMobileCallback(c *fiber.Ctx) error {
	var req callbackRequest
	if err := c.BodyParser(&req); err != nil || req.Code == "" || req.CodeVerifier == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	tokenResp, err := ExchangeCode(c.Context(), req.Code, h.mobileRedirectURI, h.mobileClientID, req.CodeVerifier)
	if err != nil {
		log.Printf("ERROR [auth/mobile/callback] ExchangeCode: %v", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "failed to exchange code"})
	}
	charInfo, err := h.validator.Validate(c.Context(), tokenResp.AccessToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "failed to verify token"})
	}
	return c.JSON(mobileTokenResponse{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
		Character: characterResponse{
			CharacterID: charInfo.CharacterID, CharacterName: charInfo.CharacterName,
			Scopes: strings.Split(charInfo.Scopes, " "), PortraitURL: GetPortraitURL(charInfo.CharacterID, 128),
		},
	})
}

// HandleMobileRefresh: POST /auth/mobile/refresh {"refresh_token":"…"} → new tokens in body.
func (h *AuthHandler) HandleMobileRefresh(c *fiber.Ctx) error {
	var req struct{ RefreshToken string `json:"refresh_token"` }
	if err := c.BodyParser(&req); err != nil || req.RefreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing refresh_token"})
	}
	tokenResp, err := RefreshToken(c.Context(), req.RefreshToken, h.mobileClientID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "failed to refresh"})
	}
	return c.JSON(mobileTokenResponse{AccessToken: tokenResp.AccessToken, RefreshToken: tokenResp.RefreshToken, ExpiresIn: tokenResp.ExpiresIn})
}
```

- [ ] **Step 4: Run tests, verify pass** — `TZ=UTC go test ./pkg/evesso/ -v` → PASS.

- [ ] **Step 5: Wire routes + config** in `backend/cmd/api/main.go`:
  - Read env: `mobileClientID := os.Getenv("EVE_MOBILE_CLIENT_ID")`, `mobileRedirect := getenvDefault("EVE_MOBILE_REDIRECT_URI", "eveoprovit://callback")`.
  - When building the AuthHandler: `c.AuthHandler.WithMobile(mobileClientID, mobileRedirect)`.
  - When building the TokenValidator: after construction, `if mobileClientID != "" { tokenValidator.AddAcceptedClientID(mobileClientID) }`.
  - Add routes under the `auth` group:
    ```go
    auth.Post("/mobile/callback", c.AuthHandler.HandleMobileCallback)
    auth.Post("/mobile/refresh", c.AuthHandler.HandleMobileRefresh)
    ```

- [ ] **Step 6: Build + full backend suite**

Run: `cd backend && go build ./... && TZ=UTC go test ./... `
Expected: builds; existing suite stays green (web cookie path unchanged).

- [ ] **Step 7: Commit**

```bash
git add backend/pkg/evesso/auth_handler.go backend/pkg/evesso/auth_handler_test.go backend/cmd/api/main.go
git commit -m "feat(auth): mobile callback/refresh endpoints returning tokens in body"
```

### Task 1.4: Document the mobile auth in the API docs

- [ ] **Step 1:** Add the two endpoints + Bearer note to `docs/eve-o-provit.md` (Auth table). Commit `docs: document mobile auth endpoints`.

---

## Phase 2 — Flutter app scaffold, theme & layout

**Files (created under `eve-o-provit/app/`):** standard Flutter project + `lib/core/theme.dart`, `lib/core/breakpoint.dart`, `lib/core/env.dart`, `lib/core/router.dart`.

### Task 2.1: Scaffold the project

- [ ] **Step 1:** `cd eve-o-provit && flutter create --org de.sternrassler --project-name eve_o_provit --platforms android app`
- [ ] **Step 2:** Add dependencies — edit `app/pubspec.yaml` `dependencies:`:
  ```yaml
  flutter_riverpod: ^3.0.0
  go_router: ^17.0.0
  dio: ^5.9.0
  flutter_secure_storage: ^10.0.0
  flutter_web_auth_2: ^4.1.0
  crypto: ^3.0.0   # PKCE S256
  ```
  Run `cd app && flutter pub get`.
- [ ] **Step 3: INTERNET permission (gotcha):** ensure `<uses-permission android:name="android.permission.INTERNET"/>` and `ACCESS_NETWORK_STATE` are in `app/android/app/src/main/AndroidManifest.xml` (NOT only debug/profile — see `brain/kontext/flutter-build.md`).
- [ ] **Step 4: Register the custom-scheme intent** for `flutter_web_auth_2` — add to `app/android/app/src/main/AndroidManifest.xml` an `<activity>` for `com.linusu.flutter_web_auth_2.CallbackActivity` with an intent-filter for scheme `eveoprovit` (per flutter_web_auth_2 README). 
- [ ] **Step 5: Commit** `chore(app): scaffold flutter project + deps`.

### Task 2.2: Env config (dart-define)

- [ ] **Step 1:** Create `app/lib/core/env.dart`:
```dart
class Env {
  static const apiBaseUrl = String.fromEnvironment('API_BASE_URL', defaultValue: 'http://10.0.2.2:9001');
  static const eveClientId = String.fromEnvironment('EVE_CLIENT_ID');
  static const redirectUri = 'eveoprovit://callback';
  static const scopes = [
    'esi-location.read_location.v1','esi-location.read_ship_type.v1','esi-skills.read_skills.v1',
    'esi-clones.read_clones.v1','esi-assets.read_assets.v1','esi-ui.write_waypoint.v1',
  ];
}
```
> `10.0.2.2` is the emulator's host alias; real-device runs pass `--dart-define=API_BASE_URL=http://<dev-lan-ip>:9001`.
- [ ] **Step 2: Commit** `feat(app): env config via dart-define`.

### Task 2.3: Theme (direction C) — TDD on the breakpoint, visual-review on theme

- [ ] **Step 1:** Create `app/lib/core/theme.dart` with light+dark `ThemeData` from a neutral seed plus the blue accent. Concrete starting tokens (from the spec):
```dart
import 'package:flutter/material.dart';

const _accent = Color(0xFF1F6FB0); // ~oklch(0.45 0.13 230), primary/active/profit
ThemeData buildTheme(Brightness b) {
  final scheme = ColorScheme.fromSeed(seedColor: _accent, brightness: b).copyWith(
    surface: b == Brightness.light ? const Color(0xFFFFFFFF) : const Color(0xFF242424),
    onSurface: b == Brightness.light ? const Color(0xFF333333) : const Color(0xFFFAFAFA),
  );
  return ThemeData(
    colorScheme: scheme, useMaterial3: true,
    cardTheme: CardThemeData(
      elevation: 1, shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
    ),
  );
}
```
- [ ] **Step 2: Commit** `feat(app): material 3 theme (direction C, light+dark)`.

### Task 2.4: Adaptive breakpoint helper (TDD)

- [ ] **Step 1: Write the failing test** `app/test/breakpoint_test.dart`:
```dart
import 'package:flutter_test/flutter_test.dart';
import 'package:eve_o_provit/core/breakpoint.dart';
void main() {
  test('two-pane at/above 840dp, single-pane below', () {
    expect(isTwoPane(800), isFalse); // tablet portrait
    expect(isTwoPane(840), isTrue);
    expect(isTwoPane(1280), isTrue); // tablet landscape
  });
}
```
- [ ] **Step 2: Run, verify fail** — `cd app && flutter test test/breakpoint_test.dart` → FAIL (no `isTwoPane`).
- [ ] **Step 3: Implement** `app/lib/core/breakpoint.dart`:
```dart
/// Single device-tuned breakpoint: Galaxy Tab portrait ~800dp (single-pane),
/// landscape ~1280dp (two-pane). 840 sits cleanly between the two.
const double kTwoPaneBreakpoint = 840;
bool isTwoPane(double widthDp) => widthDp >= kTwoPaneBreakpoint;
```
- [ ] **Step 4: Run, verify pass.**
- [ ] **Step 5: Commit** `feat(app): adaptive two-pane breakpoint (~840dp)`.

---

## Phase 3 — Auth (mobile EVE SSO → Bearer)

**Files:** `app/lib/auth/pkce.dart`, `token_store.dart`, `auth_repository.dart`, `auth_controller.dart` (Riverpod), `login_screen.dart`; `app/lib/api/dio_client.dart`.

### Task 3.1: PKCE generator (TDD)
- [ ] **Step 1: Failing test** `app/test/pkce_test.dart`: assert `pkce()` returns a verifier (43–128 chars, URL-safe) and an S256 challenge that equals base64url(sha256(verifier)) without padding.
- [ ] **Step 2:** Run → fail.
- [ ] **Step 3: Implement** `app/lib/auth/pkce.dart` using `crypto` (sha256) + `base64UrlEncode` with `=` stripped; verifier from `Random.secure()`.
- [ ] **Step 4:** Run → pass. **Step 5:** Commit `feat(app): PKCE S256 helper`.

### Task 3.2: Token store (flutter_secure_storage)
- [ ] **Step 1:** Create `token_store.dart` — `read/write/clear` for `access_token`, `refresh_token` via `FlutterSecureStorage`. (Widget/integration-tested later; thin wrapper, no unit test needed.)
- [ ] **Step 2:** Commit `feat(app): secure token store`.

### Task 3.3: Login flow (custom tab → backend → tokens)
- [ ] **Step 1:** Implement `auth_repository.dart`:
  - `login()`: build PKCE, build authorize URL (`https://login.eveonline.com/v2/oauth/authorize?response_type=code&redirect_uri=eveoprovit://callback&client_id=${Env.eveClientId}&scope=${scopes.join(' ')}&state=…&code_challenge=…&code_challenge_method=S256`), call `FlutterWebAuth2.authenticate(url, callbackUrlScheme: 'eveoprovit')`, parse `code`+`state` (verify state), `POST $apiBaseUrl/auth/mobile/callback {code, code_verifier}` via dio, store tokens, return character.
  - `refresh()`: `POST /auth/mobile/refresh {refresh_token}` → store new tokens.
  - `logout()`: clear store.
- [ ] **Step 2:** `auth_controller.dart` — Riverpod `AsyncNotifier` exposing `AuthState { unauthenticated | authenticated(character) | loading }`, methods `login/logout`, reads token store on init.
- [ ] **Step 3:** `login_screen.dart` — centered "Login mit EVE" button (accent), error display.
- [ ] **Step 4:** Commit `feat(app): EVE SSO mobile login flow`.

### Task 3.4: dio client with Bearer + refresh interceptor
- [ ] **Step 1:** `app/lib/api/dio_client.dart` — dio with `baseUrl: Env.apiBaseUrl`; request interceptor adds `Authorization: Bearer <access>`; on `401`, call `auth_repository.refresh()` once and retry; if refresh fails, clear tokens and surface unauthenticated.
- [ ] **Step 2:** Commit `feat(app): dio client with bearer + refresh interceptor`.

---

## Phase 4 — Trading feature

**Files:** `app/lib/api/trading_api.dart` (+ DTOs), `app/lib/features/trading/{providers,trading_screen,route_list,route_detail,route_card,filters}.dart`.

### Task 4.1: API + DTOs (TDD on JSON mapping)
- [ ] **Step 1: Failing test** `app/test/trading_dto_test.dart`: `RouteCalculationResponse.fromJson(sample)` maps `region_name`, `ship_name`, `cargo_capacity`, `routes[].item_name/isk_per_hour/total_profit/spread_percent`. Use a realistic JSON sample matching the backend `models.RouteCalculationResponse`.
- [ ] **Step 2:** Run → fail. **Step 3:** Implement DTOs + `trading_api.dart` (dio): `calculateRoutes(req)` → `POST /api/v1/trading/routes/calculate`; `regions()` → `GET /api/v1/sde/regions`; `staleness(region)`, `refreshMarket(region,type)`, `setWaypoint(...)` → `POST /api/v1/esi/ui/autopilot/waypoint`. **Step 4:** Run → pass. **Step 5:** Commit `feat(app): trading API client + DTOs`.

### Task 4.2: Riverpod providers
- [ ] **Step 1:** `providers.dart` — `regionsProvider` (FutureProvider), `selectedRegion/selectedShip/filters` (Notifiers), `routesProvider` (AsyncNotifier calling `calculateRoutes`; tolerates empty result by exposing `routes` + `warning`). **Step 2:** Commit `feat(app): trading providers`.

### Task 4.3: Route card + list + detail widgets (direction C)
- [ ] **Step 1:** `route_card.dart` — Card (radius 10, elevation 1): item name + sec-zone chip, big accent ISK/h number, muted meta line (spread · profit · route); selected state = accent border. Follow the approved mockup in `docs/flutter-app-design.md` (Design-System section) for spacing/colors.
- [ ] **Step 2:** `route_detail.dart` — header label, item name, big accent ISK/h, buy→sell meta, accent **"Route in EVE setzen"** filled button calling `setWaypoint`.
- [ ] **Step 3:** `route_list.dart` — `ListView` of `RouteCard`; on tap sets selected route.
- [ ] **Step 4:** Commit `feat(app): trading route card/list/detail widgets`.

### Task 4.4: Adaptive TradingScreen (TDD on layout switch)
- [ ] **Step 1: Failing widget test** `app/test/trading_screen_layout_test.dart`: pump `TradingScreen` inside a `MediaQuery` of width 1280 → expect both `RouteList` and `RouteDetail` present; width 800 → expect only `RouteList` (detail reached via navigation). Use Riverpod overrides to stub `routesProvider` with two routes.
- [ ] **Step 2:** Run → fail. **Step 3: Implement** `trading_screen.dart`:
```dart
// inside build, using LayoutBuilder:
final twoPane = isTwoPane(constraints.maxWidth);
return twoPane
  ? Row(children: [SizedBox(width: 360, child: RouteList(...)), const VerticalDivider(width: 1), Expanded(child: RouteDetail(...))])
  : RouteList(onTap: (r) => context.push('/trading/detail', extra: r));
```
Plus the controls row (region select, ship select, refresh button + staleness, filters, "Berechnen"). **Step 4:** Run → pass. **Step 5:** Commit `feat(app): adaptive two-pane trading screen`.

---

## Phase 5 — Character feature

**Files:** `app/lib/api/character_api.dart`, `app/lib/features/character/{providers,character_screen}.dart`.

- [ ] **Task 5.1:** `character_api.dart` — `GET /api/v1/character`, `/character/ship`, `/character/ships`, `/characters/:id/skills`, `/characters/:id/fitting/:shipTypeId`. DTOs with a JSON-mapping unit test. Commit.
- [ ] **Task 5.2:** `character_screen.dart` — character header (portrait, name, id), active ship + fitting summary, skills relevant to cargo/navigation. Single-pane; uses the same card style. Commit.
- [ ] **Task 5.3:** Router + shell — `router.dart` (go_router): `/login`, `/trading`, `/trading/detail`, `/character`; redirect to `/login` when unauthenticated. Bottom `NavigationBar` (Trading · Character). `main.dart` wires `ProviderScope`, `MaterialApp.router`, `buildTheme` light/dark + `themeMode: system`. Commit `feat(app): router, nav shell, app entrypoint`.

---

## Phase 6 — End-to-end tests (tablet portrait + landscape)

**Files:** `app/integration_test/trading_flow_test.dart`, `app/integration_test/support/`, `app/Makefile` (or repo Makefile targets).

- [ ] **Task 6.1:** Captured-session support — like the web E2E, store a real captured `access`/`refresh` token for the test (do NOT commit; gitignore `app/integration_test/.auth/`). Helper seeds the token store before the app boots.
- [ ] **Task 6.2:** `trading_flow_test.dart` — boot app authenticated, select The Forge, refresh market, calculate routes (tolerate cold-empty with bounded retry, mirroring the web de-flake lesson), assert routes appear, open a route, tap "Route in EVE setzen" (mock the waypoint POST). Run once forcing width ~800 (single-pane) and once ~1280 (two-pane) via `tester.binding.window.physicalSizeTestValue`.
- [ ] **Task 6.3:** Make targets `e2e-test-phone` / `e2e-test-tablet` analogous to Kochen; document in `app/README.md`. Commit.

---

## Phase 7 — Wire-up, docs, manual device check

- [ ] **Task 7.1:** `app/README.md` — run commands incl. the real-device invocation:
  `flutter run --dart-define=API_BASE_URL=http://<dev-lan-ip>:9001 --dart-define=EVE_CLIENT_ID=<mobile-client-id>`.
- [ ] **Task 7.2:** Update `docs/eve-o-provit.md` + `docs/README.md` to mention the Flutter app (`app/`). Commit.
- [ ] **Task 7.3: Manual device validation (you):** install on the Galaxy Tab, log in via EVE SSO (custom tab), run a real trading calculation portrait + landscape, set a waypoint, confirm it appears in the EVE client. Note any visual adjustments to the accent tokens.

---

## Self-review notes (coverage)

- Spec "EVE-Dev-App / single callback" → Phase 0 + Phase 1.3 config + Phase 2.2 env.
- Spec "Bearer + secure storage" → Phase 1.2, 3.2, 3.4.
- Spec "multi-audience" gotcha → Phase 1.1 (validator) — required, else mobile tokens 401 on the protected API.
- Spec "direction C / light+dark" → Phase 2.3, 4.3.
- Spec "~840dp two-pane" → Phase 2.4 + 4.4.
- Spec "trading workflow + character" → Phases 4 + 5.
- Spec "tests incl. tablet portrait+landscape E2E" → Phase 6.
- Spec "INTERNET permission / dart-define gotchas" → Phase 2.1, 2.2, 7.1.
