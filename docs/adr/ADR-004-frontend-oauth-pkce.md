# ADR-004: Frontend OAuth mit PKCE statt Backend OAuth

Status: Accepted
Datum: 2025-10-26
Autoren: Development Team

> Ablageort: ADR-Dateien werden im Verzeichnis `docs/adr/` gepflegt.

## Kontext

EVE Online SSO-Integration für Character-Authentifizierung erfordert OAuth2-Flow. Die Architektur-Entscheidung "Wo wird OAuth implementiert?" hat signifikante Auswirkungen auf Code-Komplexität, Security und Wartbarkeit.

**Fragen:**

- Soll das Backend den kompletten OAuth-Flow orchestrieren (Authorization Code Flow mit Client Secret)?
- Oder soll das Frontend den OAuth-Flow übernehmen (Authorization Code Flow mit PKCE)?
- Wo werden Tokens gespeichert (Backend Sessions vs Frontend localStorage)?

**Auslöser:**

- Issue #3 "EVE SSO Login/Logout Integration" spezifizierte ursprünglich Backend-OAuth
- ~700 Zeilen Backend OAuth-Code wurden implementiert (oauth.go, handlers.go, session.go)
- Nach Review wurde erkannt: Frontend-basierter PKCE-Flow ist für SPA-Architektur besser geeignet
- Komplettes Refactoring von Backend OAuth zu Frontend PKCE durchgeführt

**Constraints:**

- Next.js 14 App Router (Client-Side SPA)
- EVE SSO unterstützt sowohl Client Secret als auch PKCE
- Kein Server-Side Rendering geplant
- Public Client (keine Backend-Secrets nötig für PKCE)

**Stakeholder:**

- Frontend Entwicklung (Next.js)
- Backend Entwicklung (Go/Fiber)
- Security/Operations
- End Users (EVE Pilots)

## Betrachtete Optionen

### Option 1: Frontend OAuth mit PKCE (GEWÄHLT)

**Architektur:**
```
User → Frontend (OAuth Flow) → EVE SSO → Frontend (Token)
                                ↓
User → Frontend (API Call mit Bearer Token) → Backend (Token Verification)
```

**Flow:**
1. Frontend generiert PKCE Code Verifier + Challenge
2. Frontend redirect zu EVE SSO mit Code Challenge
3. User authorisiert bei EVE
4. EVE redirect zu Frontend /callback mit Authorization Code
5. Frontend tauscht Code gegen Token (mit Code Verifier) direkt bei EVE
6. Frontend speichert Token in localStorage
7. Frontend macht API Calls mit Bearer Token im Authorization Header
8. Backend validiert Bearer Token on-demand mit EVE ESI /verify

**Vorteile:**
- ✅ **Kein Client Secret nötig** - PKCE eliminiert Secret-Management
- ✅ **Einfaches Backend** - Nur Token Verification (~107 Zeilen Code)
- ✅ **Standard für SPAs** - Recommended Pattern für Next.js, React, Vue
- ✅ **EVE ESI native PKCE Support** - Offiziell unterstützt
- ✅ **Kein Backend Session State** - Stateless Backend (einfacher zu skalieren)
- ✅ **Direkter Token Zugriff** - Frontend kann Token für direkte ESI-Calls nutzen
- ✅ **Weniger Backend-Komplexität** - Keine JWT-Generierung, keine Session-Cookies

**Nachteile:**
- ❌ **Token in localStorage** - XSS-Risiko (wenn XSS-Lücke existiert, Token exponiert)
- ❌ **Token Refresh im Frontend** - Frontend muss Refresh-Logic implementieren
- ❌ **Kein HttpOnly Cookie** - Token via JavaScript zugreifbar

**Risiken:**
- XSS-Attacken können Token stehlen (Mitigation: Content Security Policy, Input Sanitization)
- Token Expiry Handling im Frontend (20min EVE Token Lifetime)

**Code-Umfang:**
- Frontend: ~300 Zeilen (eve-sso.ts, auth-context.tsx, callback/page.tsx)
- Backend: ~107 Zeilen (verify.go, middleware.go)
- **Total:** ~407 Zeilen

