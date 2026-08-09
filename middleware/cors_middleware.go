package middleware

import (
	"net/http"
	"strings"

	"kabackend/config"
)

// CORSMiddleware sets Access-Control-* headers and short-circuits preflight
// OPTIONS requests. Allowed origins come from config.CorsAllowedOrigins
// (CORS_ALLOWED_ORIGINS env var, comma-separated; "*" allows any origin).
//
// This should sit close to the outside of the middleware chain, ahead of
// routing, since a browser's preflight OPTIONS request has no route
// registered for it in main.go's mux and must be answered here directly.
func CORSMiddleware(next http.Handler) http.Handler {
	allowAll := strings.TrimSpace(config.CorsAllowedOrigins) == "*"

	allowed := map[string]bool{}
	if !allowAll {
		for _, origin := range strings.Split(config.CorsAllowedOrigins, ",") {
			origin = strings.TrimSpace(origin)
			if origin != "" {
				allowed[origin] = true
			}
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin != "" && (allowAll || allowed[origin]) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-app-version")
			w.Header().Set("Access-Control-Max-Age", "600")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
