package config

import (
	"encoding/base64"
	"log"
	"os"
	"strconv"
)

var (
	DatabaseURL               string
	SecretKey                 string
	Algorithm                 string
	AccessTokenExpiresMinutes int
	AESSecretKey               string

	MinSupportedVersion string
	LatestVersion        string
	ForceUpdateMessage   = "This version of WeBAlert is no longer supported. Please update latest version to continue receiving weather and disaster alerts."
	FirebaseCredentialsB64 string

	// Official access code an applicant must supply at signup to be
	// registered as "president" (state coordinator / admin) instead of a
	// normal "user". Must be set explicitly via the environment - see
	// requireEnv below. There is deliberately no hardcoded fallback: a
	// silent default here would mean anyone reading this source (or
	// guessing a well-known placeholder) could register themselves as an
	// admin against a deployment that forgot to override it.
	PresidentAccessCode string

	// Comma-separated list of origins allowed to make cross-origin
	// requests (e.g. "https://app.example.com,https://admin.example.com").
	// Defaults to "*" (allow any origin) so the API keeps working
	// out-of-the-box for a mobile app / same-origin setups; tighten this
	// in production if the API is also called from browser JS.
	CorsAllowedOrigins string
)

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

// requireEnv fails fast at startup rather than letting the service boot
// with an empty/missing secret and fail confusingly (or "succeed"
// insecurely) on the first request that needs it.
func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing required environment variable: %s", key)
	}
	return v
}

func init() {
	loadDotEnv()

	DatabaseURL = requireEnv("DATABASE_URL")
	SecretKey = requireEnv("SECRET_KEY")
	Algorithm = getEnv("ALGORITHM", "HS256")

	minutes, err := strconv.Atoi(getEnv("ACCESS_TOKEN_EXPIRES_MINUTES", "60"))
	if err != nil {
		minutes = 60
	}
	AccessTokenExpiresMinutes = minutes

	AESSecretKey = requireEnv("AES_SECRET_KEY")
	decoded, err := base64.StdEncoding.DecodeString(AESSecretKey)
	if err != nil {
		log.Fatalf("AES_SECRET_KEY is not valid base64: %v", err)
	}
	switch len(decoded) {
	case 16, 24, 32:
		// valid AES-128/192/256 key length
	default:
		log.Fatalf("AES_SECRET_KEY must decode to 16, 24, or 32 bytes (got %d) - generate with `openssl rand -base64 32`", len(decoded))
	}

	MinSupportedVersion = getEnv("MIN_SUPPORTED_VERSION", "2.2.8")
	LatestVersion = getEnv("LATEST_VERSION", "3.0.1")
	FirebaseCredentialsB64 = os.Getenv("FIREBASE_CREDENTIALS_B64")

	PresidentAccessCode = requireEnv("PRESIDENT_ACCESS_CODE")

	CorsAllowedOrigins = getEnv("CORS_ALLOWED_ORIGINS", "*")
}