---

### Option 2: Backend OAuth mit Client Secret

**Architektur:**
```
User → Frontend → Backend (/auth/login) → EVE SSO → Backend (/auth/callback)
                                                    ↓
                                          Backend erstellt JWT Session
                                                    ↓
User ← Frontend ← Backend (Set-Cookie: session_token)
```

**Flow:**
1. Frontend ruft Backend Endpoint GET /auth/login auf
2. Backend generiert State Parameter, redirect zu EVE SSO
3. User authorisiert bei EVE
4. EVE redirect zu Backend /auth/callback mit Authorization Code
5. Backend tauscht Code gegen Token (mit Client Secret) bei EVE
6. Backend verifiziert Token mit EVE ESI /verify
7. Backend erstellt eigenen JWT Session Token
8. Backend setzt HttpOnly Cookie mit JWT
9. Frontend macht API Calls (Cookie wird automatisch mitgeschickt)
10. Backend validiert JWT Session Token

**Vorteile:**
- ✅ **HttpOnly Cookies** - XSS-sicher (Token nicht via JavaScript zugreifbar)
- ✅ **Token nicht im Frontend** - Access Token nur im Backend
- ✅ **Server-Side Token Refresh** - Backend kann Tokens im Hintergrund refreshen
- ✅ **CSRF Protection** - SameSite Cookie Attribute

**Nachteile:**
- ❌ **Client Secret Management** - Sensitive Credential im Backend (.env, .gitignore)
- ❌ **Komplexes Backend** - OAuth Client + Session Management (~700 Zeilen Code)
- ❌ **Backend Session State** - Session Storage (Memory/Redis/PostgreSQL) nötig
- ❌ **Nicht SPA-Standard** - Pattern für Server-rendered Apps (PHP, Rails, Django)
- ❌ **JWT-Overhead** - Backend erstellt eigene Tokens parallel zu EVE Tokens
- ❌ **Cookie-Probleme** - CORS, SameSite, Domain-Matching komplexer
- ❌ **Kein direkter ESI-Zugriff** - Frontend kann nicht direkt EVE ESI aufrufen

**Risiken:**
- Client Secret Leak (Git-History, Logs, Dumps)
- Session Storage Skalierung (Redis Cluster, Session Replication)
- Cookie-basierte Auth bei Multi-Domain Setup problematisch

**Code-Umfang:**
- Frontend: ~200 Zeilen (nur UI, kein OAuth-Logic)
- Backend: ~700 Zeilen (oauth.go, handlers.go, session.go, tests)
- **Total:** ~900 Zeilen

---

### Option 3: Hybrid (Backend OAuth + Frontend Token Access)

**Architektur:**
```
Backend macht OAuth Flow → speichert Token → gibt Token an Frontend
Frontend nutzt Token für direkte ESI Calls + Backend API Calls
```

**Vorteile:**
- ✅ Client Secret sicher im Backend
- ✅ Frontend kann direkt EVE ESI aufrufen

**Nachteile:**
- ❌ **Worst of Both Worlds** - Komplexität von Option 2 + XSS-Risiko von Option 1
- ❌ Token muss vom Backend ans Frontend übertragen werden (zusätzlicher Endpoint)
- ❌ Doppelte Token-Validierung (Backend Session + EVE Token)

**Bewertung:** Nicht empfohlen - vereint Nachteile beider Ansätze.

---

## Entscheidung

**Gewählte Option:** **Option 1 - Frontend OAuth mit PKCE**

**Begründung:**

1. **SPA-Architektur:** Next.js App Router ist Client-Side SPA - Frontend OAuth ist Standard-Pattern
2. **Einfachheit:** 86% weniger Backend-Code (107 vs 700 Zeilen)
3. **Stateless Backend:** Keine Session Storage, einfacher zu skalieren
4. **EVE ESI Best Practice:** PKCE ist offiziell unterstützter Standard für Public Clients
5. **Kein Secret Management:** PKCE eliminiert Client Secret-Risiko komplett
6. **Direkter ESI-Zugriff:** Frontend kann zukünftig direkt EVE ESI aufrufen (z.B. Market Data)

