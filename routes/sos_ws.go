package routes

import (
	"encoding/json"
	"log"
	"net/http"

	"kabackend/database"
	"kabackend/security"
	"kabackend/services"
	"kabackend/utils"
)

// wsLocationIn is what the sender's app publishes over the socket on every
// GPS fix. Kept intentionally small (short field names, no envelope) since
// every byte matters on a weak 2G/3G link.
type wsLocationIn struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Accuracy  float64 `json:"accuracy_m"`
	SpeedMps  float64 `json:"speed_mps"`
	Heading   float64 `json:"heading_deg"`
}

// authenticateWSRequest mirrors getCurrentUser, but reads the token from a
// query parameter instead of an Authorization header. A browser/mobile
// WebSocket handshake can't attach custom headers as reliably as a normal
// HTTPS request on every platform, and a query param is what every WS
// client library supports uniformly - the connection itself still runs
// over wss:// (TLS), so the token is exactly as protected in transit as
// it would be in a header.
func authenticateWSRequest(r *http.Request) (userID int64, ok bool) {
	token := r.URL.Query().Get("token")
	if token == "" {
		if headerToken, has := getBearerToken(r); has {
			token = headerToken
		}
	}
	if token == "" {
		return 0, false
	}
	claims := security.DecodeAccessToken(token)
	if claims == nil {
		return 0, false
	}
	email, isStr := claims["sub"].(string)
	if !isStr || email == "" {
		return 0, false
	}
	user, err := getUserByEmail(email)
	if err != nil || user == nil {
		return 0, false
	}
	return user.ID, true
}

// SosLocationWebSocketHandler upgrades GET /ws/sos/{id}?token=... to a
// WebSocket. The alert's own sender may connect to stream GPS fixes; any
// registered protector (a safety contact with a matching account) may
// connect read-only to watch them arrive live, exactly mirroring the
// authorization rule GetSosHandler already enforces over REST.
func SosLocationWebSocketHandler(w http.ResponseWriter, r *http.Request) {
	sosID, err := parsePathID(r, "id")
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Invalid sos id")
		return
	}

	userID, ok := authenticateWSRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Could Not Validate Credentials")
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

	isOwner := alert.UserID == userID
	if !isOwner {
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
		if !protectorIDs(protectors)[userID] {
			writeError(w, http.StatusForbidden, "You are not authorized to watch this alert")
			return
		}
	}

	conn, err := services.UpgradeWebSocket(w, r)
	if err != nil {
		// UpgradeWebSocket already hijacked (or failed before doing so);
		// nothing more we can safely write to w at this point.
		log.Printf("[SOS WS] upgrade failed for sos=%d user=%d: %v", sosID, userID, err)
		return
	}
	defer conn.Close()

	services.GlobalSosHub.Join(sosID, conn, userID, isOwner)
	defer services.GlobalSosHub.Leave(sosID, conn)

	for {
		opcode, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if opcode != services.OpText { // text frames only
			continue
		}
		if !isOwner {
			// Protectors are read-only observers on this socket - anything
			// they send is ignored rather than trusted as a location.
			continue
		}

		var in wsLocationIn
		if err := json.Unmarshal(payload, &in); err != nil {
			continue
		}
		if in.Latitude < -90 || in.Latitude > 90 || in.Longitude < -180 || in.Longitude > 180 {
			continue
		}

		// Only push updates while the alert is still active, mirroring
		// UpdateSosLocationHandler's REST behaviour, so a stray late frame
		// after "mark safe" can't resurrect a resolved alert's pin.
		var status string
		if err := database.DB.QueryRow(`SELECT status FROM sos_alerts WHERE id = $1`, sosID).Scan(&status); err != nil {
			return
		}
		if status != "active" {
			return
		}

		if _, err := database.DB.Exec(
			`UPDATE sos_alerts SET latitude=$1, longitude=$2 WHERE id=$3`, in.Latitude, in.Longitude, sosID,
		); err != nil {
			log.Printf("[SOS WS] failed to persist location for sos=%d: %v", sosID, err)
			continue
		}

		ts := utils.PyUTCNowStr()
		services.GlobalSosHub.BroadcastLocation(sosID, in.Latitude, in.Longitude, in.Accuracy, in.SpeedMps, in.Heading, ts, conn)
	}
}
