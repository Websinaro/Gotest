package routes

import (
	"net/http"
	"strings"

	"kabackend/config"
	"kabackend/database"
	"kabackend/models"
	"kabackend/security"
	"kabackend/utils"
)

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var body UserCreate
	if err := decodeJSONBody(r, &body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Invalid request body")
		return
	}

	if !isValidEmail(body.Email) {
		writeError(w, http.StatusUnprocessableEntity, "value is not a valid email address")
		return
	}

	existing, err := getUserByEmail(body.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if existing != nil {
		writeError(w, http.StatusBadRequest, "Email Already Registered")
		return
	}

	validationResult := utils.PasswordValidator(body.Password)
	if validationResult != "Strong Password" {
		writeError(w, http.StatusBadRequest, validationResult)
		return
	}

	// A signup only becomes "president" (admin) if it supplies the correct
	// official access code. A wrong/missing code when the toggle was on is
	// rejected outright rather than silently falling back to a normal
	// user, so nobody quietly ends up thinking they registered as an admin.
	role := "user"
	if body.AccessCode != nil && strings.TrimSpace(*body.AccessCode) != "" {
		if strings.TrimSpace(*body.AccessCode) != config.PresidentAccessCode {
			writeError(w, http.StatusBadRequest, "Invalid president access code")
			return
		}
		role = "president"
	}

	hashed, err := security.HashPass(body.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	var newID int64
	err = database.DB.QueryRow(
		`INSERT INTO users (name, email, phone, password, district, role, created_time)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		body.Name, body.Email, body.Phone, hashed, body.District, role, utils.PyUTCNowStr(),
	).Scan(&newID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}

	writeJSON(w, http.StatusOK, UserOut{
		ID:       newID,
		Name:     body.Name,
		Email:    body.Email,
		District: body.District,
		Role:     role,
	})
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Invalid form data")
		return
	}
	email := r.FormValue("email")
	password := r.FormValue("password")

	user, err := getUserByEmail(email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if user == nil || !security.VerifyPass(password, user.Password) {
		w.Header().Set("WWW-Authenticate", "Bearer")
		writeError(w, http.StatusUnauthorized, "Incorrect email or password or User not found")
		return
	}

	accessToken, err := security.CreateAccessToken(map[string]interface{}{"sub": user.Email}, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create access token")
		return
	}

	writeJSON(w, http.StatusOK, TokenOut{AccessToken: accessToken, TokenType: "bearer"})
}

func MeHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getCurrentUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, userToOut(user))
}

func userToOut(u *models.User) UserOut {
	return UserOut{ID: u.ID, Name: u.Name, Email: u.Email, District: u.District, Role: u.Role}
}