**Akzeptierte Trade-offs:**

- **XSS-Risiko** akzeptiert - Mitigation via:
  - Content Security Policy (CSP) Headers
  - Input Sanitization (React XSS Protection)
  - Regelmäßige Security Audits
  - Dependency Scanning (npm audit, Snyk)
- **Token in localStorage** akzeptiert - Alternative SessionStorage hat gleiche XSS-Problematik
- **Frontend Token Refresh** akzeptiert - Implementierung straightforward

**Annahmen:**

- Next.js bleibt SPA (kein Server-Side Rendering geplant)
- Kein Multi-Domain Setup (localhost:9000 Frontend, localhost:9001 Backend)
- XSS-Schutz wird durch andere Maßnahmen gewährleistet (CSP, Sanitization)
- EVE Token Lifetime (20min) ist akzeptabel für User Experience

## Konsequenzen

### Positiv

- ✅ **Reduzierte Backend-Komplexität** - Von 700 auf 107 Zeilen Code (-85%)
- ✅ **Stateless Backend** - Keine Session Storage, horizontal skalierbar
- ✅ **Kein Client Secret** - Keine Credential Leaks möglich
- ✅ **Standard OAuth2 PKCE Flow** - Gut dokumentiert, viele Libraries
- ✅ **Direkter ESI-Zugriff möglich** - Frontend kann zukünftig EVE Market API direkt aufrufen
- ✅ **Schnellere Development** - Weniger Backend-Boilerplate
- ✅ **Einfacheres Testing** - Weniger Mock-Dependencies

### Negativ

- ❌ **XSS-Anfälligkeit** - Token exponiert bei XSS-Lücke (Mitigation: CSP, Sanitization)
- ❌ **Token Refresh Complexity** - Frontend muss Refresh-Logic implementieren
- ❌ **localStorage Limits** - 5-10MB Limit (für Token unkritisch)
- ❌ **Kein HttpOnly Cookie** - Token via JavaScript zugreifbar

### Risiken

- **XSS-Attacken:** Token Diebstahl möglich bei XSS-Lücke
  - **Mitigation:** Content Security Policy, React XSS Protection, Input Sanitization
- **Token Expiry UX:** User muss nach 20min re-authentifizieren
  - **Mitigation:** Token Refresh implementieren, UX für Re-Login optimieren
- **localStorage Clearing:** Browser-Clear kann User ausloggen
  - **Mitigation:** Akzeptiert - User kann neu einloggen

## Implementierung

### Phase 1: Frontend OAuth Library ✅ COMPLETED

**Dateien:**
- ✅ `frontend/src/lib/eve-sso.ts` (PKCE Functions, Token Exchange, Verification)
  - `generateState()` - CSRF Protection
  - `generateCodeVerifier()` - PKCE Verifier
  - `generateCodeChallenge()` - PKCE Challenge (SHA-256)
  - `buildAuthorizationUrl()` - EVE SSO Redirect URL
  - `exchangeCodeForToken()` - Token Exchange mit Code Verifier
  - `verifyToken()` - EVE ESI Token Verification
  - `validateState()` - State Parameter Validation
  - `TokenStorage` - localStorage Utilities

**Dependencies:**
- Web Crypto API (native, kein npm package)
- sessionStorage für State/Verifier (temporär)
- localStorage für Tokens (persistent)

---

### Phase 2: Frontend Components ✅ COMPLETED

**Dateien:**
- ✅ `frontend/src/app/callback/page.tsx` - OAuth Callback Handler
  - State Validation
  - Token Exchange
  - Character Info Verification
  - Event Dispatch für AuthContext
  - Error Handling

- ✅ `frontend/src/lib/auth-context.tsx` - Session State Management
  - Token Storage Check
  - EVE ESI Token Verification
  - Character Info State
  - Login/Logout Functions
  - getAuthHeader() für API Calls

- ✅ `frontend/src/components/eve-login-button.tsx` - Login UI
  - Initiiert OAuth Flow
  - Calls buildAuthorizationUrl()

