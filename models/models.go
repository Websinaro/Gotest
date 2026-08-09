package models

// User mirrors the `users` table / model.User.
type User struct {
	ID          int64
	Name        string
	Email       string
	Phone       string
	Password    string
	District    string
	Role        string
	CreatedTime string
}

// SafetyContact mirrors the `safety_contacts` table / model.SafetyContact.
type SafetyContact struct {
	ID           int64
	UserID       int64
	Name         string
	Relationship *string
	Phone        string
	Email        *string
	Address      *string
	CreatedTime  string
}

// SosAlert mirrors the `sos_alerts` table / model.SosAlert.
type SosAlert struct {
	ID           int64
	UserID       int64
	Latitude     float64
	Longitude    float64
	Status       string
	Message      *string
	CreatedTime  string
	ResolvedTime *string
}

// DeviceToken mirrors the `device_tokens` table / model.DeviceToken.
type DeviceToken struct {
	ID          int64
	UserID      int64
	FcmToken    string
	Platform    *string
	UpdatedTime string
}

// Notification mirrors the `notifications` table / model.Notification.
type Notification struct {
	ID            int64
	Title         string
	Message       string
	Severity      string
	District      *string
	CreatedBy     int64
	CreatedByName string
	Active        bool
	CreatedTime   string
	UpdatedTime   string
}
