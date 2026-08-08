package routes

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/lib/pq"

	"kabackend/database"
	"kabackend/models"
	"kabackend/security"
)

// pqInt64Array wraps a []int64 as a Postgres array literal for use with
// `= ANY($1)` queries, via lib/pq's pq.Array helper.
func pqInt64Array(ids []int64) driver.Valuer {
	return pq.Array(ids)
}

// parsePathID reads a {name} path wildcard (Go 1.22 http.ServeMux) and
// parses it as an int64, mirroring FastAPI's `{id}: int` path parameters.
func parsePathID(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
}

// writeJSON writes v as a JSON body with the given status code. The
// EncryptionMiddleware is responsible for wrapping/encrypting this before
// it reaches the client, mirroring how FastAPI handlers just return plain
// dicts/pydantic models in the original app.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		w.Write([]byte("null"))
		return
	}
	json.NewEncoder(w).Encode(v)
}

// writeError mirrors FastAPI's default HTTPException JSON shape:
// {"detail": "<message>"}.
func writeError(w http.ResponseWriter, status int, detail string) {
	writeJSON(w, status, map[string]string{"detail": detail})
}

func decodeJSONBody(r *http.Request, dst interface{}) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	return dec.Decode(dst)
}

// getBearerToken mirrors FastAPI's OAuth2PasswordBearer(tokenUrl="login"):
// pulls the token out of an `Authorization: Bearer <token>` header.
func getBearerToken(r *http.Request) (string, bool) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", false
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	return parts[1], true
}

// getCurrentUser mirrors security/oauth2.py's get_current_user dependency.
// On failure it writes the 401 response itself and returns ok=false, so
// callers can just `if !ok { return }`.
func getCurrentUser(w http.ResponseWriter, r *http.Request) (*models.User, bool) {
	token, ok := getBearerToken(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", "Bearer")
		writeError(w, http.StatusUnauthorized, "Could Not Validate Credentials")
		return nil, false
	}

	claims := security.DecodeAccessToken(token)
	if claims == nil {
		w.Header().Set("WWW-Authenticate", "Bearer")
		writeError(w, http.StatusUnauthorized, "Could Not Validate Credentials")
		return nil, false
	}

	emailRaw, ok := claims["sub"]
	email, isStr := emailRaw.(string)
	if !ok || !isStr || email == "" {
		w.Header().Set("WWW-Authenticate", "Bearer")
		writeError(w, http.StatusUnauthorized, "Could Not Validate Credentials")
		return nil, false
	}

	user, err := getUserByEmail(email)
	if err != nil || user == nil {
		w.Header().Set("WWW-Authenticate", "Bearer")
		writeError(w, http.StatusUnauthorized, "Could Not Validate Credentials")
		return nil, false
	}

	return user, true
}

// requirePresident mirrors security/oauth2.py's require_president dependency.
func requirePresident(w http.ResponseWriter, r *http.Request) (*models.User, bool) {
	user, ok := getCurrentUser(w, r)
	if !ok {
		return nil, false
	}
	if user.Role != "president" {
		writeError(w, http.StatusForbidden, "This action is restricted to the President / State Coordinator.")
		return nil, false
	}
	return user, true
}

func getUserByEmail(email string) (*models.User, error) {
	row := database.DB.QueryRow(
		`SELECT id, name, email, phone, password, district, role, created_time FROM users WHERE email = $1`,
		email,
	)
	var u models.User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.Phone, &u.Password, &u.District, &u.Role, &u.CreatedTime)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func getUserByID(id int64) (*models.User, error) {
	row := database.DB.QueryRow(
		`SELECT id, name, email, phone, password, district, role, created_time FROM users WHERE id = $1`,
		id,
	)
	var u models.User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.Phone, &u.Password, &u.District, &u.Role, &u.CreatedTime)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
