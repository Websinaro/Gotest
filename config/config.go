package config

import (
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
	// normal "user". Set this to a real secret via the environment in
	// production - the fallback here only exists so the app still runs
	// out of the box in dev.
	PresidentAccessCode string
)

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func init() {
	loadDotEnv()

	DatabaseURL = os.Getenv("DATABASE_URL")
	SecretKey = os.Getenv("SECRET_KEY")
	Algorithm = getEnv("ALGORITHM", "HS256")

	minutes, err := strconv.Atoi(getEnv("ACCESS_TOKEN_EXPIRES_MINUTES", "60"))
	if err != nil {
		minutes = 60
	}
	AccessTokenExpiresMinutes = minutes

	AESSecretKey = os.Getenv("AES_SECRET_KEY")

	MinSupportedVersion = getEnv("MIN_SUPPORTED_VERSION", "2.2.8")
	LatestVersion = getEnv("LATEST_VERSION", "3.0.1")
	FirebaseCredentialsB64 = os.Getenv("FIREBASE_CREDENTIALS_B64")

	PresidentAccessCode = getEnv("PRESIDENT_ACCESS_CODE", "KDMA-PRESIDENT-2026")
}