- ✅ `frontend/src/components/character-info.tsx` - Character Display
  - Portrait (clickable → /character page)
  - Character Name
  - Logout Button

---

### Phase 3: Backend Token Validation ✅ COMPLETED

**Dateien:**
- ✅ `backend/pkg/evesso/verify.go` - Token Verification (63 Zeilen)
  - `VerifyToken(ctx, accessToken)` - EVE ESI /verify Call
  - `GetPortraitURL(characterID, size)` - Portrait URL Generator
  - `CharacterInfo` Struct

- ✅ `backend/pkg/evesso/middleware.go` - Bearer Token Middleware (44 Zeilen)
  - `AuthMiddleware(c *fiber.Ctx)` - Authorization Header Validation
  - Bearer Token Extraktion
  - Token Verification mit EVE ESI
  - Character Info in c.Locals speichern

**Total Backend Code:** 107 Zeilen (vs. 700 Zeilen bei Option 2)

---

### Phase 4: Protected API Endpoints ✅ COMPLETED

**Dateien:**
- ✅ `backend/cmd/api/main.go` - API Routes
  - `protected := api.Group("", evesso.AuthMiddleware)` - Protected Group
  - `GET /api/v1/character` - Character Info Endpoint
  - `GET /api/v1/trading/profit-margins` - Trading Endpoint (placeholder)
  - `GET /api/v1/manufacturing/blueprints` - Manufacturing Endpoint (placeholder)

---

### Validation Criteria

- [x] User kann mit EVE SSO einloggen
- [x] Character Portrait wird angezeigt
- [x] Character Name wird angezeigt
- [x] Logout funktioniert (Token wird gelöscht)
- [x] Protected API Endpoints erfordern Bearer Token
- [x] Ungültige Tokens werden abgelehnt (401 Unauthorized)
- [x] Token Verification mit EVE ESI funktioniert
- [x] PKCE Flow komplett (Code Verifier/Challenge)
- [x] State Parameter Validation (CSRF Protection)
- [x] Alle Tests passing (26/26 PASS)

**Status:** ✅ **Alle Kriterien erfüllt** (2025-10-26)

---

### Aufwand

| Phase | Geschätzt | Tatsächlich | Notizen |
|-------|-----------|-------------|---------|
| Phase 1: Frontend OAuth Library | 4h | 3h | PKCE straightforward |
| Phase 2: Frontend Components | 3h | 2h | React Context einfach |
| Phase 3: Backend Verification | 2h | 1h | Nur Token Validation |
| Phase 4: Protected Endpoints | 1h | 0.5h | Middleware setup simpel |
| **Total (Option 1)** | **10h** | **6.5h** | ✅ 35% schneller als geschätzt |
| **Option 2 (Backend OAuth)** | **15h** | **~10h** | ❌ Wurde verworfen nach Implementation |

**Verschwendeter Aufwand (Backend OAuth):** ~10h (Implementation + Debugging + Refactoring)

---

### Abhängigkeiten

- **ADRs:** 
  - ADR-001 (Tech Stack: Next.js + Go) - Frontend SPA → Frontend OAuth sinnvoll
- **Issues:** 
  - Issue #3 "EVE SSO Login/Logout Integration" - Ursprüngliche Spezifikation
- **Externe Faktoren:**
  - EVE SSO PKCE Support (verfügbar seit 2020)
  - Web Crypto API Browser Support (alle modernen Browser)

## Referenzen

