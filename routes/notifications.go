package routes

import (
	"database/sql"
	"net/http"
	"strings"

	"kabackend/data"
	"kabackend/database"
	"kabackend/models"
	"kabackend/services"
	"kabackend/utils"
)

const notificationSelectCols = `id, title, message, severity, district, created_by, created_by_name, active, created_time, updated_time`

func notificationToOut(n *models.Notification) NotificationOut {
	return NotificationOut{
		ID: n.ID, Title: n.Title, Message: n.Message, Severity: n.Severity, District: n.District,
		CreatedBy: n.CreatedBy, CreatedByName: n.CreatedByName, Active: n.Active,
		CreatedTime: n.CreatedTime, UpdatedTime: n.UpdatedTime,
	}
}

func scanNotificationRow(row *sql.Row) (*models.Notification, error) {
	var n models.Notification
	err := row.Scan(&n.ID, &n.Title, &n.Message, &n.Severity, &n.District, &n.CreatedBy, &n.CreatedByName, &n.Active, &n.CreatedTime, &n.UpdatedTime)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func validateDistrict(district *string) bool {
	if district == nil {
		return true
	}
	_, ok := data.KeralaDistricts[*district]
	return ok
}

// broadcastNotification mirrors _broadcast(db, notification) in
// routes/notifications.py: fans the alert out as a push to every device
// belonging to a user in the target district, or every registered device
// for a state-wide alert. Best-effort - failures are swallowed just like
// the Python push helpers.
func broadcastNotification(n *models.Notification) {
	var rows *sql.Rows
	var err error
	if n.District != nil {
		rows, err = database.DB.Query(`SELECT id FROM users WHERE district = $1`, *n.District)
	} else {
		rows, err = database.DB.Query(`SELECT id FROM users`)
	}
	if err != nil {
		return
	}
	var userIDs []int64
	for rows.Next() {
		var id int64
		if rows.Scan(&id) == nil {
			userIDs = append(userIDs, id)
		}
	}
	rows.Close()
	if len(userIDs) == 0 {
		return
	}

	tokenRows, err := database.DB.Query(`SELECT fcm_token FROM device_tokens WHERE user_id = ANY($1::bigint[])`, pqInt64Array(userIDs))
	if err != nil {
		return
	}
	defer tokenRows.Close()
	for tokenRows.Next() {
		var token string
		if tokenRows.Scan(&token) == nil {
			services.SendAdminAlertPush(token, n.ID, n.Title, n.Message, n.Severity, n.District)
		}
	}
}

func CreateNotificationHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePresident(w, r)
	if !ok {
		return
	}

	var body NotificationCreate
	if err := decodeJSONBody(r, &body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Invalid request body")
		return
	}
	if body.Severity == "" {
		body.Severity = "orange"
	}
	if !allowedSeverities[body.Severity] {
		writeError(w, http.StatusUnprocessableEntity, "severity must be one of [dark_red, green, light_red, orange, yellow]")
		return
	}
	body.Title = strings.TrimSpace(body.Title)
	if body.Title == "" {
		writeError(w, http.StatusUnprocessableEntity, "title is required")
		return
	}
	body.Message = strings.TrimSpace(body.Message)
	if body.Message == "" {
		writeError(w, http.StatusUnprocessableEntity, "message is required")
		return
	}
	if !validateDistrict(body.District) {
		writeError(w, http.StatusBadRequest, "Unknown district")
		return
	}

	now := utils.PyUTCNowStr()
	var n models.Notification
	err := database.DB.QueryRow(
		`INSERT INTO notifications (title, message, severity, district, created_by, created_by_name, active, created_time, updated_time)
		 VALUES ($1, $2, $3, $4, $5, $6, true, $7, $7) RETURNING `+notificationSelectCols,
		body.Title, body.Message, body.Severity, body.District, user.ID, user.Name, now,
	).Scan(&n.ID, &n.Title, &n.Message, &n.Severity, &n.District, &n.CreatedBy, &n.CreatedByName, &n.Active, &n.CreatedTime, &n.UpdatedTime)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}

	broadcastNotification(&n)

	writeJSON(w, http.StatusOK, notificationToOut(&n))
}

func ListNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getCurrentUser(w, r)
	if !ok {
		return
	}

	var rows *sql.Rows
	var err error
	if user.Role == "president" {
		rows, err = database.DB.Query(
			`SELECT `+notificationSelectCols+` FROM notifications WHERE created_by = $1 ORDER BY id DESC`, user.ID,
		)
	} else {
		rows, err = database.DB.Query(
			`SELECT `+notificationSelectCols+` FROM notifications
			 WHERE active = true AND (district = $1 OR district IS NULL) ORDER BY id DESC`, user.District,
		)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	out := []NotificationOut{}
	for rows.Next() {
		var n models.Notification
		if err := rows.Scan(&n.ID, &n.Title, &n.Message, &n.Severity, &n.District, &n.CreatedBy, &n.CreatedByName, &n.Active, &n.CreatedTime, &n.UpdatedTime); err != nil {
			writeError(w, http.StatusInternalServerError, "Database error")
			return
		}
		out = append(out, notificationToOut(&n))
	}

	writeJSON(w, http.StatusOK, out)
}

func GetNotificationHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getCurrentUser(w, r)
	if !ok {
		return
	}

	notificationID, err := parsePathID(r, "id")
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Invalid notification id")
		return
	}

	n, err := scanNotificationRow(database.DB.QueryRow(`SELECT `+notificationSelectCols+` FROM notifications WHERE id = $1`, notificationID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if n == nil {
		writeError(w, http.StatusNotFound, "Notification not found")
		return
	}

	if user.Role != "president" {
		if !n.Active {
			writeError(w, http.StatusNotFound, "Notification not found")
			return
		}
		if n.District != nil && *n.District != user.District {
			writeError(w, http.StatusForbidden, "This alert does not target your district")
			return
		}
	}

	writeJSON(w, http.StatusOK, notificationToOut(n))
}

func UpdateNotificationHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePresident(w, r)
	if !ok {
		return
	}

	notificationID, err := parsePathID(r, "id")
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Invalid notification id")
		return
	}

	var body NotificationUpdate
	if err := decodeJSONBody(r, &body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Invalid request body")
		return
	}
	if body.Severity != nil && !allowedSeverities[*body.Severity] {
		writeError(w, http.StatusUnprocessableEntity, "severity must be one of [dark_red, green, light_red, orange, yellow]")
		return
	}

	n, err := scanNotificationRow(database.DB.QueryRow(`SELECT `+notificationSelectCols+` FROM notifications WHERE id = $1`, notificationID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if n == nil {
		writeError(w, http.StatusNotFound, "Notification not found")
		return
	}
	if n.CreatedBy != user.ID {
		writeError(w, http.StatusForbidden, "You can only edit alerts you sent")
		return
	}

	if body.Title != nil {
		trimmed := strings.TrimSpace(*body.Title)
		if trimmed == "" {
			writeError(w, http.StatusBadRequest, "title cannot be empty")
			return
		}
		n.Title = trimmed
	}
	if body.Message != nil {
		trimmed := strings.TrimSpace(*body.Message)
		if trimmed == "" {
			writeError(w, http.StatusBadRequest, "message cannot be empty")
			return
		}
		n.Message = trimmed
	}
	if body.Severity != nil {
		n.Severity = *body.Severity
	}
	if body.ClearDistrict {
		n.District = nil
	} else if body.District != nil {
		if !validateDistrict(body.District) {
			writeError(w, http.StatusBadRequest, "Unknown district")
			return
		}
		n.District = body.District
	}
	if body.Active != nil {
		n.Active = *body.Active
	}
	n.UpdatedTime = utils.PyUTCNowStr()

	_, err = database.DB.Exec(
		`UPDATE notifications SET title=$1, message=$2, severity=$3, district=$4, active=$5, updated_time=$6 WHERE id=$7`,
		n.Title, n.Message, n.Severity, n.District, n.Active, n.UpdatedTime, n.ID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Re-notify affected districts only when the alert is (re-)activated,
	// e.g. a president correcting a typo and re-sending, not on every edit.
	if body.Active != nil && *body.Active {
		broadcastNotification(n)
	}

	writeJSON(w, http.StatusOK, notificationToOut(n))
}

func DeleteNotificationHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePresident(w, r)
	if !ok {
		return
	}

	notificationID, err := parsePathID(r, "id")
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Invalid notification id")
		return
	}

	n, err := scanNotificationRow(database.DB.QueryRow(`SELECT `+notificationSelectCols+` FROM notifications WHERE id = $1`, notificationID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if n == nil {
		writeError(w, http.StatusNotFound, "Notification not found")
		return
	}
	if n.CreatedBy != user.ID {
		writeError(w, http.StatusForbidden, "You can only delete alerts you sent")
		return
	}

	_, err = database.DB.Exec(`DELETE FROM notifications WHERE id = $1`, notificationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Notification deleted"})
}
