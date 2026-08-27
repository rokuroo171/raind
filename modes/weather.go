package modes

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"time"
)

// Condition is the coarse weather state that drives the scene.
type Condition int

const (
	CondClear Condition = iota
	CondCloudy
	CondRain
	CondSnow
	CondThunder
)

// WeatherData is the live snapshot applied to the scene. Offline means the
// last fetch failed and raind is showing simulated weather instead.
type WeatherData struct {
	City          string
	Condition     Condition
	Temperature   float64
	Precipitation float64
	WindSpeed     float64
	WindDirection float64
	Offline       bool
	FetchedAt     time.Time
}

// WindVector returns the horizontal drift the wind causes (1 is a full
// screen-width lean, negative is left) and whether wind data exists.
// wind_direction_10m is meteorological: the direction the wind comes FROM.
func (wd WeatherData) WindVector() (vx float64, ok bool) {
	if wd.WindSpeed <= 0 {
		return 0, false
	}
	// blowing TOWARD dir+180, then rotate compass to math angle
	a := (wd.WindDirection + 90) * math.Pi / 180
	strength := min(1, wd.WindSpeed/30)
	return math.Cos(a) * strength * 0.8, true
}

// Intensity maps the current precipitation to a 0-1 storm strength used to
// scale particle density when the mode initializes.
func (wd WeatherData) Intensity() float64 {
	p := wd.Precipitation
	switch wd.Condition {
	case CondThunder:
		return 0.7 + min(0.3, p/20)
	case CondRain:
		return min(1, 0.35+p/6)
	case CondSnow:
		return min(1, 0.35+p/8)
	default:
		return 0
	}
}

// ModeForWeather picks the scene a weather condition should show.
func ModeForWeather(c Condition) Mode {
	switch c {
	case CondThunder:
		return ModeThunderstorm
	case CondSnow:
		return ModeSnow
	case CondRain:
		return ModeRain
	default:
		return ModeCalm
	}
}

// conditionFromCode maps a WMO weather code to a Condition.
// Codes follow https://open-meteo.com/en/docs (WMO 0-99).
func conditionFromCode(code int) Condition {
	switch {
	case code == 0:
		return CondClear
	case code <= 3, code == 45, code == 48:
		return CondCloudy
	case code >= 95:
		return CondThunder
	case code >= 71 && code <= 77, code >= 85 && code <= 86:
		return CondSnow
	default:
		return CondRain
	}
}

var weatherHTTP = &http.Client{Timeout: 6 * time.Second}

func getJSON(rawURL string, out any) error {
	resp, err := weatherHTTP.Get(rawURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("weather: %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

type geocodingResult struct {
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type geocodingResponse struct {
	Results []geocodingResult `json:"results"`
}

// GeocodeCity resolves a city name to coordinates via Open-Meteo's geocoder.
func GeocodeCity(name string) (lat, lon float64, resolved string, err error) {
	u := "https://geocoding-api.open-meteo.com/v1/search?name=" +
		url.QueryEscape(name) + "&count=1&language=en&format=json"
	var resp geocodingResponse
	if err := getJSON(u, &resp); err != nil {
		return 0, 0, "", err
	}
	if len(resp.Results) == 0 {
		return 0, 0, "", fmt.Errorf("weather: no city found for %q", name)
	}
	r := resp.Results[0]
	return r.Latitude, r.Longitude, r.Name, nil
}

type ipLocation struct {
	City      string  `json:"city"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// DetectCity resolves the user's location from their IP via ipwho.is.
func DetectCity() (lat, lon float64, city string, err error) {
	var loc ipLocation
	if err := getJSON("https://ipwho.is/", &loc); err != nil {
		return 0, 0, "", err
	}
	if loc.City == "" {
		return 0, 0, "", fmt.Errorf("weather: no city for this IP")
	}
	return loc.Latitude, loc.Longitude, loc.City, nil
}

type forecastCurrent struct {
	Temperature   float64 `json:"temperature_2m"`
	Precipitation float64 `json:"precipitation"`
	WeatherCode   int     `json:"weather_code"`
	WindSpeed     float64 `json:"wind_speed_10m"`
	WindDirection float64 `json:"wind_direction_10m"`
}

type forecastResponse struct {
	Current forecastCurrent `json:"current"`
}

func fetchForecast(lat, lon float64) (WeatherData, error) {
	u := fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f&current=temperature_2m,precipitation,weather_code,wind_speed_10m,wind_direction_10m",
		lat, lon)
	var resp forecastResponse
	if err := getJSON(u, &resp); err != nil {
		return WeatherData{}, err
	}
	return WeatherData{
		Condition:     conditionFromCode(resp.Current.WeatherCode),
		Temperature:   resp.Current.Temperature,
		Precipitation: resp.Current.Precipitation,
		WindSpeed:     resp.Current.WindSpeed,
		WindDirection: resp.Current.WindDirection,
	}, nil
}

// FetchWeatherForCity is the full pipeline: locate the city (from the name,
// or from the IP when the name is empty), fetch the current forecast, and
// return a snapshot. Failures come back as Offline data instead of errors so
// the render loop can keep running.
func FetchWeatherForCity(city string) WeatherData {
	var lat, lon float64
	var name string
	var err error
	if city == "" {
		lat, lon, name, err = DetectCity()
	} else {
		lat, lon, name, err = GeocodeCity(city)
	}
	if err != nil {
		return WeatherData{City: name, Offline: true, FetchedAt: time.Now()}
	}
	wd, err := fetchForecast(lat, lon)
	if err != nil {
		return WeatherData{City: name, Offline: true, FetchedAt: time.Now()}
	}
	wd.City = name
	wd.FetchedAt = time.Now()
	return wd
}
