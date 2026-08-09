package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Params mirror passlib's argon2 handler defaults (backed by argon2-cffi),
// which is what security/hashing.py's `CryptContext(schemes=["argon2"])`
// used: argon2id, 64 MiB memory, 3 iterations, 4 lanes, 16-byte salt,
// 32-byte hash.
const (
	argonMemory  uint32 = 65536 // KiB
	argonTime    uint32 = 3
	argonThreads uint8  = 4
	argonSaltLen        = 16
	argonKeyLen  uint32 = 32
)

// Hashes are stored in the same PHC string format passlib emits:
// $argon2id$v=19$m=65536,t=3,p=4$<salt-b64>$<hash-b64>

// HashPass mirrors hashPass(password) in security/hashing.py.
func HashPass(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

// VerifyPass mirrors verifyPass(plain_pass, hash_pass) in security/hashing.py.
func VerifyPass(plainPass, hashPass string) bool {
	// ["", "argon2id", "v=19", "m=65536,t=3,p=4", "<salt>", "<hash>"]
	parts := strings.Split(hashPass, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false
	}

	var memory, timeCost uint32
	var threads uint8
	for _, kv := range strings.Split(parts[3], ",") {
		pair := strings.SplitN(kv, "=", 2)
		if len(pair) != 2 {
			return false
		}
		val, err := strconv.Atoi(pair[1])
		if err != nil {
			return false
		}
		switch pair[0] {
		case "m":
			memory = uint32(val)
		case "t":
			timeCost = uint32(val)
		case "p":
			threads = uint8(val)
		default:
			return false
		}
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	actual := argon2.IDKey([]byte(plainPass), salt, timeCost, memory, threads, uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}
