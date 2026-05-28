# eve-o-provit Backend — Hetzner Production Deploy Runbook

Onboards the **backend** onto the shared `backend` host (cx23, 46.225.188.34) behind
`edge-caddy`, per the [edge-caddy single-edge ADR](https://github.com/Sternrassler/brain/blob/main/docs/superpowers/specs/2026-05-20-adr-edge-caddy-single-entry.md).

- **Domain:** `eveonline.sternrassler.de` (DNS A/AAAA already registered)
- **App key (deploy-app.sh / apps-config):** `eveoprovit` (no hyphens — `EVEOPROVIT_TAG`)
- **Containers:** `eve-o-provit-api`, `eve-o-provit-postgres`, `eve-o-provit-redis`
- **Image:** `ghcr.io/sternrassler/eve-o-provit-backend:<sha>`
- **Validated locally:** image builds (CGO/sqlite, linux/amd64); full stack boots with
  `/api/v1/health` → `{postgres:ok, sde:ok}` and real SDE region data.

## Why this app is special

Unlike the other portfolio apps (SQLite-only), eve-o-provit hard-requires **Postgres**
(`DATABASE_URL`, market data) + **Redis** (ESI cache) + a **~474 MB SDE SQLite** mounted
read-only. Postgres/Redis are scoped to this app inside the shared `apps` compose;
`deploy-app.sh eveoprovit` recreates **only** the api, never the DB.

## Prerequisites (one-time, need operator)

1. **GHCR push credentials** — GitHub PAT with `write:packages`.
2. **SSH access** to `root@46.225.188.34` (keys `ix-local` / `mac-local`).
3. **Hetzner infra repo** branch `feat/onboard-eve-o-provit` merged to `main` →
   CD scps `compose.yaml` + `Caddyfile` + `deploy-app.sh` to `/opt/apps/` and reloads Caddy.

## Step 1 — Build & push image

```sh
cd eve-o-provit/backend
TAG="sha-$(git rev-parse --short HEAD)"
docker build -t ghcr.io/sternrassler/eve-o-provit-backend:$TAG \
             --build-arg APP_VERSION=$TAG .
echo "$GHCR_PAT" | docker login ghcr.io -u Sternrassler --password-stdin
docker push ghcr.io/sternrassler/eve-o-provit-backend:$TAG
```

## Step 2 — Merge hetzner infra branch

Merge `feat/onboard-eve-o-provit` (in `~/vscode/hetzner`) to `main`. CD syncs the new
compose services + Caddy block to the server. (Caddy reload provisions the TLS cert for
`eveonline.sternrassler.de` on first request.)

## Step 3 — Provision secrets on server

Create `/opt/apps/env/eve-o-provit.env` (mode 600, root). Store the master copy in KeePass.

Use [`.env.example`](.env.example) as the template. Production uses the dedicated EVE
application "EVE Profit Maximirer (prod)" (client_id `4d0514c205af4e898b0569badd59a38e`),
with `https://eveonline.sternrassler.de/callback` registered in its Callback URL list.

```sh
ssh root@46.225.188.34 'umask 077; cat > /opt/apps/env/eve-o-provit.env' <<'EOF'
EVE_CLIENT_ID=4d0514c205af4e898b0569badd59a38e
POSTGRES_PASSWORD=<generate: openssl rand -hex 24>
REDIS_PASSWORD=<generate: openssl rand -hex 24>
DATABASE_URL=postgres://eveprovit:<POSTGRES_PASSWORD>@eve-o-provit-postgres:5432/eveprovit?sslmode=disable
REDIS_URL=redis://:<REDIS_PASSWORD>@eve-o-provit-redis:6379/0
CORS_ORIGINS=https://eveonline.sternrassler.de
EVE_CALLBACK_URL=https://eveonline.sternrassler.de/callback
ESI_USER_AGENT=eve-o-provit/0.1.0 (sternrassler@gmail.com)
# COOKIE_SECURE intentionally unset → defaults to true (fail-closed) over HTTPS.
EOF
```

> `POSTGRES_PASSWORD` and the password embedded in `DATABASE_URL` must match (env_file does
> not cross-interpolate). Same for `REDIS_PASSWORD` / `REDIS_URL`.

Register the `EVEOPROVIT_TAG` line so `deploy-app.sh`'s `sed` can update it:

```sh
ssh root@46.225.188.34 "grep -q '^EVEOPROVIT_TAG=' /opt/apps/.env || echo 'EVEOPROVIT_TAG=$TAG' >> /opt/apps/.env"
```

## Step 4 — Seed the SDE into its volume

```sh
docker --context ... volume create apps_eve_o_provit_sde   # or let compose create it
# copy the 474MB SDE from local repo into the named volume
scp eve-o-provit/backend/data/sde/eve-sde.db root@46.225.188.34:/tmp/eve-sde.db
ssh root@46.225.188.34 'docker run --rm -v apps_eve_o_provit_sde:/dst -v /tmp/eve-sde.db:/src.db:ro \
    alpine sh -c "cp /src.db /dst/eve-sde.db" && rm -f /tmp/eve-sde.db'
```

## Step 5 — First deploy (manual)

```sh
ssh root@46.225.188.34 'cd /opt/apps && docker compose up -d \
    eve-o-provit-postgres eve-o-provit-redis eve-o-provit-api'
```

No manual schema step: the backend applies `backend/migrations/*.up.sql` on startup
(single schema source, idempotent — see ADR `2026-05-28-single-schema-source`). The
old `init-db/01-init.sql` is gone.

## Step 6 — Verify

```sh
ssh root@46.225.188.34 'curl -fsS http://localhost:8080/api/v1/health'      # in-container? use service net
curl -fsS https://eveonline.sternrassler.de/api/v1/health   # → {"status":"ok",...}
curl -fsS https://eveonline.sternrassler.de/api/v1/version
curl -fsS 'https://eveonline.sternrassler.de/api/v1/sde/regions' | head -c 300
```

## Follow-ups (not done here)

- **Backup:** `apps-backup.sh` is SQLite-`.backup`-based and would corrupt a live Postgres
  data dir. eve-o-provit needs a **`pg_dump`** path before it joins the nightly backup.
  Currently only health monitoring is wired (`apps-config.sh`).
- **CD:** ✅ Done — `.github/workflows/{ci,deploy,smoke-test}.yml`. Pushing a SemVer tag
  (`vX.Y.Z`) builds+pushes the image to GHCR and runs `deploy-app.sh eveoprovit <tag>`.
  The manual steps above are only needed for the very first bring-up / infra changes.
- **RAM:** `backend` host is 4 GB and already runs 6 containers. Watch memory after adding
  Postgres + Redis + the api.
- **SSO login** is verified against the backend's `POST /auth/callback` (PKCE exchange +
  JWKS validation + cookie session). The browser-side `/callback` page needs the frontend;
  backend-only deploy serves health/SDE/market/public + the auth API.
