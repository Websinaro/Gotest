package utils

import "unicode"

// PasswordValidator mirrors passwordValidator(password) in
// utils/password.py: returns "Strong Password" if it passes, otherwise a
// human-readable reason it failed.
func PasswordValidator(password string) string {
	if len([]rune(password)) < 8 {
		return "Your password needs at least 8 characters."
	}

	hasDigit, hasUpper, hasLower, hasSymbol := false, false, false, false
	for _, ch := range password {
		switch {
		case unicode.IsDigit(ch):
			hasDigit = true
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case !unicode.IsLetter(ch) && !unicode.IsDigit(ch):
			hasSymbol = true
		}
	}

	if !hasDigit {
		return "Your password should contain at least one number."
	}
	if !hasUpper {
		return "Your password should contain at least one uppercase letter."
	}
	if !hasLower {
		return "Your password should contain at least one lowercase letter."
	}
	if !hasSymbol {
		return "Your password should contain at least one symbol (e.g. !@#$%)."
	}

	return "Strong Password"
}
