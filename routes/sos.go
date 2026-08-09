package routes

import (
	"database/sql"
	"net/http"

	"kabackend/database"
	"kabackend/models"
	"kabackend/services"
	"kabackend/utils"
)

func sosToOut(a *models.SosAlert) SosOut {
	return SosOut{
		ID: a.ID, UserID: a.UserID, Latitude: a.Latitude, Longitude: a.Longitude,
		Status: a.Status, Message: a.Message, CreatedTime: a.CreatedTime, ResolvedTime: a.ResolvedTime,
	}
}

func scanSosRow(row *sql.Row) (*models.SosAlert, error) {
	var a models.SosAlert
	err := row.Scan(&a.ID, &a.UserID, &a.Latitude, &a.Longitude, &a.Status, &a.Message, &a.CreatedTime, &a.ResolvedTime)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

const sosSelectCols = `id, user_id, latitude, longitude, status, message, created_time, resolved_time`

// findProtectorUsers mirrors _find_protector_users(db, contacts) in
// routes/sos.py: matches each safety contact to a registered WeBAlert
// account by phone or email, if one exists.
func findProtectorUsers(contacts []models.SafetyContact) ([]models.User, error) {
	var protectors []models.User
	for _, contact := range contacts {
		row := database.DB.QueryRow(
			`SELECT id, name, email, phone, password, district, role, created_time
			 FROM users WHERE phone = $1 OR email = $2`,
			contact.Phone, contact.Email,
		)
		var u models.User
		err := row.Scan(&u.ID, &u.Name, &u.Email, &u.Phone, &u.Password, &u.District, &u.Role, &u.CreatedTime)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return nil, err
		}
		protectors = append(protectors, u)
	}
	return protectors, nil
}

func protectorIDs(protectors []models.User) map[int64]bool {
	ids := make(map[int64]bool, len(protectors))
	for _, p := range protectors {
		ids[p.ID] = true
	}
	return ids
}

func CreateSosHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getCurrentUser(w, r)
	if !ok {
		return
	}

	var body SosCreate
	if err := decodeJSONBody(r, &body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Invalid request body")
		return
	}

	var existingID int64
	err := database.DB.QueryRow(
		`SELECT id FROM sos_alerts WHERE user_id = $1 AND status = 'active'`, user.ID,
	).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if err == nil {
		writeError(w, http.StatusBadRequest, "You already have an active SOS alert.")
		return
	}

	createdTime := utils.PyUTCNowStr()
	var alert models.SosAlert
	err = database.DB.QueryRow(
		`INSERT INTO sos_alerts (user_id, latitude, longitude, message, status, created_time)
		 VALUES ($1, $2, $3, $4, 'active', $5) RETURNING `+sosSelectCols,
		user.ID, body.Latitude, body.Longitude, body.Message, createdTime,
	).Scan(&alert.ID, &alert.UserID, &alert.Latitude, &alert.Longitude, &alert.Status, &alert.Message, &alert.CreatedTime, &alert.ResolvedTime)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}

	contacts, err := getSafetyContactsForUser(user.ID)
	if err == nil {
		protectors, err := findProtectorUsers(contacts)
		if err == nil {
			for _, protector := range protectors {
				rows, err := database.DB.Query(`SELECT fcm_token FROM device_tokens WHERE user_id = $1`, protector.ID)
				if err != nil {
					continue
				}
				var tokens []string
				for rows.Next() {
					var t string
					if rows.Scan(&t) == nil {
						tokens = append(tokens, t)
					}
				}
				rows.Close()
				for _, token := range tokens {
					services.SendSosPush(token, alert.ID, user.Name, alert.Latitude, alert.Longitude)
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, sosToOut(&alert))
}

func GetSosHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getCurrentUser(w, r)
	if !ok {
		return
	}

	sosID, err := parsePathID(r, "id")
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Invalid sos id")
		return
	}

	alert, err := scanSosRow(database.DB.QueryRow(`SELECT `+sosSelectCols+` FROM sos_alerts WHERE id = $1`, sosID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if alert == nil {
		writeError(w, http.StatusNotFound, "SOS alert not found")
		return
	}

	if alert.UserID != user.ID {
		contacts, err := getSafetyContactsForUser(alert.UserID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Database error")
			return
		}
		protectors, err := findProtectorUsers(contacts)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Database error")
			return
		}
		if !protectorIDs(protectors)[user.ID] {
			writeError(w, http.StatusForbidden, "You are not authorized to view this alert")
			return
		}
	}

	writeJSON(w, http.StatusOK, sosToOut(alert))
}

func UpdateSosLocationHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getCurrentUser(w, r)
	if !ok {
		return
	}

	sosID, err := parsePathID(r, "id")
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Invalid sos id")
		return
	}

	var body SosLocationUpdate
	if err := decodeJSONBody(r, &body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Invalid request body")
		return
	}

	alert, err := scanSosRow(database.DB.QueryRow(
		`SELECT `+sosSelectCols+` FROM sos_alerts WHERE id = $1 AND user_id = $2`, sosID, user.ID,
	))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if alert == nil {
		writeError(w, http.StatusNotFound, "SOS alert not found")
		return
	}
	if alert.Status != "active" {
		writeError(w, http.StatusBadRequest, "This SOS alert is no longer active")
		return
	}

	_, err = database.DB.Exec(`UPDATE sos_alerts SET latitude=$1, longitude=$2 WHERE id=$3`, body.Latitude, body.Longitude, sosID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	alert.Latitude = body.Latitude
	alert.Longitude = body.Longitude

	// This REST endpoint stays as the reliable fallback path for when the
	// WebSocket is down (poor 2G/3G connectivity), but any protector who
	// does have a live socket open should still see the update instantly
	// rather than waiting for their own next poll - so mirror it into the
	// same hub the WS handler broadcasts from.
	services.GlobalSosHub.BroadcastLocation(sosID, body.Latitude, body.Longitude, 0, 0, 0, utils.PyUTCNowStr(), nil)

	writeJSON(w, http.StatusOK, sosToOut(alert))
}

func ResolveSosHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getCurrentUser(w, r)
	if !ok {
		return
	}

	sosID, err := parsePathID(r, "id")
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Invalid sos id")
		return
	}

	alert, err := scanSosRow(database.DB.QueryRow(
		`SELECT `+sosSelectCols+` FROM sos_alerts WHERE id = $1 AND user_id = $2`, sosID, user.ID,
	))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if alert == nil {
		writeError(w, http.StatusNotFound, "SOS alert not found")
		return
	}

	resolvedTime := utils.PyUTCNowStr()
	_, err = database.DB.Exec(`UPDATE sos_alerts SET status='resolved', resolved_time=$1 WHERE id=$2`, resolvedTime, sosID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	alert.Status = "resolved"
	alert.ResolvedTime = &resolvedTime

	// Let any open WebSocket (sender's own app, protectors' live map
	// screens) know immediately, and close out that room.
	services.GlobalSosHub.BroadcastResolved(sosID, resolvedTime)

	// Also push a "sos_resolved" FCM message to every protector so their
	// app cancels the stale ongoing notification even if they don't have
	// the live map open right now (background/killed app) - see the doc
	// comment on SendSosResolvedPush for why this matters for notification
	// reliability on the next SOS.
	contacts, err := getSafetyContactsForUser(user.ID)
	if err == nil {
		protectors, err := findProtectorUsers(contacts)
		if err == nil {
			for _, protector := range protectors {
				rows, err := database.DB.Query(`SELECT fcm_token FROM device_tokens WHERE user_id = $1`, protector.ID)
				if err != nil {
					continue
				}
				var tokens []string
				for rows.Next() {
					var t string
					if rows.Scan(&t) == nil {
						tokens = append(tokens, t)
					}
				}
				rows.Close()
				for _, token := range tokens {
					services.SendSosResolvedPush(token, sosID)
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, sosToOut(alert))
}

// ListIncomingActiveSosHandler mirrors GET /sos/active/incoming: all
// currently-active SOS alerts where the logged-in user is a registered
// protector for the sender.
func ListIncomingActiveSosHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getCurrentUser(w, r)
	if !ok {
		return
	}

	rows, err := database.DB.Query(`SELECT ` + sosSelectCols + ` FROM sos_alerts WHERE status = 'active'`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	var allActive []models.SosAlert
	for rows.Next() {
		var a models.SosAlert
		if err := rows.Scan(&a.ID, &a.UserID, &a.Latitude, &a.Longitude, &a.Status, &a.Message, &a.CreatedTime, &a.ResolvedTime); err != nil {
			rows.Close()
			writeError(w, http.StatusInternalServerError, "Database error")
			return
		}
		allActive = append(allActive, a)
	}
	rows.Close()

	result := []SosOut{}
	for _, alert := range allActive {
		contacts, err := getSafetyContactsForUser(alert.UserID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Database error")
			return
		}
		protectors, err := findProtectorUsers(contacts)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Database error")
			return
		}
		if protectorIDs(protectors)[user.ID] {
			result = append(result, sosToOut(&alert))
		}
	}

	writeJSON(w, http.StatusOK, result)
}

// GetMyActiveSosHandler mirrors GET /sos/mine/active. Lets the app check on
// launch whether the logged-in user already has an active SOS running.
func GetMyActiveSosHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getCurrentUser(w, r)
	if !ok {
		return
	}

	alert, err := scanSosRow(database.DB.QueryRow(
		`SELECT `+sosSelectCols+` FROM sos_alerts WHERE user_id = $1 AND status = 'active'`, user.ID,
	))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if alert == nil {
		writeJSON(w, http.StatusOK, nil)
		return
	}

	writeJSON(w, http.StatusOK, sosToOut(alert))
}
