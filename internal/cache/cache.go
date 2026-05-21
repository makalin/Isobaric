package cache

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/makalin/isobaric/internal/meteorology"
	"github.com/makalin/isobaric/internal/provider"
)

type CacheEntry struct {
	Timestamp   time.Time             `json:"timestamp"`
	WeatherData *provider.WeatherData `json:"weather_data"`
}

type SensorData struct {
	PressureHpa  float64 `json:"pressure_hpa"`
	TemperatureC float64 `json:"temperature_c"`
	Humidity     float64 `json:"humidity"`
	Timestamp    string  `json:"timestamp"`
}

// GetCacheDir returns the cache directory ~/.cache/isobaric (or OS equivalent).
func GetCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cacheDir := filepath.Join(home, ".cache", "isobaric")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}
	return cacheDir, nil
}

// GetConfigDir returns ~/.config/isobaric
func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configDir := filepath.Join(home, ".config", "isobaric")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}
	return configDir, nil
}

// GetCachePath returns the path for a specific query cache file.
func GetCachePath(lat, lon float64, providerName string) (string, error) {
	dir, err := GetCacheDir()
	if err != nil {
		return "", err
	}
	// Round coords to 2 decimals to prevent cache miss on micro-changes
	latKey := fmt.Sprintf("%.2f", lat)
	lonKey := fmt.Sprintf("%.2f", lon)
	fileName := fmt.Sprintf("cache_%s_%s_%s.json", latKey, lonKey, providerName)
	return filepath.Join(dir, fileName), nil
}

// ReadCache attempts to load a cached payload.
// Returns WeatherData and true if hit and valid (within TTL), else false.
func ReadCache(lat, lon float64, providerName string, ttlMinutes int) (*provider.WeatherData, bool) {
	path, err := GetCachePath(lat, lon, providerName)
	if err != nil {
		return nil, false
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer file.Close()

	var entry CacheEntry
	if err := json.NewDecoder(file).Decode(&entry); err != nil {
		return nil, false
	}

	// Check if cached version is still fresh
	if time.Since(entry.Timestamp).Minutes() < float64(ttlMinutes) {
		return entry.WeatherData, true
	}

	// Exists but expired
	return entry.WeatherData, false
}

// WriteCache saves a WeatherData payload to cache.
func WriteCache(lat, lon float64, providerName string, data *provider.WeatherData) error {
	path, err := GetCachePath(lat, lon, providerName)
	if err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	entry := CacheEntry{
		Timestamp:   time.Now(),
		WeatherData: data,
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(entry)
}

// LoadSensorData attempts to load sensor telemetry from ~/.config/isobaric/sensor.json.
// If file does not exist, it generates a simulated pressure falling trend (barometer drop).
func LoadSensorData() (*SensorData, bool) {
	cfgDir, err := GetConfigDir()
	if err != nil {
		return fallbackSensorData(), false
	}

	path := filepath.Join(cfgDir, "sensor.json")
	file, err := os.Open(path)
	if err != nil {
		// No sensor.json exists, return simulated fall
		return fallbackSensorData(), false
	}
	defer file.Close()

	var sd SensorData
	if err := json.NewDecoder(file).Decode(&sd); err != nil {
		return fallbackSensorData(), false
	}

	return &sd, true
}

func fallbackSensorData() *SensorData {
	// Simulate an approaching cold front (falling pressure)
	// base standard pressure is 1013.25 hPa, let's simulate a drop to 1008.4 hPa
	return &SensorData{
		PressureHpa:  1008.4,
		TemperatureC: 18.5,
		Humidity:     78.0,
		Timestamp:    time.Now().Format(time.RFC3339),
	}
}

// IntegrateSensor combines the last cached WeatherData with current sensor readings
// to calculate offline trend updates and returns the updated WeatherData.
func IntegrateSensor(cached *provider.WeatherData, sensor *SensorData) *provider.WeatherData {
	if cached == nil {
		// No cache to build upon; return a minimal WeatherData payload representing the sensor values
		t, err := time.Parse(time.RFC3339, sensor.Timestamp)
		if err != nil {
			t = time.Now()
		}

		dp := meteorology.Dewpoint(sensor.TemperatureC, sensor.Humidity)
		wd := &provider.WeatherData{
			Time:             t,
			Temperature:      sensor.TemperatureC,
			RelativeHumidity: sensor.Humidity,
			Dewpoint:         dp,
			SurfacePressure:  sensor.PressureHpa,
			SeaLevelPressure: sensor.PressureHpa, // Approx
			ProviderName:     "local-sensor",
			HourlyPressures:  []float64{sensor.PressureHpa},
			HourlyTimes:      []time.Time{t},
		}
		return wd
	}

	// Copy cached data
	updated := *cached
	updated.ProviderName = "local-sensor-hybrid"

	// Parse sensor timestamp
	sTime, err := time.Parse(time.RFC3339, sensor.Timestamp)
	if err != nil {
		sTime = time.Now()
	}
	updated.Time = sTime

	// Update telemetry from sensor
	updated.Temperature = sensor.TemperatureC
	updated.RelativeHumidity = sensor.Humidity
	updated.Dewpoint = meteorology.Dewpoint(sensor.TemperatureC, sensor.Humidity)
	updated.SurfacePressure = sensor.PressureHpa
	updated.SeaLevelPressure = sensor.PressureHpa // Approximation

	// Update hourly series for charts and tendency calculations
	// Check if we need to insert the sensor reading or if it matches the current hour
	if len(updated.HourlyPressures) > 0 {
		// Drop first hourly value and append sensor value to simulate moving time window
		// Keep the size matching to let charts draw correctly
		updated.HourlyPressures = append(updated.HourlyPressures[1:], sensor.PressureHpa)
		updated.HourlyTimes = append(updated.HourlyTimes[1:], sTime)
		updated.HourlyTemperatures = append(updated.HourlyTemperatures[1:], sensor.TemperatureC)
	} else {
		updated.HourlyPressures = []float64{sensor.PressureHpa}
		updated.HourlyTimes = []time.Time{sTime}
		updated.HourlyTemperatures = []float64{sensor.TemperatureC}
	}

	// Recalculate 3-hour tendency:
	// We locate the pressure 3 hours ago from the hourly slice.
	// 3 hours ago is len(updated.HourlyPressures) - 4 (if entries are hourly, index -3)
	// Let's safe-check bounds.
	hCount := len(updated.HourlyPressures)
	if hCount >= 4 {
		// Current is last index (hCount-1)
		// 3h ago is index (hCount-4)
		p3hAgo := updated.HourlyPressures[hCount-4]
		_, tend := meteorology.PressureTendency(sensor.PressureHpa, p3hAgo)
		// We will log/display this tendency
		_ = tend
	}

	// Since we are offline, CAPE and Solar Irradiance can be modeled:
	// E.g. Solar irradiance is 0 if night, or estimated based on time.
	// But let's keep the last cached values as persistent baseline.
	// If the time shift is significant (e.g. night vs day), we can zero out irradiance.
	hour := sTime.Hour()
	if hour < 6 || hour > 19 {
		updated.SolarIrradiance = 0.0
	} else if updated.SolarIrradiance > 0 {
		// Scale based on solar angle approximation
		scale := math.Sin(math.Pi * float64(hour-6) / 13.0)
		updated.SolarIrradiance = math.Max(0.0, updated.SolarIrradiance*scale)
	}

	return &updated
}
