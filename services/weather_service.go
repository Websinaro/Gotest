package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	weatherURL    = "https://api.open-meteo.com/v1/forecast"
	airQualityURL = "https://air-quality-api.open-meteo.com/v1/air-quality"
	geocodeURL    = "https://geocoding-api.open-meteo.com/v1/reverse"
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

// ErrWeatherUnavailable mirrors the (httpx.TimeoutException, httpx.ConnectError,
// httpx.HTTPStatusError) tuple that routes/weather.py catches to return a 503.
type ErrWeatherUnavailable struct {
	Err error
}

func (e *ErrWeatherUnavailable) Error() string { return e.Err.Error() }
func (e *ErrWeatherUnavailable) Unwrap() error { return e.Err }

func buildQuery(params map[string]string) string {
	v := url.Values{}
	for k, val := range params {
		v.Set(k, val)
	}
	return v.Encode()
}

// getWithRetry mirrors _get_with_retry: the primary weather call gets one
// near-instant retry (0.6s backoff) on a transient failure before giving up.
func getWithRetry(rawURL string, params map[string]string, attempts int) (*http.Response, error) {
	fullURL := rawURL + "?" + buildQuery(params)

	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		resp, err := httpClient.Get(fullURL)
		if err == nil {
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return resp, nil
			}
			resp.Body.Close()
			lastErr = fmt.Errorf("unexpected status %d from %s", resp.StatusCode, rawURL)
		} else {
			lastErr = err
		}
		if attempt < attempts-1 {
			time.Sleep(600 * time.Millisecond)
		}
	}
	return nil, &ErrWeatherUnavailable{Err: lastErr}
}

func decodeJSON(resp *http.Response, dst interface{}) error {
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(dst)
}

// WeatherResult mirrors the dict returned by fetch_weather().
type WeatherResult struct {
	Weather   map[string]interface{}
	Air       map[string]interface{} // nil if the air-quality call failed
	PlaceName *string
}

// FetchWeather mirrors fetch_weather(lat, lon) in services/weather_service.py.
func FetchWeather(lat, lon float64) (*WeatherResult, error) {
	weatherParams := map[string]string{
		"latitude":      fmt.Sprintf("%v", lat),
		"longitude":     fmt.Sprintf("%v", lon),
		"current":       "temperature_2m,relative_humidity_2m,apparent_temperature,precipitation,rain,weather_code,cloud_cover,pressure_msl,wind_speed_10m,wind_direction_10m,wind_gusts_10m,uv_index,is_day",
		"hourly":        "temperature_2m,relative_humidity_2m,apparent_temperature,precipitation_probability,precipitation,weather_code,wind_speed_10m,wind_gusts_10m,uv_index,dew_point_2m,visibility",
		"daily":         "weather_code,temperature_2m_max,temperature_2m_min,apparent_temperature_max,apparent_temperature_min,sunrise,sunset,uv_index_max,precipitation_sum,precipitation_probability_max,wind_speed_10m_max,wind_gusts_10m_max",
		"timezone":      "Asia/Kolkata",
		"forecast_days": "7",
	}
	airParams := map[string]string{
		"latitude":  fmt.Sprintf("%v", lat),
		"longitude": fmt.Sprintf("%v", lon),
		"current":   "us_aqi,pm2_5,pm10,ozone,carbon_monoxide",
		"timezone":  "Asia/Kolkata",
	}
	geocodeParams := map[string]string{
		"latitude":  fmt.Sprintf("%v", lat),
		"longitude": fmt.Sprintf("%v", lon),
		"language":  "en",
	}

	weatherResp, err := getWithRetry(weatherURL, weatherParams, 2)
	if err != nil {
		return nil, err
	}
	var weatherData map[string]interface{}
	if err := decodeJSON(weatherResp, &weatherData); err != nil {
		return nil, &ErrWeatherUnavailable{Err: err}
	}

	var placeName *string
	if geoResp, err := httpClient.Get(geocodeURL + "?" + buildQuery(geocodeParams)); err == nil {
		if geoResp.StatusCode >= 200 && geoResp.StatusCode < 300 {
			var geoData map[string]interface{}
			if decodeJSON(geoResp, &geoData) == nil {
				if results, ok := geoData["results"].([]interface{}); ok && len(results) > 0 {
					if top, ok := results[0].(map[string]interface{}); ok {
						name, _ := top["name"].(string)
						admin1, _ := top["admin1"].(string)
						formatted := fmt.Sprintf("%s, %s", name, admin1)
						placeName = &formatted
					}
				}
			}
		} else {
			geoResp.Body.Close()
		}
	}

	var airData map[string]interface{}
	if airResp, err := httpClient.Get(airQualityURL + "?" + buildQuery(airParams)); err == nil {
		if airResp.StatusCode >= 200 && airResp.StatusCode < 300 {
			decodeJSON(airResp, &airData)
		} else {
			airResp.Body.Close()
		}
	}

	return &WeatherResult{Weather: weatherData, Air: airData, PlaceName: placeName}, nil
}

// FetchCurrentOnly mirrors fetch_current_only(lat, lon) in
// services/weather_service.py, used by the Kerala map endpoint.
func FetchCurrentOnly(lat, lon float64) (map[string]interface{}, error) {
	params := map[string]string{
		"latitude":      fmt.Sprintf("%v", lat),
		"longitude":     fmt.Sprintf("%v", lon),
		"current":       "temperature_2m,relative_humidity_2m,precipitation,weather_code,wind_speed_10m,wind_direction_10m,wind_gusts_10m",
		"hourly":        "precipitation_probability",
		"timezone":      "Asia/Kolkata",
		"forecast_days": "1",
	}
	resp, err := getWithRetry(weatherURL, params, 2)
	if err != nil {
		return nil, err
	}
	var data map[string]interface{}
	if err := decodeJSON(resp, &data); err != nil {
		return nil, &ErrWeatherUnavailable{Err: err}
	}
	return data, nil
}
