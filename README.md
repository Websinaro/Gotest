# kabackend (Go port of the FastAPI backend)

Same API, same routes, same request/response JSON shapes, same AES-GCM
envelope encryption, same version-gate header — rewritten from Python/FastAPI
to Go. No endpoint paths, field names, or client-visible behavior changed.

## Build & run

```bash
cp .env.example .env   # fill in real values
go mod tidy             # fetches golang.org/x/crypto, github.com/lib/pq, and
                         # github.com/coder/websocket, and regenerates go.sum
                         # with verified checksums
go build -o kabackend .
./kabackend              # listens on $PORT, default 8000
```

`go mod tidy` needs network access once. The `github.com/lib/pq` and
`golang.org/x/crypto`/`golang.org/x/sys` lines in the go.sum shipped here
were written by hand (this environment has no Go toolchain or internet
access to run the Go module proxy) — **run `go mod tidy` before your first
build** to replace them with real, verified checksums. The
`github.com/coder/websocket` line is already a real, verified checksum
(pinned at v1.8.14) since it could be sourced from the public checksum
database without needing a live `go get`; `go mod tidy` will simply confirm
it rather than rewrite it.

## Dependency footprint

Three third-party packages: `github.com/lib/pq` (Postgres driver, pure Go,
zero transitive deps), `golang.org/x/crypto` (for `argon2.IDKey`, which pulls
in `golang.org/x/sys` transitively via blake2b's CPU-feature detection), and
`github.com/coder/websocket` (RFC 6455 WebSocket implementation used for the
SOS live-location socket — zero dependencies of its own; see "WebSocket
implementation" below). Everything else that the Python version got from
libraries — JWT, AES-GCM, Google OAuth2 for Firebase push — is implemented
directly against Go's standard library:

- `security/jwt.go` — hand-rolled HS256 JWT (was `python-jose`)
- `security/crypto.go` — `crypto/aes` + `crypto/cipher` GCM (was `cryptography`)
- `security/hashing.go` — `golang.org/x/crypto/argon2` (argon2id), encoded
  in the same PHC string format passlib's argon2 handler uses
  (`$argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>`, same defaults: 64 MiB
  memory, 3 iterations, 4 lanes) — was `passlib[argon2]`
- `services/firebase_auth.go` — hand-rolled Google service-account
  JWT-bearer OAuth2 flow + FCM HTTP v1 REST calls (was `firebase-admin`)
- Routing uses Go 1.22's built-in `net/http.ServeMux` method+path patterns
  (was FastAPI/Starlette routing) — no web framework dependency.

## WebSocket implementation

`services/websocket.go` is now a thin wrapper around
[`github.com/coder/websocket`](https://github.com/coder/websocket) (the
actively maintained successor to `nhooyr.io/websocket`) instead of a
hand-rolled RFC 6455 frame parser. It exposes the same small `WSConn` API
(`UpgradeWebSocket` / `ReadMessage` / `WriteText` / `WritePing` / `Close`)
that `services/ws_hub.go` and `routes/sos_ws.go` were already written
against, so nothing outside that one file changed. Swapping in
`coder/websocket` gets `context.Context`-aware reads/writes, a spec-correct
close handshake, concurrent-write safety, and Autobahn-tested framing
without hand-maintaining a frame parser. `gorilla/websocket` is also a solid
production-grade option if you'd rather standardize on it instead — the same
`WSConn` wrapper shape makes that swap a similarly contained change limited
to this one file.

## Not ported

`routes/official_alerts.py` from the original repo is dead code — it's
never imported by `main.py` and references an undefined `model.OfficialAlert`,
so it can't run in the original app either. Excluded here to match actual
runtime behavior.

## Production hardening added beyond a literal port

Two things Go doesn't give you for free the way FastAPI/SQLAlchemy's
defaults did, now handled explicitly:

- **DB connection pool limits** (`database/database.go`) — capped at 15 max
  open / 5 idle connections by default, matching SQLAlchemy's
  `pool_size=5, max_overflow=10`. Without this, Go's `database/sql` defaults
  to unlimited open connections, so a traffic spike could open far more
  Postgres connections than the Python service ever would and exhaust the
  database's connection limit. Tune via `DB_MAX_OPEN_CONNS` /
  `DB_MAX_IDLE_CONNS` / `DB_CONN_MAX_LIFETIME_MIN`.
- **Panic recovery** (`middleware/recovery_middleware.go`) — outermost
  middleware; catches any panic from a handler, logs the method/path/stack
  trace, and returns a plain `{"detail": "Internal Server Error"}` 500
  instead of the connection just dying. `net/http` already recovers per
  connection so a panic couldn't crash the whole server either way, but
  this adds the structured logging needed to actually notice and debug it.
- **CORS** (`middleware/cors_middleware.go`) — sets
  `Access-Control-Allow-*` headers and answers preflight `OPTIONS` requests
  directly (the route mux has no handler registered for `OPTIONS`, so
  without this a browser preflight would just 404). Allowed origins come
  from `CORS_ALLOWED_ORIGINS` (comma-separated; `*` by default).
- **Fail-fast config** (`config/config.go`) — `DATABASE_URL`, `SECRET_KEY`,
  `AES_SECRET_KEY`, and `PRESIDENT_ACCESS_CODE` are now required: the
  process calls `log.Fatal` at startup if any is missing, rather than
  booting with an empty JWT signing key, un-decryptable AES key, or (worst
  case) the hardcoded `KDMA-PRESIDENT-2026` fallback silently still active
  in production. `AES_SECRET_KEY` is also checked to decode to a valid
  16/24/32-byte AES key at startup instead of only failing on first use.
