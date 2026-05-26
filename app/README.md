# eve-o-provit Flutter Tablet App

Flutter Android-tablet client for eve-o-provit. Covers the core trading workflow (region/ship select → filter → route calculation → waypoint in EVE) and character view (info, active ship, skills/fitting). Consumes the existing Go/Fiber backend API over Bearer-token auth.

**Device target:** Galaxy Tab SM-X236B (portrait ~800dp, landscape ~1280dp). Two-pane layout at ≥840dp (landscape), single-pane below that.

---

## Prerequisites

1. **Running eve-o-provit backend** — see `../backend/`. The backend must be built with mobile auth support (`EVE_MOBILE_CLIENT_ID` env set, see [Backend mobile auth](#backend-mobile-auth)).

2. **A second EVE developer application** — EVE SSO allows only **one** callback URL per app. The web app uses `localhost:9000/callback`; the mobile app needs its own app with callback `eveauth-eveoprovit://callback`.
   - Register at <https://developers.eveonline.com/> → Manage Applications → Create New Application.
   - Connection type: *Authentication & API Access*.
   - Scopes: `esi-location.read_location.v1`, `esi-location.read_ship_type.v1`, `esi-skills.read_skills.v1`, `esi-clones.read_clones.v1`, `esi-assets.read_assets.v1`, `esi-ui.write_waypoint.v1`.
   - Callback URL: `eveauth-eveoprovit://callback`.
   - Note the **Client ID** — this is your `EVE_CLIENT_ID` for `--dart-define`.

3. **Flutter SDK** — tested with Flutter/Dart SDK matching `environment.sdk: ^3.11.4` in `pubspec.yaml`.

---

## Build / Run

Both values below are required at build time — there are no safe hardcoded defaults for production use.

```bash
flutter run \
  --dart-define=API_BASE_URL=http://<host>:9001 \
  --dart-define=EVE_CLIENT_ID=<mobile-client-id>
```

### Host address by device type

| Scenario | `API_BASE_URL` | Extra step |
|---|---|---|
| Android emulator | `http://10.0.2.2:9001` | none (`10.0.2.2` = host loopback) |
| USB-connected physical device | `http://localhost:9001` | `adb reverse tcp:9001 tcp:9001` |
| LAN (same Wi-Fi) | `http://<dev-lan-ip>:9001` | none |

The `adb reverse` command tunnels port 9001 from the device back to the development machine. Run it once before `flutter run`; repeat after re-connecting the cable.

---

## Backend Mobile Auth

The Go backend needs two additional env variables (passed via `deployments/docker-compose.yml`):

| Variable | Required | Default | Description |
|---|---|---|---|
| `EVE_MOBILE_CLIENT_ID` | yes | — | Client ID of the second EVE developer app |
| `EVE_MOBILE_REDIRECT_URI` | no | `eveauth-eveoprovit://callback` | Must match the EVE app registration |

The backend exposes two mobile-only endpoints (web cookie path unchanged):
- `POST /auth/mobile/callback` — exchanges PKCE code, returns `{access_token, refresh_token, character}` in JSON body.
- `POST /auth/mobile/refresh` — exchanges a refresh token, returns new tokens in JSON body.

---

## Tests

```bash
# All unit + widget tests (no device needed)
make test           # or: flutter test

# E2E / integration-style flow tests (headless, deviceless)
make e2e-test       # or: flutter test test/e2e/

# Static analysis
make analyze        # or: flutter analyze
```

**What is not in CI:** The real EVE SSO login (Custom Tab), live route calculation, and waypoint-setting require a physical device or emulator connected to a live backend. These are validated manually on-device.

---

## Stack

| Purpose | Package |
|---|---|
| State | `flutter_riverpod` |
| Routing | `go_router` |
| HTTP | `dio` (Bearer interceptor + auto-refresh on 401) |
| Token storage | `flutter_secure_storage` |
| EVE SSO login | `flutter_web_auth_2` (Custom Tab, PKCE/S256 on device) |
| UI | Flutter Material 3, Design direction C |

Design direction C: neutral shadcn-base palette + one blue accent (`~oklch(0.45 0.13 230)`), light + dark `ColorScheme`. Cards with 10px radius, accent on primary actions, profit highlight, and active states.

---

## Gotchas

- **INTERNET permission is in the main manifest** (`android/app/src/main/AndroidManifest.xml`), not only in the debug overlay. Without it the release build throws `Failed host lookup`.
- **Cleartext HTTP is debug-only.** `android:usesCleartextTraffic="true"` lives in `android/app/src/debug/AndroidManifest.xml` and is not merged into release builds. Release/production connects over HTTPS.
- **`--dart-define` values are burned in at build time** via `String.fromEnvironment(...)`. If the app points to the wrong host, the build was run without the flag — not a runtime config issue.
- **Custom scheme `eveauth-eveoprovit`** is registered in the main manifest (`com.linusu.flutter_web_auth_2.CallbackActivity`) and must match the callback URL registered in the EVE developer portal.
- **Second EVE app is mandatory.** The existing web app's `client_id` cannot be reused — EVE allows exactly one callback URL per registered application.
