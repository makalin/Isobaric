package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type WeatherData struct {
	Latitude           float64
	Longitude          float64
	Time               time.Time
	Temperature        float64   // Celsius
	RelativeHumidity   float64   // %
	Dewpoint           float64   // Celsius
	SurfacePressure    float64   // hPa
	SeaLevelPressure   float64   // hPa (Altimeter setting equivalent)
	WindSpeed          float64   // m/s
	WindDirection      float64   // degrees
	WindGusts          float64   // m/s
	Cape               float64   // J/kg
	SolarIrradiance    float64   // W/m^2
	HourlyPressures    []float64 // past 24 hours + forecast
	HourlyTimes        []time.Time
	HourlyTemperatures []float64
	ProviderName       string
}

type Provider interface {
	FetchWeather(ctx context.Context, lat, lon float64) (*WeatherData, error)
	Name() string
}

// OpenMeteoProvider queries the Open-Meteo API
type OpenMeteoProvider struct {
	Client *http.Client
}

func NewOpenMeteoProvider() *OpenMeteoProvider {
	return &OpenMeteoProvider{
		Client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *OpenMeteoProvider) Name() string {
	return "open-meteo"
}

type openMeteoResponse struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Current   struct {
		Time               string  `json:"time"`
		Temperature2m      float64 `json:"temperature_2m"`
		RelativeHumidity2m float64 `json:"relative_humidity_2m"`
		DewPoint2m         float64 `json:"dew_point_2m"`
		SurfacePressure    float64 `json:"surface_pressure"`
		PressureMsl        float64 `json:"pressure_msl"`
		WindSpeed10m       float64 `json:"wind_speed_10m"`
		WindDirection10m   float64 `json:"wind_direction_10m"`
		WindGusts10m       float64 `json:"wind_gusts_10m"`
	} `json:"current"`
	Hourly struct {
		Time               []string  `json:"time"`
		PressureMsl        []float64 `json:"pressure_msl"`
		Temperature2m      []float64 `json:"temperature_2m"`
		Cape               []float64 `json:"cape"`
		ShortwaveRadiation []float64 `json:"shortwave_radiation"`
	} `json:"hourly"`
}

func (p *OpenMeteoProvider) FetchWeather(ctx context.Context, lat, lon float64) (*WeatherData, error) {
	url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&current=temperature_2m,relative_humidity_2m,dew_point_2m,surface_pressure,pressure_msl,wind_speed_10m,wind_direction_10m,wind_gusts_10m&hourly=pressure_msl,temperature_2m,cape,shortwave_radiation&wind_speed_unit=ms&past_hours=24&forecast_days=1", lat, lon)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("received non-200 status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var data openMeteoResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode json: %w", err)
	}

	parsedTime, err := time.Parse("2006-01-02T15:04", data.Current.Time)
	if err != nil {
		parsedTime = time.Now()
	}

	wd := &WeatherData{
		Latitude:         data.Latitude,
		Longitude:        data.Longitude,
		Time:             parsedTime,
		Temperature:      data.Current.Temperature2m,
		RelativeHumidity: data.Current.RelativeHumidity2m,
		Dewpoint:         data.Current.DewPoint2m,
		SurfacePressure:  data.Current.SurfacePressure,
		SeaLevelPressure: data.Current.PressureMsl,
		WindSpeed:        data.Current.WindSpeed10m,
		WindDirection:    data.Current.WindDirection10m,
		WindGusts:        data.Current.WindGusts10m,
		ProviderName:     p.Name(),
	}

	// Parse hourly lists
	length := len(data.Hourly.Time)
	wd.HourlyPressures = make([]float64, 0, length)
	wd.HourlyTimes = make([]time.Time, 0, length)
	wd.HourlyTemperatures = make([]float64, 0, length)

	currentVal, err := time.Parse("2006-01-02T15:04", data.Current.Time)
	if err != nil {
		currentVal = parsedTime
	}

	// Map current indices
	currentIndex := -1
	var minDiff int64 = 9223372036854775807 // MaxInt64

	for i, tStr := range data.Hourly.Time {
		tVal, err := time.Parse("2006-01-02T15:04", tStr)
		if err == nil {
			wd.HourlyTimes = append(wd.HourlyTimes, tVal)
			if i < len(data.Hourly.PressureMsl) {
				wd.HourlyPressures = append(wd.HourlyPressures, data.Hourly.PressureMsl[i])
			}
			if i < len(data.Hourly.Temperature2m) {
				wd.HourlyTemperatures = append(wd.HourlyTemperatures, data.Hourly.Temperature2m[i])
			}

			// Find closest index to the current observation time
			diff := currentVal.Sub(tVal).Nanoseconds()
			if diff < 0 {
				diff = -diff
			}
			if diff < minDiff {
				minDiff = diff
				currentIndex = i
			}
		}
	}

	// Map CAPE and Solar Irradiance
	if currentIndex != -1 {
		if currentIndex < len(data.Hourly.Cape) {
			wd.Cape = data.Hourly.Cape[currentIndex]
		}
		if currentIndex < len(data.Hourly.ShortwaveRadiation) {
			wd.SolarIrradiance = data.Hourly.ShortwaveRadiation[currentIndex]
		}

		// Truncate hourly lists to current time for accurate historical charting & tendency
		if currentIndex < len(wd.HourlyPressures) {
			wd.HourlyPressures = wd.HourlyPressures[:currentIndex+1]
		}
		if currentIndex < len(wd.HourlyTimes) {
			wd.HourlyTimes = wd.HourlyTimes[:currentIndex+1]
		}
		if currentIndex < len(wd.HourlyTemperatures) {
			wd.HourlyTemperatures = wd.HourlyTemperatures[:currentIndex+1]
		}
	}

	return wd, nil
}

