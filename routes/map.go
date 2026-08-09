package routes

import (
	"net/http"
	"sync"
	"time"

	"kabackend/data"
	"kabackend/services"
	"kabackend/utils"
)

type districtWeatherResult struct {
	District        string  `json:"district"`
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
	Temperature     float64 `json:"temperature"`
	Humidity        float64 `json:"humidity"`
	RainProbability float64 `json:"rain_probability"`
	WeatherCode     int     `json:"weather_code"`
	WeatherLabel    string  `json:"weather_label"`
	WindSpeed       float64 `json:"wind_speed"`
	WindDirection   float64 `json:"wind_direction"`
	WindGusts       float64 `json:"wind_gusts"`
	AlertLevel      string  `json:"alert_level"`
}

var mapSemaphore = make(chan struct{}, 5) // max 5 in-flight requests at once

func districtWeather(name string, coords data.DistrictCoords) *districtWeatherResult {
	mapSemaphore <- struct{}{}
	defer func() { <-mapSemaphore }()

	var weatherData map[string]interface{}
	var err error
	for attempt := 0; attempt < 2; attempt++ { // one retry on transient failure
		weatherData, err = services.FetchCurrentOnly(coords.Lat, coords.Lon)
		if err == nil {
			break
		}
		if attempt == 1 {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	if weatherData == nil {
		return nil
	}

	current := subMap(weatherData, "current")
	hourly := subMap(weatherData, "hourly")
	hourlyTimes := toStringSlice(mapGet(hourly, "time"))
	hourIndex := utils.GetCurrentHourIndex(hourlyTimes)

	rainProbs := toFloat64PtrSlice(mapGet(hourly, "precipitation_probability"))
	rainProb := 0.0
	if hourIndex < len(rainProbs) && rainProbs[hourIndex] != nil {
		rainProb = *rainProbs[hourIndex]
	}

	weatherCode := toInt(mapGet(current, "weather_code"))
	windSpeed := toFloat64(mapGet(current, "wind_speed_10m"))

	alertLevel := data.GetAlertLevel(weatherCode, rainProb, windSpeed)

	return &districtWeatherResult{
		District:        name,
		Latitude:        coords.Lat,
		Longitude:       coords.Lon,
		Temperature:     toFloat64(mapGet(current, "temperature_2m")),
		Humidity:        toFloat64(mapGet(current, "relative_humidity_2m")),
		RainProbability: rainProb,
		WeatherCode:     weatherCode,
		WeatherLabel:    data.GetWeatherInfo(weatherCode).Label,
		WindSpeed:       windSpeed,
		WindDirection:   toFloat64(mapGet(current, "wind_direction_10m")),
		WindGusts:       toFloat64(mapGet(current, "wind_gusts_10m")),
		AlertLevel:      alertLevel,
	}
}

// KeralaMapHandler mirrors GET /weather/kerala-map in routes/map.py.
func KeralaMapHandler(w http.ResponseWriter, r *http.Request) {
	type indexedResult struct {
		idx    int
		result *districtWeatherResult
	}

	names := make([]string, 0, len(data.KeralaDistricts))
	for name := range data.KeralaDistricts {
		names = append(names, name)
	}

	results := make([]*districtWeatherResult, len(names))
	var wg sync.WaitGroup
	resultsCh := make(chan indexedResult, len(names))

	for i, name := range names {
		wg.Add(1)
		go func(idx int, n string) {
			defer wg.Done()
			resultsCh <- indexedResult{idx: idx, result: districtWeather(n, data.KeralaDistricts[n])}
		}(i, name)
	}

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	for r := range resultsCh {
		results[r.idx] = r.result
	}

	districts := make([]districtWeatherResult, 0, len(results))
	for _, res := range results {
		if res != nil {
			districts = append(districts, *res)
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"districts": districts,
		"total":     len(data.KeralaDistricts),
		"loaded":    len(districts),
	})
}
