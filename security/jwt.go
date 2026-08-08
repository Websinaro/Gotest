package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"kabackend/config"
)

var jwtHeaderB64 = base64URLEncode([]byte(`{"alg":"HS256","typ":"JWT"}`))

func base64URLEncode(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

// CreateAccessToken mirrors create_access_token(data, expires_delta) in
// security/jwt.py. data currently only ever carries {"sub": email}.
func CreateAccessToken(data map[string]interface{}, expiresDelta *time.Duration) (string, error) {
	claims := map[string]interface{}{}
	for k, v := range data {
		claims[k] = v
	}

	var expiry time.Duration
	if expiresDelta != nil {
		expiry = *expiresDelta
	} else {
		expiry = time.Duration(config.AccessTokenExpiresMinutes) * time.Minute
	}
	claims["exp"] = time.Now().UTC().Add(expiry).Unix()

	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payloadB64 := base64URLEncode(payloadBytes)

	signingInput := jwtHeaderB64 + "." + payloadB64
	mac := hmac.New(sha256.New, []byte(config.SecretKey))
	mac.Write([]byte(signingInput))
	signature := base64URLEncode(mac.Sum(nil))

	return signingInput + "." + signature, nil
}

// DecodeAccessToken mirrors decode_access_token(token) in security/jwt.py:
// returns the claims map, or nil if the token is invalid/expired/malformed
// (matching the Python function returning None on JWTError).
func DecodeAccessToken(token string) map[string]interface{} {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil
	}

	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(config.SecretKey))
	mac.Write([]byte(signingInput))
	expectedSig := mac.Sum(nil)

	actualSig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil
	}
	if subtle.ConstantTimeCompare(expectedSig, actualSig) != 1 {
		return nil
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil
	}

	if expRaw, ok := claims["exp"]; ok {
		exp, ok := expRaw.(float64)
		if !ok {
			return nil
		}
		if time.Now().UTC().Unix() > int64(exp) {
			return nil
		}
	}

	return claims
}

var ErrInvalidToken = errors.New("invalid token")
