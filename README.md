# kabackend (Go port of the FastAPI backend)

Same API, same routes, same request/response JSON shapes, same AES-GCM
envelope encryption, same version-gate header — rewritten from Python/FastAPI
to Go. No endpoint paths, field names, or client-visible behavior changed.

## Build & run

```bash
cp .env.example .env   # fill in real values
go mod tidy             # fetches the one dependency (github.com/lib/pq) and
                         # regenerates go.sum with verified checksums
go build -o kabackend .
./kabackend              # listens on $PORT, default 8000
```

`go mod tidy` needs network access once. The go.sum shipped here was written
by hand (this environment has no Go toolchain or internet access to run the
Go module proxy), so **run `go mod tidy` before your first build** — it will
just confirm/refresh the checksum for `lib/pq`, nothing else, since that's
the only external dependency in the whole project.

## Dependency footprint

Deliberately minimal: **one** third-party package (`github.com/lib/pq`, the
Postgres driver — pure Go, zero transitive dependencies). Everything else
that the Python version got from libraries — JWT, AES-GCM, password hashing,
Google OAuth2 for Firebase push — is implemented directly against Go's
standard library:

- `security/jwt.go` — hand-rolled HS256 JWT (was `python-jose`)
- `security/crypto.go` — `crypto/aes` + `crypto/cipher` GCM (was `cryptography`)
- `security/hashing.go` + `pbkdf2.go` — PBKDF2-HMAC-SHA256 password hashing
  (was `passlib[argon2]`; algorithm changed because there's no dependency-free
  argon2 in the stdlib — verify/hash behavior is otherwise identical)
- `services/firebase_auth.go` — hand-rolled Google service-account
  JWT-bearer OAuth2 flow + FCM HTTP v1 REST calls (was `firebase-admin`)
- Routing uses Go 1.22's built-in `net/http.ServeMux` method+path patterns
  (was FastAPI/Starlette routing) — no web framework dependency.

## Not ported

`routes/official_alerts.py` from the original repo is dead code — it's
never imported by `main.py` and references an undefined `model.OfficialAlert`,
so it can't run in the original app either. Excluded here to match actual
runtime behavior.
