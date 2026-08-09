package middleware

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"kabackend/config"
)

var versionExemptPaths = map[string]bool{
	"/":            true,
	"/docs":        true,
	"/redoc":       true,
	"/openapi.json": true,
	"/app/version": true,
}

// wsPathPrefix mirrors the same exemption in EncryptionMiddleware. A
// WebSocket handshake is issued by the platform's WS client (not our own
// http.Client wrapper), so it won't reliably carry the custom
// X-App-Version header the way a normal REST call does - don't block the
// SOS live-location socket over a header a WS client can't easily set.
const wsPathPrefix = "/ws/"

// compareVersions returns -1, 0, or 1 comparing dot-separated numeric
// version strings (e.g. "2.2.8" vs "3.0.1"), mirroring the ordering
// behaviour of packaging.version.Version for these simple version strings.
// A malformed component is treated the same as the Python code's
// InvalidVersion case: the whole comparison is skipped (returns 0, "equal",
// so the request is let through rather than blocked).
func compareVersions(a, b string) (int, bool) {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	n := len(aParts)
	if len(bParts) > n {
		n = len(bParts)
	}

	for i := 0; i < n; i++ {
		var av, bv int
		var err error
		if i < len(aParts) {
			av, err = strconv.Atoi(aParts[i])
			if err != nil {
				return 0, false
			}
		}
		if i < len(bParts) {
			bv, err = strconv.Atoi(bParts[i])
			if err != nil {
				return 0, false
			}
		}
		if av != bv {
			if av < bv {
				return -1, true
			}
			return 1, true
		}
	}
	return 0, true
}

// VersionCheckMiddleware mirrors middleware/version_middleware.py.
func VersionCheckMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if versionExemptPaths[r.URL.Path] || strings.HasPrefix(r.URL.Path, wsPathPrefix) {
			next.ServeHTTP(w, r)
			return
		}

		appVersion := r.Header.Get("x-app-version")
		if appVersion == "" {
			appVersion = r.Header.Get("X-App-Version")
		}

		if appVersion != "" {
			cmp, ok := compareVersions(appVersion, config.MinSupportedVersion)
			if ok && cmp < 0 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(426) // 426 Upgrade Required
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":                "update_required",
					"message":              config.ForceUpdateMessage,
					"min_supported_version": config.MinSupportedVersion,
					"latest_version":        config.LatestVersion,
				})
				return
			}
			// malformed header - let it through rather than block a real user
		}

		next.ServeHTTP(w, r)
	})
}
