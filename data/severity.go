package data

func intIn(v int, set ...int) bool {
	for _, s := range set {
		if v == s {
			return true
		}
	}
	return false
}

// GetAlertLevel mirrors get_alert_level(weather_code, rain_probability,
// wind_speed) in data/severity.py.
func GetAlertLevel(weatherCode int, rainProbability float64, windSpeed float64) string {
	// DARK RED: very high risk — cyclonic / extreme
	if intIn(weatherCode, 65, 67, 82, 96, 99) || rainProbability >= 90 || windSpeed >= 62 {
		return "dark_red"
	}

	// LIGHT RED: high risk
	if intIn(weatherCode, 63, 66, 80, 81, 95) || rainProbability >= 70 || windSpeed >= 45 {
		return "light_red"
	}

	// ORANGE: risk
	if intIn(weatherCode, 51, 53, 55, 56, 57, 61, 85, 86) || rainProbability >= 50 || windSpeed >= 30 {
		return "orange"
	}

	// YELLOW: low risk
	if rainProbability >= 20 || windSpeed >= 15 || intIn(weatherCode, 2, 3, 45, 48) {
		return "yellow"
	}

	// GREEN: safe
	return "green"
}
