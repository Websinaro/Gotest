package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type fcmAndroidConfig struct {
	Priority string `json:"priority"`
	TTL      string `json:"ttl"`
}

type fcmMessage struct {
	Token   string            `json:"token"`
	Data    map[string]string `json:"data"`
	Android fcmAndroidConfig  `json:"android"`
}

// sendMessage mirrors the shared plumbing behind messaging.send(message) in
// services/push_service.py: a data-only FCM message (no "notification"
// block) so the client app has full control over how it's displayed, sent
// with android high priority and a 24h TTL so FCM queues it while the
// recipient's device is offline.
func sendMessage(fcmToken string, data map[string]string) error {
	projectID, err := firebaseProjectID()
	if err != nil {
		return err
	}
	accessToken, err := getAccessToken()
	if err != nil {
		return err
	}

	msg := fcmMessage{
		Token: fcmToken,
		Data:  data,
		Android: fcmAndroidConfig{
			Priority: "high",
			TTL:      "86400s", // 24 hours
		},
	}
	body, err := json.Marshal(map[string]interface{}{"message": msg})
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", projectID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("fcm send failed with status %d", resp.StatusCode)
	}
	return nil
}

// SendSosPush mirrors send_sos_push(fcm_token, sos_id, sender_name,
// latitude, longitude) in services/push_service.py. Errors are logged, not
// returned, matching the Python function's best-effort `except Exception`.
func SendSosPush(fcmToken string, sosID int64, senderName string, latitude, longitude float64) {
	err := sendMessage(fcmToken, map[string]string{
		"type":       "sos_alert",
		"sos_id":     fmt.Sprintf("%d", sosID),
		"sender_name": senderName,
		"latitude":   fmt.Sprintf("%v", latitude),
		"longitude":  fmt.Sprintf("%v", longitude),
	})
	if err != nil {
		log.Printf("[SOS PUSH ERROR] token=%s error=%v", fcmToken, err)
	}
}

// SendSosResolvedPush tells a protector's device that an SOS they were
// watching has been marked safe. The app uses this to cancel the ongoing,
// non-dismissible local notification for that alert.
//
// Without this, the "EMERGENCY: X needs help" notification (posted with
// ongoing:true/autoCancel:false so it can't be swiped away by accident)
// just sits in the tray forever once resolved. On several Android builds a
// full-screen-intent notification is suppressed while an earlier one from
// the same app is still showing/ongoing - so the very next real SOS to the
// same phone can silently fail to alert (no heads-up, no sound) even
// though the FCM message was delivered and technically "worked". Clearing
// the stale notification the moment the alert resolves closes that gap.
func SendSosResolvedPush(fcmToken string, sosID int64) {
	err := sendMessage(fcmToken, map[string]string{
		"type":   "sos_resolved",
		"sos_id": fmt.Sprintf("%d", sosID),
	})
	if err != nil {
		log.Printf("[SOS RESOLVED PUSH ERROR] token=%s error=%v", fcmToken, err)
	}
}

// SendAdminAlertPush mirrors send_admin_alert_push(fcm_token,
// notification_id, title, body, severity, district) in
// services/push_service.py.
func SendAdminAlertPush(fcmToken string, notificationID int64, title, body, severity string, district *string) {
	districtVal := ""
	if district != nil {
		districtVal = *district
	}
	err := sendMessage(fcmToken, map[string]string{
		"type":            "admin_alert",
		"notification_id": fmt.Sprintf("%d", notificationID),
		"title":           title,
		"body":            body,
		"severity":        severity,
		"district":        districtVal,
	})
	if err != nil {
		log.Printf("[ADMIN ALERT PUSH ERROR] token=%s error=%v", fcmToken, err)
	}
}