// NoaaProvider queries the NOAA (US National Weather Service) API
type NoaaProvider struct {
	Client *http.Client
}

func NewNoaaProvider() *NoaaProvider {
	return &NoaaProvider{
		Client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *NoaaProvider) Name() string {
	return "noaa"
}

// NOAA grid points API response structural shells
type noaaPointsResponse struct {
	ForecastGridData string `json:"forecastGridData"`
	ForecastHourly   string `json:"forecastHourly"`
	Properties       struct {
		ForecastGridData string `json:"forecastGridData"`
		ForecastHourly   string `json:"forecastHourly"`
	} `json:"properties"`
}

type noaaGridResponse struct {
	Properties struct {
		Temperature struct {
			Values []struct {
				ValidTime string  `json:"validTime"`
				Value     float64 `json:"value"`
			} `json:"values"`
		} `json:"temperature"`
		Dewpoint struct {
			Values []struct {
				ValidTime string  `json:"validTime"`
				Value     float64 `json:"value"`
			} `json:"values"`
		} `json:"dewpoint"`
		RelativeHumidity struct {
			Values []struct {
				ValidTime string  `json:"validTime"`
				Value     float64 `json:"value"`
			} `json:"values"`
		} `json:"relativeHumidity"`
		SeaLevelPressure struct {
			Values []struct {
				ValidTime string  `json:"validTime"`
				Value     float64 `json:"value"`
			} `json:"values"`
		} `json:"seaLevelPressure"`
		WindSpeed struct {
			Values []struct {
				ValidTime string  `json:"validTime"`
				Value     float64 `json:"value"`
			} `json:"values"`
		} `json:"windSpeed"`
		WindDirection struct {
			Values []struct {
				ValidTime string  `json:"validTime"`
				Value     float64 `json:"value"`
			} `json:"values"`
		} `json:"windDirection"`
		WindGust struct {
			Values []struct {
				ValidTime string  `json:"validTime"`
				Value     float64 `json:"value"`
			} `json:"values"`
		} `json:"windGust"`
	} `json:"properties"`
}

func (p *NoaaProvider) FetchWeather(ctx context.Context, lat, lon float64) (*WeatherData, error) {
	// Step 1: Query Points API to get Office and Grid coordinates
	pointsUrl := fmt.Sprintf("https://api.weather.gov/points/%f,%f", lat, lon)
	req, err := http.NewRequestWithContext(ctx, "GET", pointsUrl, nil)
	if err != nil {
		return nil, err
	}

	// NOAA requires a custom user-agent string
	req.Header.Set("User-Agent", "IsobaricMeteorologicalEngine/1.0 (contact@isobaric.com)")
	req.Header.Set("Accept", "application/ld+json")

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("noaa points request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("noaa points API returned status %d (possibly coordinates outside US territory)", resp.StatusCode)
	}

	var points noaaPointsResponse
	if err := json.NewDecoder(resp.Body).Decode(&points); err != nil {
		return nil, err
	}

	gridDataUrl := points.ForecastGridData
	if gridDataUrl == "" {
		gridDataUrl = points.Properties.ForecastGridData
	}

	if gridDataUrl == "" {
		return nil, fmt.Errorf("noaa did not return grid data URL")
	}

	// Step 2: Fetch raw grid data
	gridReq, err := http.NewRequestWithContext(ctx, "GET", gridDataUrl, nil)
	if err != nil {
		return nil, err
	}
	gridReq.Header.Set("User-Agent", "IsobaricMeteorologicalEngine/1.0 (contact@isobaric.com)")

	gridResp, err := p.Client.Do(gridReq)
	if err != nil {
		return nil, fmt.Errorf("noaa grid data request failed: %w", err)
	}
	defer gridResp.Body.Close()

	if gridResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("noaa grid API returned status %d", gridResp.StatusCode)
	}

	var grid noaaGridResponse
	if err := json.NewDecoder(gridResp.Body).Decode(&grid); err != nil {
		return nil, err
	}

	// NOAA returns arrays of measurements over time. We map the first value or calculate.
	wd := &WeatherData{
		Latitude:     lat,
		Longitude:    lon,
		Time:         time.Now(),
		ProviderName: p.Name(),
	}

	// Helper to extract first valid float value from NOAA's JSON structure
	extractVal := func(vals []struct {
		ValidTime string  `json:"validTime"`
		Value     float64 `json:"value"`
	}) float64 {
		if len(vals) > 0 {
			return vals[0].Value
		}
		return 0.0
	}

	wd.Temperature = extractVal(grid.Properties.Temperature.Values)
	wd.Dewpoint = extractVal(grid.Properties.Dewpoint.Values)
	wd.RelativeHumidity = extractVal(grid.Properties.RelativeHumidity.Values)

	// NOAA wind speed is typically returned in km/h, convert to m/s
	wd.WindSpeed = extractVal(grid.Properties.WindSpeed.Values) / 3.6
	wd.WindDirection = extractVal(grid.Properties.WindDirection.Values)
	wd.WindGusts = extractVal(grid.Properties.WindGust.Values) / 3.6

	// NOAA SeaLevelPressure is in Pascals, convert to hPa (Pascal / 100)
	rawPa := extractVal(grid.Properties.SeaLevelPressure.Values)
	if rawPa > 0 {
		wd.SeaLevelPressure = rawPa / 100.0
	} else {
		wd.SeaLevelPressure = 1013.25 // standard fallback
	}
	wd.SurfacePressure = wd.SeaLevelPressure // Approximation for NOAA

	// Parse hourly pressures from sea level pressure list for display
	wd.HourlyPressures = make([]float64, 0, len(grid.Properties.SeaLevelPressure.Values))
	wd.HourlyTimes = make([]time.Time, 0, len(grid.Properties.SeaLevelPressure.Values))
	wd.HourlyTemperatures = make([]float64, 0, len(grid.Properties.Temperature.Values))

	for _, v := range grid.Properties.SeaLevelPressure.Values {
		// NOAA times look like "2026-05-21T06:00:00+00:00/PT1H" or similar ISO 8601 interval
		tVal, err := parseNoaaTime(v.ValidTime)
		if err == nil {
			wd.HourlyTimes = append(wd.HourlyTimes, tVal)
			wd.HourlyPressures = append(wd.HourlyPressures, v.Value/100.0)
		}
	}

	for _, v := range grid.Properties.Temperature.Values {
		wd.HourlyTemperatures = append(wd.HourlyTemperatures, v.Value)
	}

	// NOAA grid point data might not provide CAPE or SolarIrradiance easily in these parameters,
	// so we default them to 0 or estimates, or they are fetched by the client.
	wd.Cape = 0.0
	wd.SolarIrradiance = 0.0

	return wd, nil
}

func parseNoaaTime(isoStr string) (time.Time, error) {
	// Strip the ISO duration interval if present (e.g. "/PT1H")
	cleanStr := isoStr
	for i, char := range isoStr {
		if char == '/' {
			cleanStr = isoStr[:i]
			break
		}
	}
	// Parse ISO 8601 / RFC 3339 layout
	return time.Parse(time.RFC3339, cleanStr)
}
