package routes

import (
	"net/http"
	"strconv"

	"kabackend/data"
	"kabackend/services"
	"kabackend/utils"
)

func toFloat64(v interface{}) float64 {
	if v == nil {
		return 0
	}
	f, _ := v.(float64)
	return f
}

func toFloat64Ptr(v interface{}) *float64 {
	if v == nil {
		return nil
	}
	f, ok := v.(float64)
	if !ok {
		return nil
	}
	return &f
}

func toInt(v interface{}) int {
	f := toFloat64(v)
	return int(f)
}

func toStringSlice(v interface{}) []string {
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, len(arr))
	for i, item := range arr {
		s, _ := item.(string)
		out[i] = s
	}
	return out
}

func toFloat64Slice(v interface{}) []float64 {
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]float64, len(arr))
	for i, item := range arr {
		out[i] = toFloat64(item)
	}
	return out
}

func toFloat64PtrSlice(v interface{}) []*float64 {
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]*float64, len(arr))
	for i, item := range arr {
		out[i] = toFloat64Ptr(item)
	}
	return out
}

func toIntSlice(v interface{}) []int {
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]int, len(arr))
	for i, item := range arr {
		out[i] = toInt(item)
	}
	return out
}

func mapGet(m map[string]interface{}, key string) interface{} {
	if m == nil {
		return nil
	}
	return m[key]
}

func subMap(m map[string]interface{}, key string) map[string]interface{} {
	v, _ := mapGet(m, key).(map[string]interface{})
	return v
}

// WeatherHandler mirrors GET /weather in routes/weather.py.
func WeatherHandler(w http.ResponseWriter, r *http.Request) {
	latStr := r.URL.Query().Get("lat")
	lonStr := r.URL.Query().Get("lon")

	lat, err1 := strconv.ParseFloat(latStr, 64)
	lon, err2 := strconv.ParseFloat(lonStr, 64)
	if err1 != nil || err2 != nil || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		writeError(w, http.StatusUnprocessableEntity, "lat and lon are required and must be valid coordinates")
		return
	}

	result, err := services.FetchWeather(lat, lon)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "Weather data is temporarily unavailable. Please try again in a few seconds.")
		return
	}

	current := subMap(result.Weather, "current")
	hourly := subMap(result.Weather, "hourly")
	daily := subMap(result.Weather, "daily")

	weatherCode := toInt(mapGet(current, "weather_code"))
	info := data.GetWeatherInfo(weatherCode)
	label := info.Label
	icon := info.Icon

	resp := WeatherResponse{
		LocationName: result.PlaceName,
		Latitude:     lat,
		Longitude:    lon,
		Current: CurrentWeather{
			Temperature:   toFloat64(mapGet(current, "temperature_2m")),
			FeelsLike:     toFloat64(mapGet(current, "apparent_temperature")),
			Humidity:      toFloat64(mapGet(current, "relative_humidity_2m")),
			Precipitation: toFloat64(mapGet(current, "precipitation")),
			Rain:          toFloat64(mapGet(current, "rain")),
			WeatherCode:   weatherCode,
			WeatherLabel:  &label,
			WeatherIcon:   &icon,
			CloudCover:    toFloat64(mapGet(current, "cloud_cover")),
			Pressure:      toFloat64(mapGet(current, "pressure_msl")),
			WindSpeed:     toFloat64(mapGet(current, "wind_speed_10m")),
			WindDirection: toFloat64(mapGet(current, "wind_direction_10m")),
			WindGusts:     toFloat64(mapGet(current, "wind_gusts_10m")),
			UVIndex:       toFloat64Ptr(mapGet(current, "uv_index")),
			IsDay:         toInt(mapGet(current, "is_day")),
		},
		Hourly: HourlyForecast{
			Time:            toStringSlice(mapGet(hourly, "time")),
			Temperature:     toFloat64Slice(mapGet(hourly, "temperature_2m")),
			FeelsLike:       toFloat64Slice(mapGet(hourly, "apparent_temperature")),
			Humidity:        toFloat64Slice(mapGet(hourly, "relative_humidity_2m")),
			RainProbability: toFloat64PtrSlice(mapGet(hourly, "precipitation_probability")),
			Precipitation:   toFloat64Slice(mapGet(hourly, "precipitation")),
			WindSpeed:       toFloat64Slice(mapGet(hourly, "wind_speed_10m")),
			WindGusts:       toFloat64Slice(mapGet(hourly, "wind_gusts_10m")),
			UVIndex:         toFloat64PtrSlice(mapGet(hourly, "uv_index")),
			DewPoint:        toFloat64Slice(mapGet(hourly, "dew_point_2m")),
			Visibility:      toFloat64PtrSlice(mapGet(hourly, "visibility")),
			WeatherCode:     toIntSlice(mapGet(hourly, "weather_code")),
		},
		Daily: DailyForecast{
			Date:               toStringSlice(mapGet(daily, "time")),
			TempMax:            toFloat64Slice(mapGet(daily, "temperature_2m_max")),
			TempMin:            toFloat64Slice(mapGet(daily, "temperature_2m_min")),
			FeelsLikeMax:       toFloat64Slice(mapGet(daily, "apparent_temperature_max")),
			FeelsLikeMin:       toFloat64Slice(mapGet(daily, "apparent_temperature_min")),
			Sunrise:            toStringSlice(mapGet(daily, "sunrise")),
			Sunset:             toStringSlice(mapGet(daily, "sunset")),
			UVIndexMax:         toFloat64PtrSlice(mapGet(daily, "uv_index_max")),
			RainProbabilityMax: toFloat64PtrSlice(mapGet(daily, "precipitation_probability_max")),
			PrecipitationSum:   toFloat64Slice(mapGet(daily, "precipitation_sum")),
			WindSpeedMax:       toFloat64Slice(mapGet(daily, "wind_speed_10m_max")),
			WindGustsMax:       toFloat64Slice(mapGet(daily, "wind_gusts_10m_max")),
			WeatherCode:        toIntSlice(mapGet(daily, "weather_code")),
		},
	}

	if result.Air != nil {
		airCurrent := subMap(result.Air, "current")
		resp.AirQuality = &AirQuality{
			AQI:            toFloat64Ptr(mapGet(airCurrent, "us_aqi")),
			PM25:           toFloat64Ptr(mapGet(airCurrent, "pm2_5")),
			PM10:           toFloat64Ptr(mapGet(airCurrent, "pm10")),
			Ozone:          toFloat64Ptr(mapGet(airCurrent, "ozone")),
			CarbonMonoxide: toFloat64Ptr(mapGet(airCurrent, "carbon_monoxide")),
		}
	}

	hourIndex := utils.GetCurrentHourIndex(resp.Hourly.Time)
	rainProb := 0.0
	if hourIndex < len(resp.Hourly.RainProbability) && resp.Hourly.RainProbability[hourIndex] != nil {
		rainProb = *resp.Hourly.RainProbability[hourIndex]
	}
	resp.AlertLevel = data.GetAlertLevel(resp.Current.WeatherCode, rainProb, resp.Current.WindSpeed)

	writeJSON(w, http.StatusOK, resp)
}