**EVE SSO Dokumentation:**
- [EVE SSO Overview](https://docs.esi.evetech.net/docs/sso/)
- [EVE SSO PKCE Support](https://developers.eveonline.com/blog/article/sso-to-authenticated-calls)

**OAuth2 Standards:**
- [RFC 6749: OAuth 2.0 Authorization Framework](https://datatracker.ietf.org/doc/html/rfc6749)
- [RFC 7636: PKCE (Proof Key for Code Exchange)](https://datatracker.ietf.org/doc/html/rfc7636)

**Best Practices:**
- [OAuth 2.0 for Browser-Based Apps (IETF Draft)](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-browser-based-apps)
- [OWASP: OAuth 2.0 Security Best Practices](https://cheatsheetseries.owasp.org/cheatsheets/OAuth2_Cheat_Sheet.html)

**Implementation Guides:**
- [Next.js Authentication Patterns](https://nextjs.org/docs/app/building-your-application/authentication)
- [React Context API Best Practices](https://react.dev/reference/react/useContext)

**Related ADRs:**
- ADR-001: Tech Stack (Go Backend + Next.js Frontend)
- ADR-002: API Design (tRPC vs. OpenAPI) - TODO
- ADR-003: Deployment-Strategie - TODO

**Related Issues:**
- Issue #3: EVE SSO Login/Logout Integration (OAuth2)

**Post-Mortem:**
- `tmp/issue-3-analyse.md` - Detaillierte Analyse warum Backend OAuth ursprünglich implementiert wurde

## Notizen

### Lessons Learned (aus Issue #3 Post-Mortem)

**Was schief ging:**
1. Issue #3 spezifizierte Backend-OAuth als impliziten Standard (ohne ADR)
2. Flow-Diagramm zeigte Backend als Orchestrator
3. Endpoints suggestierten Backend-Verantwortung (/auth/login, /auth/callback)
4. Environment Variables enthielten Backend OAuth-Secrets (EVE_CLIENT_SECRET, JWT_SECRET)
5. Keine Architektur-Entscheidung zwischen Frontend vs. Backend OAuth

**Impact:**
- ~700 Zeilen unnötiger Backend-Code implementiert
- ~10 Stunden verschwendete Entwicklungszeit
- Vollständiges Refactoring nötig am nächsten Tag

**Verbesserungen:**
- ✅ ADR-004 erstellt (diese Datei) - Dokumentiert Architektur-Entscheidung
- ✅ Issue Template erweitert mit Architektur-Sektion (siehe .github/ISSUE_TEMPLATE/)
- ✅ Post-Mortem Analyse erstellt (tmp/issue-3-analyse.md)

### Security Considerations (PKCE vs Client Secret)

**PKCE Vorteile:**
- Kein Secret Management (kein .env, kein .gitignore Risk)
- Public Client Best Practice (OAuth 2.0 for Browser-Based Apps)
- Schutz gegen Authorization Code Interception

**Client Secret Nachteile:**
- Secret muss sicher gespeichert werden (Backend .env)
- Secret Rotation kompliziert (Multi-Instance Deployments)
- Secret Leak Risiko (Git-History, Logs, Dumps)

**PKCE Security:**
- Code Verifier: 256-bit random (crypto.getRandomValues)
- Code Challenge: SHA-256 Hash des Verifiers
- Verifier wird nur einmal verwendet (sessionStorage, nach Token Exchange gelöscht)
- State Parameter zusätzlich gegen CSRF

### Future Enhancements

**Token Refresh (TODO):**
- EVE Tokens laufen nach 20min ab
- Refresh Token kann für neue Access Tokens verwendet werden
- Implementation: `refreshToken()` Function in eve-sso.ts
- UX: Automatisches Silent Refresh im Hintergrund

**Token Revocation (TODO):**
- EVE SSO Revocation Endpoint für explizite Token-Invalidierung
- Beim Logout: Token revoken statt nur lokal löschen
- Implementation: `revokeToken()` Function

**Multi-Character Support (Future):**
- Mehrere EVE Characters pro User
- Token Storage pro Character (localStorage Array)
- Character Switcher UI

**ESI Direct Calls (Future):**
- Frontend kann direkt EVE ESI Market API aufrufen
- Kein Backend Proxy nötig für öffentliche ESI Endpoints
- Reduces Backend Load

---

**Change Log:**

- 2025-10-26: Status auf Accepted gesetzt (Development Team)
- 2025-10-26: Implementation completed, alle Validierung Criteria erfüllt
- 2025-10-26: Post-Mortem Analyse verlinkt (tmp/issue-3-analyse.md)
- 2025-10-26: Umbenannt von ADR-003 zu ADR-004 (ADR-002/003 bereits in ADR-001 reserviert)
