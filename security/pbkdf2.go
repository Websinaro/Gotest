package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
)

// pbkdf2Key derives a key of the requested length from password+salt using
// PBKDF2-HMAC-SHA256 (RFC 8018). Implemented directly against the standard
// library (crypto/hmac + crypto/sha256) so password hashing has zero
// third-party dependencies.
func pbkdf2Key(password, salt []byte, iterations, keyLen int) []byte {
	prf := hmac.New(sha256.New, password)
	hashLen := prf.Size()
	numBlocks := (keyLen + hashLen - 1) / hashLen

	derived := make([]byte, 0, numBlocks*hashLen)
	buf := make([]byte, len(salt)+4)
	copy(buf, salt)

	for block := 1; block <= numBlocks; block++ {
		binary.BigEndian.PutUint32(buf[len(salt):], uint32(block))

		prf.Reset()
		prf.Write(buf)
		u := prf.Sum(nil)

		t := make([]byte, hashLen)
		copy(t, u)

		for i := 1; i < iterations; i++ {
			prf.Reset()
			prf.Write(u)
			u = prf.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}

		derived = append(derived, t...)
	}

	return derived[:keyLen]
}
