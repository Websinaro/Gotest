package routes

import (
	"net/http"

	"kabackend/data"
	"kabackend/database"
)

func PresidentDashboardHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := requirePresident(w, r)
	if !ok {
		return
	}

	var totalUsers int
	if err := database.DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&totalUsers); err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}

	var totalActiveSos int
	if err := database.DB.QueryRow(`SELECT COUNT(*) FROM sos_alerts WHERE status = 'active'`).Scan(&totalActiveSos); err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Only the current president's own active notifications count here -
	// matches routes/president.py filtering active_notifications by
	// created_by == current_user.id.
	var totalActiveNotifications int
	if err := database.DB.QueryRow(
		`SELECT COUNT(*) FROM notifications WHERE active = true AND created_by = $1`, user.ID,
	).Scan(&totalActiveNotifications); err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}

	districts := make([]DistrictStat, 0, len(data.KeralaDistricts))
	for name := range data.KeralaDistricts {
		var stat DistrictStat
		stat.District = name

		if err := database.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE district = $1`, name).Scan(&stat.RegisteredUsers); err != nil {
			writeError(w, http.StatusInternalServerError, "Database error")
			return
		}

		if err := database.DB.QueryRow(
			`SELECT COUNT(*) FROM sos_alerts sa JOIN users u ON sa.user_id = u.id
			 WHERE sa.status = 'active' AND u.district = $1`, name,
		).Scan(&stat.ActiveSos); err != nil {
			writeError(w, http.StatusInternalServerError, "Database error")
			return
		}

		// An all-Kerala alert (district IS NULL) counts against every
		// district, matching the Python dashboard's fan-out loop.
		if err := database.DB.QueryRow(
			`SELECT COUNT(*) FROM notifications
			 WHERE active = true AND created_by = $1 AND (district = $2 OR district IS NULL)`,
			user.ID, name,
		).Scan(&stat.ActiveNotifications); err != nil {
			writeError(w, http.StatusInternalServerError, "Database error")
			return
		}

		districts = append(districts, stat)
	}

	rows, err := database.DB.Query(
		`SELECT sa.id, sa.user_id, u.name, u.district, sa.latitude, sa.longitude, sa.message, sa.created_time
		 FROM sos_alerts sa JOIN users u ON sa.user_id = u.id
		 WHERE sa.status = 'active' ORDER BY sa.created_time DESC`,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	activeSosAlerts := []ActiveSosSummary{}
	for rows.Next() {
		var s ActiveSosSummary
		if err := rows.Scan(&s.ID, &s.UserID, &s.UserName, &s.District, &s.Latitude, &s.Longitude, &s.Message, &s.CreatedTime); err != nil {
			writeError(w, http.StatusInternalServerError, "Database error")
			return
		}
		activeSosAlerts = append(activeSosAlerts, s)
	}

	writeJSON(w, http.StatusOK, PresidentDashboard{
		TotalUsers:               totalUsers,
		TotalActiveSos:           totalActiveSos,
		TotalActiveNotifications: totalActiveNotifications,
		Districts:                districts,
		ActiveSosAlerts:          activeSosAlerts,
	})
}
