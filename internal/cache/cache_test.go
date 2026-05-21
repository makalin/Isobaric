package cache

import (
	"math"
	"testing"
	"time"

	"github.com/makalin/isobaric/internal/provider"
)

func TestGetCachePath(t *testing.T) {
	path, err := GetCachePath(40.9912, 29.4189, "open-meteo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain coordinates rounded to 2 decimals
	expectedPart := "cache_40.99_29.42_open-meteo.json"
	if !contains(path, expectedPart) {
		t.Errorf("expected path to contain %s, got %s", expectedPart, path)
	}
}

func TestIntegrateSensor(t *testing.T) {
	// Setup a cached data block
	now := time.Now()
	cached := &provider.WeatherData{
		Time:             now.Add(-4 * time.Hour),
		Temperature:      25.0,
		RelativeHumidity: 40.0,
		Dewpoint:         10.0,
		SurfacePressure:  1012.0,
		SeaLevelPressure: 1012.0,
		HourlyPressures:  []float64{1010.0, 1011.0, 1012.0, 1012.0},
		HourlyTimes:      []time.Time{now.Add(-3 * time.Hour), now.Add(-2 * time.Hour), now.Add(-1 * time.Hour), now},
		HourlyTemperatures: []float64{22.0, 23.0, 24.0, 25.0},
		SolarIrradiance:  400.0,
	}

	sensor := &SensorData{
		PressureHpa:  1009.0, // Significant drop
		TemperatureC: 20.0,
		Humidity:     80.0,
		Timestamp:    now.Format(time.RFC3339),
	}

	updated := IntegrateSensor(cached, sensor)

	if updated.ProviderName != "local-sensor-hybrid" {
		t.Errorf("expected provider name local-sensor-hybrid, got %s", updated.ProviderName)
	}

	if updated.SurfacePressure != 1009.0 {
		t.Errorf("expected pressure 1009.0, got %f", updated.SurfacePressure)
	}

	if updated.Temperature != 20.0 {
		t.Errorf("expected temp 20.0, got %f", updated.Temperature)
	}

	// RH=80%, Temp=20C -> Dewpoint should be around 16.4C
	if math.Abs(updated.Dewpoint-16.4) > 0.5 {
		t.Errorf("expected dewpoint around 16.4, got %f", updated.Dewpoint)
	}

	// The hourly pressures slice should have been shifted and the last entry should be the sensor value (1009.0)
	if len(updated.HourlyPressures) != 4 {
		t.Errorf("expected hourly pressures length 4, got %d", len(updated.HourlyPressures))
	}

	if updated.HourlyPressures[3] != 1009.0 {
		t.Errorf("expected last hourly pressure to be 1009.0, got %f", updated.HourlyPressures[3])
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && func() bool {
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}()
}
