# VPN Management Backend

Control-plane API for the VPN admin dashboard, written in Go + Gin +
PostgreSQL. It manages administrators, VPN users, devices, access keys,
servers, usage, and audit logs — it never carries VPN traffic itself.

```
Admin Frontend (Next.js) ---> Go/Gin API (:8080) ---> PostgreSQL
                                     |
                                     +--> Outline Management API ---> Shadowsocks
```

## Requirements

- Go 1.25+
- PostgreSQL 16+
- An Outline server (https://github.com/Jigsaw-Code/outline-server) for the
  actual Shadowsocks data plane. In development you can leave
  `OUTLINE_API_URL` unset — access-key operations will fail loudly, but
  everything else (auth, users, devices, servers, audit logs) works.

## Getting started

```bash
cp .env.example .env      # then fill in DATABASE_URL, JWT_SECRET, etc.
go run ./cmd/server
```

Generate the two required secrets:

```bash
openssl rand -base64 48   # JWT_SECRET
openssl rand -base64 32   # SECRET_ENCRYPTION_KEY (must decode to exactly 32 bytes)
```

Migrations in `migrations/*.up.sql` run automatically on startup (tracked
in a `schema_migrations` table, safe to re-run).

### First login

Admins are **never** auto-created from an arbitrary Google sign-in. Seed
the first one:

```bash
psql "$DATABASE_URL" \
  -v google_sub="'<your-google-sub>'" \
  -v email="'you@example.com'" \
  -v name="'Your Name'" \
  -f scripts/seed-admin.sql
```

Find your Google `sub` by inspecting the `id_token` JWT payload from a
sign-in attempt (e.g. paste it into https://jwt.io — the token itself is
not secret once it has expired).

## Running with Docker

```bash
docker compose up --build
```

PostgreSQL is bound to `127.0.0.1:5432` only — never exposed publicly. The
API is bound to `127.0.0.1:8080`; put Caddy in front of it for TLS (see
`scripts/Caddyfile.example`).

## Testing

```bash
go test ./...
```

Tests run against an in-memory SQLite database (`internal/testutil`) and a
fake `VPNProvider`, so they need neither PostgreSQL nor a real Outline
server.

## Architecture

```
cmd/server        entrypoint: config, DB, migrations, DI, graceful shutdown
internal/config    env var loading
internal/database  PostgreSQL connection + migration runner
internal/models    GORM models
internal/repositories  data access per aggregate
internal/services  business logic (auth, users, devices, access keys, servers, usage, audit)
internal/handlers  Gin handlers + router + role-based route wiring
internal/middleware  auth, CORS, rate limiting, security headers, logging, recovery
internal/vpn       VPNProvider interface + OutlineShadowsocksProvider + config/QR generation
internal/auth      Google ID token verification + backend session JWTs
internal/crypto    AES-256-GCM envelope for access-key secrets at rest
internal/metrics   Prometheus counters/gauges
pkg/response       consistent {data, error, meta} JSON envelope
migrations/        versioned .up.sql/.down.sql pairs
docs/openapi.yaml  API reference
```

## Authentication model

1. The frontend runs Google OAuth itself (via Auth.js) and gets a Google
   ID token.
2. It POSTs that token to `POST /api/v1/auth/google`. This backend verifies
   the token's signature against Google's public keys, looks up the admin
   **only** by `google_sub`, and rejects anything that isn't an
   already-provisioned, `ACTIVE` admin.
3. On success, the backend issues its own short-lived JWT (`JWT_EXPIRY`,
   default 24h) containing only the admin's ID — never a role or status.
4. Every subsequent request carries `Authorization: Bearer <token>`.
   `RequireAuth` re-loads the admin from PostgreSQL on **every request**,
   so disabling an admin takes effect immediately rather than waiting for
   their token to expire.
5. `RequireRole` authorizes purely against that freshly-loaded role. A
   role claimed by the client is never consulted.

Sessions are stateless JWTs (no server-side revocation list) — an
acceptable tradeoff for a single-VPS internal admin tool per the resource
strategy in the project prompt (no Redis). If revocation-on-demand becomes
a requirement, add a `revoked_sessions` table before reaching for Redis.

## VPN provider abstraction

`internal/vpn.VPNProvider` is implemented today by
`OutlineShadowsocksProvider`, an HTTP client for Outline's Management REST
API. This backend never implements Shadowsocks framing, ciphers, or TLS
itself — it only orchestrates that mature, already-deployed
implementation. The Outline management API is served on a self-signed
certificate; instead of trusting a CA (there is none), the client pins the
certificate's SHA-256 fingerprint (`OUTLINE_API_CERT_SHA256`), exactly like
the official Outline Manager app does.

Access-key passwords returned by Outline are encrypted at rest
(`internal/crypto`, AES-256-GCM) and only decrypted in memory when an
authorized administrator explicitly requests
`GET /access-keys/:id/config` or `/qr`. They are never logged.

## Deployment (single DigitalOcean VPS)

```
DigitalOcean VPS (Ubuntu 24.04, Singapore)
├── Caddy            :443  (TLS, WSS proxying — see scripts/Caddyfile.example)
├── Go/Gin API        :8080 (127.0.0.1 only)
├── PostgreSQL        :5432 (127.0.0.1 only)
└── Outline server    (Shadowsocks data plane)
```

DNS:

```
vpn.thestrm.space     -> DigitalOcean IP
api.vpn.thestrm.space -> DigitalOcean IP
```

Do not introduce Kubernetes, Redis, managed PostgreSQL, or microservices
until actual load requires it.
