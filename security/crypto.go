package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"

	"kabackend/config"
)

var aesKey []byte

// InitCrypto decodes AES_SECRET_KEY once at startup, mirroring the
// module-level `AES_KEY = base64.b64decode(AES_SECRET_KEY)` in
// security/crypto.py.
func InitCrypto() error {
	key, err := base64.StdEncoding.DecodeString(config.AESSecretKey)
	if err != nil {
		return err
	}
	aesKey = key
	return nil
}

func newAESGCM() (cipher.AEAD, error) {
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// EncryptPayload mirrors encrypt_payload(data): JSON-encode, AES-256-GCM
// encrypt with a random 12-byte nonce, return base64(nonce+ciphertext).
func EncryptPayload(data interface{}) (string, error) {
	aesgcm, err := newAESGCM()
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	plaintext, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	ciphertext := aesgcm.Seal(nil, nonce, plaintext, nil)
	combined := append(nonce, ciphertext...)
	return base64.StdEncoding.EncodeToString(combined), nil
}

// DecryptPayload mirrors decrypt_payload(token): base64-decode, split
// nonce(12)/ciphertext, AES-256-GCM decrypt, JSON-decode into a generic value.
func DecryptPayload(token string) (interface{}, error) {
	raw, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, err
	}

	aesgcm, err := newAESGCM()
	if err != nil {
		return nil, err
	}

	nonceSize := aesgcm.NonceSize()
	if len(raw) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ciphertext := raw[:nonceSize], raw[nonceSize:]

	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	var data interface{}
	if err := json.Unmarshal(plaintext, &data); err != nil {
		return nil, err
	}
	return data, nil
}
