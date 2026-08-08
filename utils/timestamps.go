package utils

import (
	"fmt"
	"time"
)

var istLocation *time.Location

func init() {
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		loc = time.FixedZone("IST", 5*3600+30*60)
	}
	istLocation = loc
}

// UTCNowStr mirrors utc_now_str() in utils/timestamps.py: always emits all
// 6 microsecond digits (Asia/Kolkata wall clock, despite the name - that
// matches the original, which is never actually called from a wired-up
// route, but is kept for parity).
func UTCNowStr() string {
	return time.Now().In(istLocation).Format("2006-01-02 15:04:05.000000")
}

// PyUTCNowStr mirrors str(datetime.utcnow()) as used directly in
// routes/auth.py, safety_contacts.py, sos.py, device_token.py and
// notifications.py: true UTC, and - matching Python's datetime.__str__ -
// the ".ffffff" microsecond suffix is *omitted* entirely when microseconds
// happen to be exactly zero.
func PyUTCNowStr() string {
	now := time.Now().UTC()
	if now.Nanosecond() == 0 {
		return now.Format("2006-01-02 15:04:05")
	}
	micro := now.Nanosecond() / 1000
	return fmt.Sprintf("%s.%06d", now.Format("2006-01-02 15:04:05"), micro)
}
