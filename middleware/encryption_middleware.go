package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"kabackend/security"
)

// Paths that must stay plain JSON — Swagger/OpenAPI need to read these
// directly, and "/" is a manual health-check people hit in a browser.
var encryptionExcludedPaths = map[string]bool{
	"/docs":        true,
	"/redoc":       true,
	"/openapi.json": true,
	"/":            true,
	"/app/version": true,
}

// responseBuffer captures everything a handler writes so the middleware can
// inspect/replace the body afterwards, mirroring how Starlette's
// BaseHTTPMiddleware lets encryption_middleware.py buffer call_next's
// response before deciding whether to re-wrap it.
type responseBuffer struct {
	header      http.Header
	statusCode  int
	body        bytes.Buffer
	wroteHeader bool
}

func newResponseBuffer() *responseBuffer {
	return &responseBuffer{header: make(http.Header), statusCode: http.StatusOK}
}

func (b *responseBuffer) Header() http.Header { return b.header }

func (b *responseBuffer) WriteHeader(statusCode int) {
	if !b.wroteHeader {
		b.statusCode = statusCode
		b.wroteHeader = true
	}
}

func (b *responseBuffer) Write(p []byte) (int, error) {
	return b.body.Write(p)
}

// EncryptionMiddleware mirrors middleware/encryption_middleware.py.
func EncryptionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if encryptionExcludedPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}

		// WebSocket upgrades must reach the handler with the original
		// http.ResponseWriter intact so it can be type-asserted to
		// http.Hijacker - the responseBuffer below doesn't implement that
		// interface, so wrapping it here would make every /ws/ connection
		// fail to upgrade. WS payloads aren't run through this JSON
		// encrypt/decrypt envelope at all; they rely on wss:// (TLS) for
		// transport security, same as any other WebSocket deployment.
		if strings.HasPrefix(r.URL.Path, "/ws/") {
			next.ServeHTTP(w, r)
			return
		}

		if (r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch) &&
			strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			bodyBytes, err := io.ReadAll(r.Body)
			if err == nil && len(bodyBytes) > 0 {
				var wrapper map[string]interface{}
				if err := json.Unmarshal(bodyBytes, &wrapper); err == nil {
					if dataField, ok := wrapper["data"]; ok {
						if dataStr, ok := dataField.(string); ok {
							if decrypted, err := security.DecryptPayload(dataStr); err == nil {
								newBody, err := json.Marshal(decrypted)
								if err == nil {
									r.Body = io.NopCloser(bytes.NewReader(newBody))
									r.ContentLength = int64(len(newBody))
								}
							}
							// on decrypt error, fall through silently, matching
							// the Python `except Exception: pass`
						}
					} else {
						r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
					}
				} else {
					r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				}
			}
		}

		buf := newResponseBuffer()
		next.ServeHTTP(buf, r)

		status := buf.statusCode
		contentType := buf.header.Get("Content-Type")

		if status >= 200 && status < 300 && strings.HasPrefix(contentType, "application/json") {
			var data interface{}
			if err := json.Unmarshal(buf.body.Bytes(), &data); err == nil {
				encrypted, err := security.EncryptPayload(data)
				if err == nil {
					wrapped, _ := json.Marshal(map[string]string{"data": encrypted})
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(status)
					w.Write(wrapped)
					return
				}
			}
			// fall through to plain passthrough on any encode/encrypt failure
		}

		for k, values := range buf.header {
			for _, v := range values {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(status)
		w.Write(buf.body.Bytes())
	})
}
