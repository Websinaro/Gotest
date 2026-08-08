package routes

import (
	"database/sql"
	"net/http"

	"kabackend/database"
	"kabackend/utils"
)

func RegisterDeviceTokenHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getCurrentUser(w, r)
	if !ok {
		return
	}

	var body DeviceTokenCreate
	if err := decodeJSONBody(r, &body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Invalid request body")
		return
	}
	now := utils.PyUTCNowStr()

	var existingID int64
	err := database.DB.QueryRow(`SELECT id FROM device_tokens WHERE fcm_token = $1`, body.FcmToken).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if err == sql.ErrNoRows {
		_, err = database.DB.Exec(
			`INSERT INTO device_tokens (user_id, fcm_token, platform, updated_time) VALUES ($1, $2, $3, $4)`,
			user.ID, body.FcmToken, body.Platform, now,
		)
	} else {
		_, err = database.DB.Exec(
			`UPDATE device_tokens SET user_id=$1, platform=$2, updated_time=$3 WHERE id=$4`,
			user.ID, body.Platform, now, existingID,
		)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Device token registered"})
}
