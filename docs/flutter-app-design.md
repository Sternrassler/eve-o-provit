# Flutter Tablet-App — Design

**Stand:** 2026-05-26 · **Status:** Design abgenommen, Implementierungs-Plan ausstehend

Ein zweites Frontend für eve-o-provit: eine Flutter-App für das Android-Tablet, parallel zum Next.js-Web-Frontend. Konsumiert die bestehende Go/Fiber-Backend-API. Farbstil am Web orientiert (neutrales shadcn-System), modernes, UX-zentriertes Material-3-Design.

## Zielbild & Scope

**In Scope (v1):** Der vollständige Trading-Workflow + Character-Ansicht.
- Region/Schiff wählen → Filter → **Routen berechnen** → Ergebnis → **Waypoint in EVE setzen**
- Markt-Daten-Refresh + Staleness-Anzeige je Region
- Character-Info (Skills, aktives Schiff, Fitting)

**Out of Scope (v1, bewusst später):** Multi-Hub, ROI-Rechner, Trends, Watchlist (im Web Phase 2/3, teils Platzhalter).

**Nutzungsgerät:** Galaxy Tab `SM X236B` — Hochkant ~800dp, Quer ~1280dp.

## Kern-Entscheidungen

| Thema | Entscheidung |
|------|--------------|
| Umfang | Kern-Trading-Fokus + Character |
| Backend-Ziel | Konfigurierbar via `--dart-define=API_BASE_URL=…` (Dev-LAN / optional Prod) |
| Mobile-Auth | EVE SSO via Custom Tab (PKCE on device) → **Bearer-Token + `flutter_secure_storage`** |
| Repo-Ort | Monorepo, neuer Ordner `app/` (neben `frontend/` + `backend/`, Portfolio-Konvention) |
| Design-Richtung | **C** — neutrale shadcn-Basis + **ein blauer Akzent**, Light + Dark |
| Tablet-Layout | **Modern two-pane** (List-Detail) im Quer, single-pane im Hoch; **ein Breakpoint ~840dp** auf das Gerät zugeschnitten |

## Architektur & Stack

Identisch zum bewährten Portfolio (kochen/foodlens/chartwarden):

| Zweck | Paket |
|------|-------|
| State | `flutter_riverpod` (+ `riverpod_annotation`) |
| Routing | `go_router` |
| HTTP | `dio` (Interceptor für `Authorization: Bearer` + Token-Refresh) |
| Token-Storage | `flutter_secure_storage` |
| SSO-Login | `flutter_web_auth_2` (Custom Tab) + PKCE/S256 on device |
| Backend-Host | `--dart-define=API_BASE_URL` (Default-Wert greift nur ohne Flag) |

```
app/lib/
  core/        Theme (Richtung C, Light+Dark ColorScheme) · Breakpoint-Helper (~840dp) · Router (go_router)
  api/         dio-Client + Auth-/Refresh-Interceptor · Repository-Layer (mappt Backend-Endpunkte)
  auth/        Login-Flow, Token-Store, Riverpod auth-Provider
  features/
    trading/   Provider + TradingScreen (two-pane) + RouteList + RouteDetail
    character/ Provider + CharacterScreen
```

## Design-System (Richtung C)

Basis ist die neutrale shadcn-Palette des Web-Frontends (`frontend/src/app/globals.css`, OKLCH, Chroma 0 = Graustufen) — übersetzt in ein Material-3 `ColorScheme` für Light + Dark, plus **ein** blauer Akzent (abgeleitet aus der Web-Chart-Blau-Familie, für Primäraktionen/aktive Zustände/Profit-Highlights).

Start-Tokens (in der Implementierung final justierbar):
- **Neutral:** Surface/Background `oklch(1 0 0)` (light) / `oklch(0.145 0 0)` (dark); Text `oklch(0.205 0 0)` / `oklch(0.985 0 0)`; Muted `oklch(0.556 0 0)`; Border `oklch(0.922 0 0)`.
- **Akzent (primary):** `~oklch(0.45 0.13 230)` (light) / hellere Variante für dark. Verwendet für Primär-Button, aktive Karte, große Profit-Zahl.
- **Destructive:** rot, analog Web.
- **Radius:** ~10px (entspricht Web `--radius: 0.625rem`). **Schrift:** Material-3-Default (Roboto/Geräte-Sans); Geist ist Web-spezifisch und wird nicht nachgebaut.

UX-Prinzipien: Karten mit Luft, weiche Flächen, klare Hierarchie, große Profit-Zahl, Akzent lenkt den Blick (kein dichtes Tabellen-Master-Detail).

## Layout-Regel

Ein einziger, gerätebezogener Breakpoint — **kein** generisches 600/840/1200-Stufenmodell:

