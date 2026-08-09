package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime/debug"
)

// RecoveryMiddleware catches any panic from a handler so one bad request
// can't take down the whole server (net/http already recovers per-connection
// so this isn't a crash fix), and turns it into a structured log line plus
// a plain 500 JSON response instead of a silently dropped connection.
//
// This should be the outermost middleware, wrapping everything else,
// so a panic anywhere downstream (including in other middleware) is caught.
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf(
					"[PANIC RECOVERED] method=%s path=%s error=%v\n%s",
					r.Method, r.URL.Path, rec, debug.Stack(),
				)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"detail": "Internal Server Error"})
			}
		}()

		next.ServeHTTP(w, r)
	})
}
