package weather

import (
	"encoding/json"
	"fmt"

	"resty.dev/v3"
)

// WeatherClient is a thin wrapper around resty.Client for the WeatherAPI.
type WeatherClient struct {
	apiKey string
	client *resty.Client
}

// NewWeatherClient creates a new WeatherClient with the given API key.
func NewWeatherClient(apiKey string) *WeatherClient {
	return &WeatherClient{
		apiKey: apiKey,
		client: resty.New().
			SetBaseURL("https://api.weatherapi.com/v1").
			SetHeader("Accept", "application/json"),
	}
}

// GetForecast retrieves the forecast for the given latitude/longitude and number of days.
func (w *WeatherClient) GetForecast(lat, lon string, days int) (*WeatherResponse, error) {
	resp, err := w.client.R().
		SetQueryParams(map[string]string{
			"q":    fmt.Sprintf("%s %s", lat, lon),
			"days": fmt.Sprintf("%d", days),
			"key":  w.apiKey,
		}).
		Get("/forecast.json")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("weather API error: %s", resp.Status())
	}

	var weatherResp WeatherResponse
	if err := json.Unmarshal(resp.Bytes(), &weatherResp); err != nil {
		return nil, err
	}
	return &weatherResp, nil
}

// WeatherResponse represents the top‑level JSON structure returned by the API.
type WeatherResponse struct {
	Location Location `json:"location"`
	Current  Current  `json:"current"`
	Forecast Forecast `json:"forecast"`
}

// Location holds location metadata.
type Location struct {
	Name      string  `json:"name"`
	Region    string  `json:"region"`
	Country   string  `json:"country"`
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	TzID      string  `json:"tz_id"`
	Localtime string  `json:"localtime"`
}

// Current holds current weather data.
type Current struct {
	LastUpdated string    `json:"last_updated"`
	TempC       float64   `json:"temp_c"`
	IsDay       int       `json:"is_day"`
	Condition   Condition `json:"condition"`
	WindKph     float64   `json:"wind_kph"`
	WindDir     string    `json:"wind_dir"`
	PressureMb  float64   `json:"pressure_mb"`
	PrecipMm    float64   `json:"precip_mm"`
	Humidity    int       `json:"humidity"`
	Cloud       int       `json:"cloud"`
	FeelsLikeC  float64   `json:"feelslike_c"`
	WindChillC  float64   `json:"windchill_c"`
	HeatIndexC  float64   `json:"heatindex_c"`
	DewPointC   float64   `json:"dewpoint_c"`
	VisKm       float64   `json:"vis_km"`
	UV          float64   `json:"uv"`
	GustMph     float64   `json:"gust_mph"`
}

// Condition holds weather condition text.
type Condition struct {
	Text string `json:"text"`
}

// Forecast holds forecast data.
type Forecast struct {
	ForecastDay []ForecastDay `json:"forecastday"`
}

// ForecastDay represents a single day's forecast.
type ForecastDay struct {
	Date      string `json:"date"`
	DateEpoch int64  `json:"date_epoch"`
	Day       Day    `json:"day"`
}

// Day holds a detailed forecast for a day.
type Day struct {
	MaxTempC          float64   `json:"maxtemp_c"`
	MinTempC          float64   `json:"mintemp_c"`
	AvgTempC          float64   `json:"avgtemp_c"`
	MaxWindKph        float64   `json:"maxwind_kph"`
	TotalPrecipMm     float64   `json:"totalprecip_mm"`
	TotalSnowCm       float64   `json:"totalsnow_cm"`
	AvgVisKm          float64   `json:"avgvis_km"`
	AvgHumidity       float64   `json:"avghumidity"`
	DailyWillItRain   int       `json:"daily_will_it_rain"`
	DailyChanceOfRain int       `json:"daily_chance_of_rain"`
	DailyWillItSnow   int       `json:"daily_will_it_snow"`
	DailyChanceOfSnow int       `json:"daily_chance_of_snow"`
	Condition         Condition `json:"condition"`
	UV                float64   `json:"uv"`
}
