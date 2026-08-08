package utils

import "time"

// GetCurrentHourIndex mirrors get_current_hour_index(hourly_times) in
// utils/time_index.py.
func GetCurrentHourIndex(hourlyTimes []string) int {
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		loc = time.FixedZone("IST", 5*3600+30*60)
	}
	nowKey := time.Now().In(loc).Format("2006-01-02T15:00")

	for i, t := range hourlyTimes {
		if t == nowKey {
			return i
		}
	}
	return 0
}
