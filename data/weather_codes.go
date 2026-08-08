package data

// WeatherInfo mirrors the {"label":..., "icon":...} dicts in
// data/weather_codes.py.
type WeatherInfo struct {
	Label string `json:"label"`
	Icon  string `json:"icon"`
}

var weatherCodes = map[int]WeatherInfo{
	0:  {"Clear Sky", "sunny"},
	1:  {"Mainly Clear", "mostly_sunny"},
	2:  {"Partly Cloudy", "partly_cloudy"},
	3:  {"Overcast", "cloudy"},
	45: {"Fog", "fog"},
	48: {"Depositing Rime Fog", "fog"},
	51: {"Light Drizzle", "drizzle"},
	53: {"Moderate Drizzle", "drizzle"},
	55: {"Dense Drizzle", "drizzle"},
	56: {"Light Freezing Drizzle", "drizzle"},
	57: {"Dense Freezing Drizzle", "drizzle"},
	61: {"Slight Rain", "rain"},
	63: {"Moderate Rain", "rain"},
	65: {"Heavy Rain", "heavy_rain"},
	66: {"Light Freezing Rain", "rain"},
	67: {"Heavy Freezing Rain", "heavy_rain"},
	71: {"Slight Snow Fall", "snow"},
	73: {"Moderate Snow Fall", "snow"},
	75: {"Heavy Snow Fall", "snow"},
	77: {"Snow Grains", "snow"},
	80: {"Slight Rain Showers", "rain_showers"},
	81: {"Moderate Rain Showers", "rain_showers"},
	82: {"Violent Rain Showers", "heavy_rain"},
	85: {"Slight Snow Showers", "snow"},
	86: {"Heavy Snow Showers", "snow"},
	95: {"Thunderstorm", "thunderstorm"},
	96: {"Thunderstorm with Slight Hail", "thunderstorm_hail"},
	99: {"Thunderstorm with Heavy Hail", "thunderstorm_hail"},
}

// GetWeatherInfo mirrors get_weather_info(code) in data/weather_codes.py.
func GetWeatherInfo(code int) WeatherInfo {
	if info, ok := weatherCodes[code]; ok {
		return info
	}
	return WeatherInfo{"Unknown", "unknown"}
}
