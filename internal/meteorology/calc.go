package meteorology

import (
	"math"
)

// Dewpoint calculates the dew point in Celsius from ambient temperature (Celsius)
// and relative humidity (0-100) using the Magnus-Tetens formula.
func Dewpoint(temp float64, humidity float64) float64 {
	if humidity <= 0 {
		return -99.0
	}
	if humidity > 100 {
		humidity = 100
	}
	// Magnus-Tetens coefficients
	a := 17.625
	b := 243.04

	alpha := ((a * temp) / (b + temp)) + math.Log(humidity/100.0)
	dewpoint := (b * alpha) / (a - alpha)
	return dewpoint
}

// LCLCeiling estimates the Lifting Condensation Level (condensation ceiling)
// in meters based on the temperature-dewpoint spread.
// Return value is in meters. If dewpoint is higher than temperature, returns 0.
func LCLCeiling(temp float64, dewpoint float64) float64 {
	if dewpoint >= temp {
		return 0.0
	}
	// General approximation: 125 meters per degree Celsius spread
	return 125.0 * (temp - dewpoint)
}

// WindDirectionCardinal converts wind direction in degrees (0-360) to a cardinal string.
func WindDirectionCardinal(degrees float64) string {
	// Ensure degrees is normalized to [0, 360)
	degrees = math.Mod(degrees, 360.0)
	if degrees < 0 {
		degrees += 360.0
	}

	cardinals := []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
	idx := int(math.Floor((degrees + 11.25) / 22.5)) % 16
	return cardinals[idx]
}

// PressureTendency computes the 3-hour pressure trend vector.
// A delta > 1.5 hPa is RISING, < -1.5 hPa is FALLING, else STEADY.
func PressureTendency(currentPressure, pressure3hAgo float64) (float64, string) {
	delta := currentPressure - pressure3hAgo
	if delta > 1.5 {
		return delta, "RISING"
	} else if delta < -1.5 {
		return delta, "FALLING"
	}
	return delta, "STEADY"
}

// CtoF converts Celsius to Fahrenheit.
func CtoF(c float64) float64 {
	return c*1.8 + 32.0
}

// MpsToKnots converts meters/second to knots.
func MpsToKnots(mps float64) float64 {
	return mps * 1.94384
}

// MpsToMph converts meters/second to miles per hour.
func MpsToMph(mps float64) float64 {
	return mps * 2.23694
}

// KmhToKnots converts kilometers/hour to knots.
func KmhToKnots(kmh float64) float64 {
	return kmh * 0.539957
}

// KmhToMph converts kilometers/hour to miles per hour.
func KmhToMph(kmh float64) float64 {
	return kmh * 0.621371
}

// HpaToInHg converts hectopascals (hPa) to inches of mercury (inHg).
func HpaToInHg(hpa float64) float64 {
	return hpa * 0.02953
}

// MetersToFeet converts meters to feet.
func MetersToFeet(m float64) float64 {
	return m * 3.28084
}
