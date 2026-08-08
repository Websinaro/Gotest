package services

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"kabackend/config"
)

// serviceAccount mirrors the fields firebase_admin's credentials.Certificate()
// reads out of the Firebase service-account JSON.
type serviceAccount struct {
	ProjectID   string `json:"project_id"`
	PrivateKey  string `json:"private_key"`
	ClientEmail string `json:"client_email"`
	TokenURI    string `json:"token_uri"`
}

var (
	fbOnce    sync.Once
	fbAccount *serviceAccount
	fbKey     *rsa.PrivateKey
	fbInitErr error

	tokenMu      sync.Mutex
	cachedToken  string
	cachedExpiry time.Time
)

func loadServiceAccount() (*serviceAccount, *rsa.PrivateKey, error) {
	fbOnce.Do(func() {
		decoded, err := base64.StdEncoding.DecodeString(config.FirebaseCredentialsB64)
		if err != nil {
			fbInitErr = fmt.Errorf("invalid FIREBASE_CREDENTIALS_B64: %w", err)
			return
		}

		var sa serviceAccount
		if err := json.Unmarshal(decoded, &sa); err != nil {
			fbInitErr = fmt.Errorf("invalid firebase credentials JSON: %w", err)
			return
		}

		block, _ := pem.Decode([]byte(sa.PrivateKey))
		if block == nil {
			fbInitErr = errors.New("invalid firebase private key PEM")
			return
		}

		var key *rsa.PrivateKey
		if parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
			rsaKey, ok := parsed.(*rsa.PrivateKey)
			if !ok {
				fbInitErr = errors.New("firebase private key is not RSA")
				return
			}
			key = rsaKey
		} else if parsed, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
			key = parsed
		} else {
			fbInitErr = fmt.Errorf("failed to parse firebase private key: %w", err)
			return
		}

		if sa.TokenURI == "" {
			sa.TokenURI = "https://oauth2.googleapis.com/token"
		}

		fbAccount = &sa
		fbKey = key
	})
	return fbAccount, fbKey, fbInitErr
}

func base64URL(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

// getAccessToken obtains (and caches) an OAuth2 access token for the
// firebase.messaging scope via the service-account JWT-bearer grant
// (RFC 7523) - a from-scratch equivalent of what firebase_admin does
// internally so this Go port needs no Google API client library.
func getAccessToken() (string, error) {
	sa, key, err := loadServiceAccount()
	if err != nil {
		return "", err
	}

	tokenMu.Lock()
	defer tokenMu.Unlock()

	if cachedToken != "" && time.Now().Before(cachedExpiry) {
		return cachedToken, nil
	}

	now := time.Now()
	header := map[string]string{"alg": "RS256", "typ": "JWT"}
	claims := map[string]interface{}{
		"iss":   sa.ClientEmail,
		"scope": "https://www.googleapis.com/auth/firebase.messaging",
		"aud":   sa.TokenURI,
		"iat":   now.Unix(),
		"exp":   now.Add(1 * time.Hour).Unix(),
	}

	headerBytes, _ := json.Marshal(header)
	claimsBytes, _ := json.Marshal(claims)
	signingInput := base64URL(headerBytes) + "." + base64URL(claimsBytes)

	hashed := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hashed[:])
	if err != nil {
		return "", err
	}

	assertion := signingInput + "." + base64URL(signature)

	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Set("assertion", assertion)

	resp, err := http.Post(sa.TokenURI, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("firebase token request failed: %s", tokenResp.Error)
	}

	cachedToken = tokenResp.AccessToken
	cachedExpiry = now.Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)

	return cachedToken, nil
}

func firebaseProjectID() (string, error) {
	sa, _, err := loadServiceAccount()
	if err != nil {
		return "", err
	}
	return sa.ProjectID, nil
}
