package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// Password hashes are stored as: pbkdf2_sha256$<iterations>$<salt-b64>$<hash-b64>
// NOTE: the original Python backend used passlib's argon2. This Go port has
// no third-party dependency for that, so it uses PBKDF2-HMAC-SHA256
// (implemented against the standard library) instead. Behavior (hash on
// register, verify on login) is identical; only the underlying KDF differs.
const (
	pbkdf2Iterations = 210000
	pbkdf2KeyLen     = 32
	pbkdf2SaltLen    = 16
)

// HashPass mirrors hashPass(password) in security/hashing.py.
func HashPass(password string) (string, error) {
	salt := make([]byte, pbkdf2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	derived := pbkdf2Key([]byte(password), salt, pbkdf2Iterations, pbkdf2KeyLen)

	return fmt.Sprintf(
		"pbkdf2_sha256$%d$%s$%s",
		pbkdf2Iterations,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(derived),
	), nil
}

// VerifyPass mirrors verifyPass(plain_pass, hash_pass) in security/hashing.py.
func VerifyPass(plainPass, hashPass string) bool {
	parts := strings.Split(hashPass, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2_sha256" {
		return false
	}

	iterations, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}

	actual := pbkdf2Key([]byte(plainPass), salt, iterations, len(expected))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}
