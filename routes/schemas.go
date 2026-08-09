package routes

// ---- scheme/scheme.py ----

type UserCreate struct {
	Name       string  `json:"name"`
	Email      string  `json:"email"`
	Phone      string  `json:"phone"`
	Password   string  `json:"password"`
	District   string  `json:"district"`
	AccessCode *string `json:"access_code"`
}

type UserOut struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	District string `json:"district"`
	Role     string `json:"role"`
}

type TokenOut struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

// ---- scheme/sos_scheme.py ----

type SafetyContactCreate struct {
	Name         string  `json:"name"`
	Relationship *string `json:"relationship"`
	Phone        string  `json:"phone"`
	Email        *string `json:"email"`
	Address      *string `json:"address"`
}

type SafetyContactOut struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	Relationship *string `json:"relationship"`
	Phone        string  `json:"phone"`
	Email        *string `json:"email"`
	Address      *string `json:"address"`
}

type SosCreate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Message   *string `json:"message"`
}

type SosLocationUpdate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type SosOut struct {
	ID           int64   `json:"id"`
	UserID       int64   `json:"user_id"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	Status       string  `json:"status"`
	Message      *string `json:"message"`
	CreatedTime  string  `json:"created_time"`
	ResolvedTime *string `json:"resolved_time"`
}

type DeviceTokenCreate struct {
	FcmToken string  `json:"fcm_token"`
	Platform *string `json:"platform"`
}

// ---- scheme/notification_scheme.py ----

var allowedSeverities = map[string]bool{
	"green": true, "yellow": true, "orange": true, "light_red": true, "dark_red": true,
}

type NotificationCreate struct {
	Title    string  `json:"title"`
	Message  string  `json:"message"`
	Severity string  `json:"severity"`
	District *string `json:"district"`
}

type NotificationUpdate struct {
	Title         *string `json:"title"`
	Message       *string `json:"message"`
	Severity      *string `json:"severity"`
	District      *string `json:"district"`
	ClearDistrict bool    `json:"clear_district"`
	Active        *bool   `json:"active"`
}

type NotificationOut struct {
	ID            int64   `json:"id"`
	Title         string  `json:"title"`
	Message       string  `json:"message"`
	Severity      string  `json:"severity"`
	District      *string `json:"district"`
	CreatedBy     int64   `json:"created_by"`
	CreatedByName string  `json:"created_by_name"`
	Active        bool    `json:"active"`
	CreatedTime   string  `json:"created_time"`
	UpdatedTime   string  `json:"updated_time"`
}

// ---- scheme/president_scheme.py ----

type DistrictStat struct {
	District            string `json:"district"`
	RegisteredUsers     int    `json:"registered_users"`
	ActiveSos           int    `json:"active_sos"`
	ActiveNotifications int    `json:"active_notifications"`
}

type ActiveSosSummary struct {
	ID          int64   `json:"id"`
	UserID      int64   `json:"user_id"`
	UserName    string  `json:"user_name"`
	District    string  `json:"district"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Message     *string `json:"message"`
	CreatedTime string  `json:"created_time"`
}

type PresidentDashboard struct {
	TotalUsers               int                `json:"total_users"`
	TotalActiveSos           int                `json:"total_active_sos"`
	TotalActiveNotifications int                `json:"total_active_notifications"`
	Districts                []DistrictStat     `json:"districts"`
	ActiveSosAlerts          []ActiveSosSummary `json:"active_sos_alerts"`
}

// ---- scheme/weather_scheme.py ----

type CurrentWeather struct {
	Temperature  float64  `json:"temperature"`
	FeelsLike    float64  `json:"feels_like"`
	Humidity     float64  `json:"humidity"`
	Precipitation float64 `json:"precipitation"`
	Rain         float64  `json:"rain"`
	WeatherCode  int      `json:"weather_code"`
	WeatherLabel *string  `json:"weather_label"`
	WeatherIcon  *string  `json:"weather_icon"`
	CloudCover   float64  `json:"cloud_cover"`
	Pressure     float64  `json:"pressure"`
	WindSpeed    float64  `json:"wind_speed"`
	WindDirection float64 `json:"wind_direction"`
	WindGusts    float64  `json:"wind_gusts"`
	UVIndex      *float64 `json:"uv_index"`
	IsDay        int      `json:"is_day"`
}

type AirQuality struct {
	AQI            *float64 `json:"aqi"`
	PM25           *float64 `json:"pm2_5"`
	PM10           *float64 `json:"pm10"`
	Ozone          *float64 `json:"ozone"`
	CarbonMonoxide *float64 `json:"carbon_monoxide"`
}

type HourlyForecast struct {
	Time            []string   `json:"time"`
	Temperature     []float64  `json:"temperature"`
	FeelsLike       []float64  `json:"feels_like"`
	Humidity        []float64  `json:"humidity"`
	RainProbability []*float64 `json:"rain_probability"`
	Precipitation   []float64  `json:"precipitation"`
	WindSpeed       []float64  `json:"wind_speed"`
	WindGusts       []float64  `json:"wind_gusts"`
	UVIndex         []*float64 `json:"uv_index"`
	DewPoint        []float64  `json:"dew_point"`
	Visibility      []*float64 `json:"visibility"`
	WeatherCode     []int      `json:"weather_code"`
}

type DailyForecast struct {
	Date               []string   `json:"date"`
	TempMax            []float64  `json:"temp_max"`
	TempMin            []float64  `json:"temp_min"`
	FeelsLikeMax       []float64  `json:"feels_like_max"`
	FeelsLikeMin       []float64  `json:"feels_like_min"`
	Sunrise            []string   `json:"sunrise"`
	Sunset             []string   `json:"sunset"`
	UVIndexMax         []*float64 `json:"uv_index_max"`
	RainProbabilityMax []*float64 `json:"rain_probability_max"`
	PrecipitationSum   []float64  `json:"precipitation_sum"`
	WindSpeedMax       []float64  `json:"wind_speed_max"`
	WindGustsMax       []float64  `json:"wind_gusts_max"`
	WeatherCode        []int      `json:"weather_code"`
}

type WeatherResponse struct {
	LocationName *string        `json:"location_name"`
	Latitude     float64        `json:"latitude"`
	Longitude    float64        `json:"longitude"`
	AlertLevel   string         `json:"alert_level"`
	Current      CurrentWeather `json:"current"`
	AirQuality   *AirQuality    `json:"air_quality"`
	Hourly       HourlyForecast `json:"hourly"`
	Daily        DailyForecast  `json:"daily"`
}