- Breite **< 840dp** → **single-pane** (Hochkant ~800dp): Routen-Liste; Tippen → Detail-Screen (Navigation push).
- Breite **≥ 840dp** → **two-pane** (Quer ~1280dp): Liste links, Detail rechts gleichzeitig sichtbar.
- Width-basiert (überlebt Split-Screen) statt reiner Orientierung. Später erweiterbar, falls andere Geräte dazukommen.

## EVE-SSO-Login-Flow (mobil)

1. „Login mit EVE" → App erzeugt `state` + `code_verifier` (PKCE/S256), öffnet die EVE-Authorize-URL im Custom Tab (`flutter_web_auth_2`), `redirect_uri = eveoprovit://callback`, dieselben 6 ESI-Scopes wie das Web (`esi-location.read_location.v1`, `…read_ship_type.v1`, `esi-skills.read_skills.v1`, `esi-clones.read_clones.v1`, `esi-assets.read_assets.v1`, `esi-ui.write_waypoint.v1`).
2. EVE leitet auf `eveoprovit://callback?code=…&state=…` zurück → App prüft `state`.
3. `POST /auth/mobile/callback {code, code_verifier}` → Backend tauscht Code bei EVE SSO (PKCE), validiert das JWT lokal (jwx/v2), antwortet mit `{access_token, refresh_token, character}` **im JSON-Body**.
4. App speichert Tokens in `flutter_secure_storage`; dio-Interceptor hängt `Authorization: Bearer <access>` an alle API-Calls.
5. Bei `401` → Interceptor ruft `/auth/mobile/refresh {refresh_token}` → neuer Access-Token; schlägt das fehl → Re-Login.

## Backend-Erweiterung (Go, im selben Repo/PR)

Der Web-Cookie-Pfad bleibt **unverändert**; mobil kommt additiv dazu:

1. **Mobile Auth-Antwort:** Endpoint (`/auth/mobile/callback`), der nach dem Code-Exchange `access`/`refresh`-Token im JSON-Body liefert (statt nur HttpOnly-Set-Cookie). Akzeptiert die mobile `redirect_uri`.
2. **Bearer in der Middleware:** Die API-Auth-Middleware akzeptiert `Authorization: Bearer` **zusätzlich** zum Cookie. Lokale JWT-Validierung (jwx/v2) bleibt identisch.
3. **Refresh-Endpoint:** `/auth/mobile/refresh` tauscht den Refresh-Token gegen einen neuen Access-Token (JSON-Body).

## API-Konsum (bestehende Endpunkte, unverändert)

`POST /api/v1/trading/routes/calculate` (✓ Auth) · `GET /api/v1/sde/regions` · `GET /api/v1/market/staleness/:region` + Refresh (`GET /market/:region/:type?refresh=true`) · `GET /api/v1/character`, `/character/ship`, `/character/ships` · `GET /api/v1/characters/:characterId/skills`, `/fitting/:shipTypeId` · `POST /api/v1/esi/ui/autopilot/waypoint`.

State je Feature über Riverpod-Provider; Repository-Layer kapselt dio-Calls + DTO-Mapping.

## Tests

- **Backend (Go):** Unit-/Integrationstests für die neuen mobile-Auth-Endpunkte + Bearer-Middleware. Bestehender Web-Cookie-Pfad muss grün bleiben.
- **App — Dart-Unit:** Provider-Logik, API-/DTO-Mapping, Token-Store/Refresh-Logik.
- **App — Widget:** Breakpoint-Umschaltung two-pane↔single-pane an ~840dp; RouteCard; Auth-Zustände.
- **App — Integration-E2E:** gegen ein laufendes Backend, Tablet **Hoch + Quer** (analog `make e2e-test-tablet` bei Kochen). Auth im E2E über eine erfasste Session (wie beim Web etabliert: Token-Capture, `expires` mitsetzen).

## Externe Voraussetzungen / Abhängigkeiten

- **EVE-Developer-App:** mobile Callback-URL `eveoprovit://callback` zusätzlich registrieren (die Web-Callback `localhost:9000/callback` funktioniert nativ nicht).
- **Backend erreichbar:** Dev über LAN (`http://<dev-ip>:9001`) bzw. optional späteres Prod-Deployment.

## Gotchas (aus dem Flutter-Portfolio, `brain/kontext/flutter-build.md`)

- **INTERNET-Permission MUSS im Main-Manifest stehen** (`android/app/src/main/AndroidManifest.xml`), nicht nur im debug/profile-Overlay — sonst wirft der Release-Build `Failed host lookup`.
- `--dart-define` wird im Release korrekt gebrannt (`String.fromEnvironment` mit `const`); zeigt die App auf den falschen Host, fehlte das Build-Flag.

## Offene Punkte (für den Implementierungs-Plan)

- Exakte Akzent-Token light/dark final justieren (am Gerät prüfen).
- Release-/CI-Anbindung: ob die App in eine `make`-/release-Linie des Repos eingebunden wird (kein eigenes Deployment für v1 zwingend).
- Ordnername final: `app/` (Vorschlag, Portfolio-Konvention) vs. `mobile/`.
