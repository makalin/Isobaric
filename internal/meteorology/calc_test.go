package meteorology

import (
	"math"
	"testing"
)

func TestDewpoint(t *testing.T) {
	// Test case: Temp=20C, RH=50% -> Dewpoint should be around 9.3C
	dp := Dewpoint(20.0, 50.0)
	expected := 9.27
	if math.Abs(dp-expected) > 0.1 {
		t.Errorf("Dewpoint(20.0, 50.0) = %f; expected %f", dp, expected)
	}

	// Test case: RH=100% -> Dewpoint should equal temperature
	dp = Dewpoint(15.0, 100.0)
	if math.Abs(dp-15.0) > 0.01 {
		t.Errorf("Dewpoint(15.0, 100.0) = %f; expected 15.0", dp)
	}
}

func TestLCLCeiling(t *testing.T) {
	// Temp=20C, Dewpoint=10C -> Spread=10C -> LCL should be 1250m
	lcl := LCLCeiling(20.0, 10.0)
	if lcl != 1250.0 {
		t.Errorf("LCLCeiling(20.0, 10.0) = %f; expected 1250.0", lcl)
	}

	// Temp=10C, Dewpoint=15C -> LCL should be 0.0
	lcl = LCLCeiling(10.0, 15.0)
	if lcl != 0.0 {
		t.Errorf("LCLCeiling(10.0, 15.0) = %f; expected 0.0", lcl)
	}
}

func TestWindDirectionCardinal(t *testing.T) {
	tests := []struct {
		deg  float64
		card string
	}{
		{0.0, "N"},
		{10.0, "N"},
		{22.5, "NNE"},
		{45.0, "NE"},
		{90.0, "E"},
		{180.0, "S"},
		{270.0, "W"},
		{350.0, "N"},
		{-45.0, "NW"}, // Negative wrap-around
		{385.0, "NNE"}, // Exceeding 360 wrap-around
	}

	for _, tt := range tests {
		res := WindDirectionCardinal(tt.deg)
		if res != tt.card {
			t.Errorf("WindDirectionCardinal(%f) = %s; expected %s", tt.deg, res, tt.card)
		}
	}
}

func TestPressureTendency(t *testing.T) {
	// Delta = 2.0 -> RISING
	_, tend := PressureTendency(1015.0, 1013.0)
	if tend != "RISING" {
		t.Errorf("PressureTendency(1015.0, 1013.0) = %s; expected RISING", tend)
	}

	// Delta = -2.0 -> FALLING
	_, tend = PressureTendency(1011.0, 1013.0)
	if tend != "FALLING" {
		t.Errorf("PressureTendency(1011.0, 1013.0) = %s; expected FALLING", tend)
	}

	// Delta = 0.5 -> STEADY
	_, tend = PressureTendency(1013.5, 1013.0)
	if tend != "STEADY" {
		t.Errorf("PressureTendency(1013.5, 1013.0) = %s; expected STEADY", tend)
	}
}
