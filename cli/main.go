package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/makalin/isobaric/internal/cache"
	"github.com/makalin/isobaric/internal/config"
	"github.com/makalin/isobaric/internal/meteorology"
	"github.com/makalin/isobaric/internal/provider"
)

// ANSI colors for premium terminal UI styling
const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorPurple = "\033[35m"
	colorGray   = "\033[90m"
)

func main() {
	coordsFlag := flag.String("coords", "", "Latitude and longitude separated by comma (e.g. 40.99,29.41)")
	configFlag := flag.String("config", "", "Path to custom config.toml")
	modeFlag := flag.String("mode", "", "Override display mode: high_density or compact")
	verboseFlag := flag.Bool("verbose", false, "Print verbose log outputs to stderr")

	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config Load Warning: %v. Using defaults.\n", err)
	}

	// Apply CLI flags overrides
	if *modeFlag != "" {
		cfg.Display.Mode = *modeFlag
	}

	if *coordsFlag == "" {
		fmt.Fprintf(os.Stderr, "%sError: Coordinates --coords=lat,lon are required.%s\n\n", colorRed, colorReset)
		fmt.Fprintf(os.Stderr, "Usage: isobaric --coords=<latitude>,<longitude> [options]\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	lat, lon, err := parseCoords(*coordsFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError parsing coordinates: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}

	if *verboseFlag {
		fmt.Fprintf(os.Stderr, "[verbose] Coords: Lat=%.4f, Lon=%.4f\n", lat, lon)
		fmt.Fprintf(os.Stderr, "[verbose] Config units: %s\n", cfg.Engine.Units)
		fmt.Fprintf(os.Stderr, "[verbose] Config providers: Primary=%s, Secondary=%s\n", cfg.Providers.Primary, cfg.Providers.Secondary)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var data *provider.WeatherData
	var activeProvider string
	cacheHit := false
	isHybrid := false

	// Define providers mapping
	noaaProv := provider.NewNoaaProvider()
	omProv := provider.NewOpenMeteoProvider()

	getProvider := func(name string) provider.Provider {
		if name == "noaa" {
			return noaaProv
		}
		return omProv
	}

	// Setup provider fallback order
	primaryName := strings.ToLower(cfg.Providers.Primary)
	secondaryName := strings.ToLower(cfg.Providers.Secondary)
	if primaryName != "noaa" && primaryName != "open-meteo" {
		primaryName = "open-meteo"
	}
	if secondaryName != "noaa" && secondaryName != "open-meteo" {
		secondaryName = "open-meteo"
	}

	primary := getProvider(primaryName)
	secondary := getProvider(secondaryName)

	// Step 1: Read Cache (check freshness using TTL)
	cachedData, fresh := cache.ReadCache(lat, lon, primaryName, cfg.Engine.CacheTTLMinutes)
	if fresh {
		data = cachedData
		cacheHit = true
		activeProvider = primaryName
		if *verboseFlag {
			fmt.Fprintf(os.Stderr, "[verbose] Cache Hit for primary provider: %s\n", primaryName)
		}
	} else {
		// Cache is missing or expired. Let's try downloading new data.
		if *verboseFlag {
			if cachedData != nil {
				fmt.Fprintf(os.Stderr, "[verbose] Cache exists but is expired. Fetching fresh telemetry...\n")
			} else {
				fmt.Fprintf(os.Stderr, "[verbose] Cache miss. Fetching fresh telemetry...\n")
			}
		}

		// Try Primary
		data, err = primary.FetchWeather(ctx, lat, lon)
		if err == nil {
			activeProvider = primaryName
			_ = cache.WriteCache(lat, lon, primaryName, data)
		} else {
			if *verboseFlag {
				fmt.Fprintf(os.Stderr, "[verbose] Primary provider %s failed: %v\n", primaryName, err)
			}

			// Try Secondary
			data, err = secondary.FetchWeather(ctx, lat, lon)
			if err == nil {
				activeProvider = secondaryName
				_ = cache.WriteCache(lat, lon, secondaryName, data)
			} else {
				if *verboseFlag {
					fmt.Fprintf(os.Stderr, "[verbose] Secondary provider %s failed: %v\n", secondaryName, err)
				}

				// Both failed. Check if fallback to local sensor is enabled
				if cfg.Engine.FallbackToSensor {
					if *verboseFlag {
						fmt.Fprintf(os.Stderr, "[verbose] All API requests failed. Fallback to local sensor is active.\n")
					}

					sensorData, ok := cache.LoadSensorData()
					if cachedData != nil {
						// We have an expired cache and a sensor reading: combine them
						data = cache.IntegrateSensor(cachedData, sensorData)
						isHybrid = true
						activeProvider = "local-sensor-hybrid"
						if *verboseFlag {
							fmt.Fprintf(os.Stderr, "[verbose] Combined expired cache data with sensor telemetry. Read success=%t.\n", ok)
						}
					} else {
						// No cache available, build mock from sensor data only
						data = cache.IntegrateSensor(nil, sensorData)
						isHybrid = true
						activeProvider = "local-sensor-pure"
						if *verboseFlag {
							fmt.Fprintf(os.Stderr, "[verbose] No cached data available. Generating minimal telemetry from sensor.\n")
						}
					}
				} else {
					fmt.Fprintf(os.Stderr, "%sError: Weather feeds are offline, and local sensor fallback is disabled.%s\n", colorRed, colorReset)
					os.Exit(1)
				}
			}
		}
	}

	// Render output
	if cfg.Display.Mode == "compact" {
		renderCompact(data, cfg.Engine.Units, activeProvider, cacheHit, isHybrid)
	} else {
		renderHighDensity(data, cfg.Engine.Units, activeProvider, cacheHit, isHybrid, cfg.Display.RenderAsciiCharts)
	}
}

func parseCoords(val string) (float64, float64, error) {
	parts := strings.Split(val, ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("coordinates must be exactly 'latitude,longitude'")
	}
	lat, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid latitude")
	}
	lon, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid longitude")
	}
	return lat, lon, nil
}

// Sparkline renders a high-density vertical block-level sparkline representing pressure trend.
func renderSparkline(values []float64) string {
	if len(values) == 0 {
		return ""
	}
	minVal, maxVal := values[0], values[0]
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}
	span := maxVal - minVal
	if span == 0 {
		span = 1.0
	}

	blocks := []rune{' ', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	var sb strings.Builder
	for _, v := range values {
		ratio := (v - minVal) / span
		idx := int(ratio * 7.0)
		if idx < 0 {
			idx = 0
		}
		if idx > 7 {
			idx = 7
		}
		sb.WriteRune(blocks[idx])
	}
	return sb.String()
}

func formatPressureTendency(currPressure float64, hourly []float64) (string, string) {
	if len(hourly) < 4 {
		return "N/A", "STEADY"
	}
	// Pressure 3 hours ago is len - 4
	p3hAgo := hourly[len(hourly)-4]
	delta, state := meteorology.PressureTendency(currPressure, p3hAgo)

	prefix := ""
	if delta > 0 {
		prefix = "+"
	}

	symbol := "➡️ "
	colorCode := colorGreen
	if state == "RISING" {
		symbol = "📈 "
		colorCode = colorCyan
	} else if state == "FALLING" {
		symbol = "📉 "
		colorCode = colorRed
	}

	return fmt.Sprintf("%s%s%.2f hPa/3h", colorCode, prefix, delta), fmt.Sprintf("%s%s%s%s", colorCode, symbol, state, colorReset)
}

func getCapeClassification(cape float64) (string, string) {
	if cape <= 0 {
		return "Stable", colorGreen
	} else if cape < 1000 {
		return "Marginally Unstable (Weak convection)", colorGreen
	} else if cape < 2500 {
		return "Moderately Unstable (Moderate convective potential)", colorYellow
	} else if cape < 4000 {
		return "Very Unstable (Severe storms possible)", colorRed
	}
	return "Extremely Unstable (Severe storm hazard)", colorPurple
}

func getSolarClassification(w float64) (string, string) {
	if w <= 0 {
		return "Zero flux (Overcast / Night)", colorGray
	} else if w < 250 {
		return "Low irradiance", colorGreen
	} else if w < 600 {
		return "Moderate irradiance", colorYellow
	}
	return "High irradiance", colorRed
}

func renderHighDensity(wd *provider.WeatherData, units string, source string, cacheHit bool, isHybrid bool, drawChart bool) {
	// Status flag
	status := "LIVE"
	if cacheHit {
		status = "CACHED"
	}
	if isHybrid {
		status = "HYBRID-OFFLINE"
	}

	// Calculate dewpoint and LCL
	dp := wd.Dewpoint
	if dp == -99.0 {
		dp = meteorology.Dewpoint(wd.Temperature, wd.RelativeHumidity)
	}
	lclMeters := meteorology.LCLCeiling(wd.Temperature, dp)

	// Unit conversions
	tempStr := fmt.Sprintf("%.1f°C", wd.Temperature)
	dpStr := fmt.Sprintf("%.1f°C", dp)
	lclStr := fmt.Sprintf("%.0fm", lclMeters)
	speedStr := fmt.Sprintf("%.1fm/s", wd.WindSpeed)
	gustStr := fmt.Sprintf("%.1fm/s", wd.WindGusts)
	pressStr := fmt.Sprintf("%.2fhPa", wd.SurfacePressure)
	altimeterStr := fmt.Sprintf("%.2fhPa", wd.SeaLevelPressure)

	if units == "imperial" {
		tempStr = fmt.Sprintf("%.1f°F", meteorology.CtoF(wd.Temperature))
		dpStr = fmt.Sprintf("%.1f°F", meteorology.CtoF(dp))
		lclStr = fmt.Sprintf("%.0fft", meteorology.MetersToFeet(lclMeters))
		speedStr = fmt.Sprintf("%.1fmph", meteorology.MpsToMph(wd.WindSpeed))
		gustStr = fmt.Sprintf("%.1fmph", meteorology.MpsToMph(wd.WindGusts))
		pressStr = fmt.Sprintf("%.2finHg", meteorology.HpaToInHg(wd.SurfacePressure))
		altimeterStr = fmt.Sprintf("%.2finHg", meteorology.HpaToInHg(wd.SeaLevelPressure))
	} else if units == "aviation" {
		tempStr = fmt.Sprintf("%.1f°C", wd.Temperature)
		dpStr = fmt.Sprintf("%.1f°C", dp)
		lclStr = fmt.Sprintf("%.0fft", meteorology.MetersToFeet(lclMeters))
		speedStr = fmt.Sprintf("%.0fkt", meteorology.MpsToKnots(wd.WindSpeed))
		gustStr = fmt.Sprintf("%.0fkt", meteorology.MpsToKnots(wd.WindGusts))
		pressStr = fmt.Sprintf("%.2fhPa", wd.SurfacePressure)
		altimeterStr = fmt.Sprintf("%.2finHg", meteorology.HpaToInHg(wd.SeaLevelPressure))
	}

	// Convective and Solar metadata
	capeDesc, capeColor := getCapeClassification(wd.Cape)
	solarDesc, solarColor := getSolarClassification(wd.SolarIrradiance)

	// Tendency vector
	tendDelta, tendState := formatPressureTendency(wd.SeaLevelPressure, wd.HourlyPressures)

	cardinalDir := meteorology.WindDirectionCardinal(wd.WindDirection)

	// ASCII Panel Outline
	fmt.Printf("%s┌────────────────────────────────────────────────────────┐%s\n", colorBold+colorBlue, colorReset)
	fmt.Printf("%s│ ISOBARIC METEOROLOGICAL ENGINE ──── LOCAL-FIRST HYBRID │%s\n", colorBold+colorBlue, colorReset)
	fmt.Printf("%s└────────────────────────────────────────────────────────┘%s\n", colorBold+colorBlue, colorReset)
	fmt.Printf("%sCoordinates:%s %.4f°N, %.4f°E | %sSource:%s %s (%s)\n", colorBold, colorReset, wd.Latitude, wd.Longitude, colorBold, colorReset, source, status)
	fmt.Printf("%sObservation:%s %s\n\n", colorBold, colorReset, wd.Time.Local().Format("2006-05-02 15:04:05 MST"))

	// SURFACE PANEL
	fmt.Printf("%s[ SURFACE METRICS ] %s%s\n", colorBold+colorCyan, colorGray, strings.Repeat("─", 36)+colorReset)
	fmt.Printf("  Ambient Temp: %-15s Relative Humidity: %.1f%%\n", colorBold+colorGreen+tempStr+colorReset, wd.RelativeHumidity)
	fmt.Printf("  Dewpoint:     %-15s LCL Cond. Ceiling:  %s\n\n", colorBold+colorGreen+dpStr+colorReset, colorBold+colorCyan+lclStr+colorReset)

	// BAROMETRIC PANEL
	fmt.Printf("%s[ BAROMETRIC TELEMETRY ] %s%s\n", colorBold+colorCyan, colorGray, strings.Repeat("─", 32)+colorReset)
	fmt.Printf("  Surface Pressure: %-12s Altimeter Setting: %s\n", pressStr, altimeterStr)
	fmt.Printf("  3h Tendency:      %-12s Vector State:      %s\n", tendDelta, tendState)
	if drawChart && len(wd.HourlyPressures) > 0 {
		// Slice last 24 values to draw sparkline
		sparkVals := wd.HourlyPressures
		if len(sparkVals) > 24 {
			sparkVals = sparkVals[len(sparkVals)-24:]
		}
		sparkline := renderSparkline(sparkVals)
		fmt.Printf("  Baro-Trend (24h):  %s%s%s (%.1f hPa ➔ %.1f hPa)\n", colorCyan, sparkline, colorReset, sparkVals[0], sparkVals[len(sparkVals)-1])
	}
	fmt.Println()

	// WIND PANEL
	fmt.Printf("%s[ WIND & KINETICS ] %s%s\n", colorBold+colorCyan, colorGray, strings.Repeat("─", 37)+colorReset)
	fmt.Printf("  Sustained Velocity: %-10s Cardinal Vector:  %s (%.1f°)\n", speedStr, cardinalDir, wd.WindDirection)
	fmt.Printf("  Maximum Gusts:      %s\n\n", colorBold+colorYellow+gustStr+colorReset)

	// FORECAST PROFILING
	fmt.Printf("%s[ ATMOSPHERIC PROFILING ] %s%s\n", colorBold+colorCyan, colorGray, strings.Repeat("─", 31)+colorReset)
	fmt.Printf("  CAPE Index: %-18s Convective Risk:  %s%s%s\n", fmt.Sprintf("%.0f J/kg", wd.Cape), capeColor, capeDesc, colorReset)
	fmt.Printf("  Solar Flux: %-18s Solar Intensity: %s%s%s\n", fmt.Sprintf("%.0f W/m²", wd.SolarIrradiance), solarColor, solarDesc, colorReset)
	fmt.Printf("%s%s%s\n", colorGray, strings.Repeat("─", 58), colorReset)
}

func renderCompact(wd *provider.WeatherData, units string, source string, cacheHit bool, isHybrid bool) {
	status := "L"
	if cacheHit {
		status = "C"
	}
	if isHybrid {
		status = "H"
	}

	dp := wd.Dewpoint
	if dp == -99.0 {
		dp = meteorology.Dewpoint(wd.Temperature, wd.RelativeHumidity)
	}
	lclMeters := meteorology.LCLCeiling(wd.Temperature, dp)

	tempStr := fmt.Sprintf("%.1fC", wd.Temperature)
	dpStr := fmt.Sprintf("%.1fC", dp)
	lclStr := fmt.Sprintf("%.0fm", lclMeters)
	speedStr := fmt.Sprintf("%.1fm/s", wd.WindSpeed)
	gustStr := fmt.Sprintf("%.1fm/s", wd.WindGusts)
	altimeterStr := fmt.Sprintf("%.1fhPa", wd.SeaLevelPressure)

	if units == "imperial" {
		tempStr = fmt.Sprintf("%.1fF", meteorology.CtoF(wd.Temperature))
		dpStr = fmt.Sprintf("%.1fF", meteorology.CtoF(dp))
		lclStr = fmt.Sprintf("%.0fft", meteorology.MetersToFeet(lclMeters))
		speedStr = fmt.Sprintf("%.1fmph", meteorology.MpsToMph(wd.WindSpeed))
		gustStr = fmt.Sprintf("%.1fmph", meteorology.MpsToMph(wd.WindGusts))
		altimeterStr = fmt.Sprintf("%.2fin", meteorology.HpaToInHg(wd.SeaLevelPressure))
	} else if units == "aviation" {
		tempStr = fmt.Sprintf("%.1fC", wd.Temperature)
		dpStr = fmt.Sprintf("%.1fC", dp)
		lclStr = fmt.Sprintf("%.0fft", meteorology.MetersToFeet(lclMeters))
		speedStr = fmt.Sprintf("%.0fkt", meteorology.MpsToKnots(wd.WindSpeed))
		gustStr = fmt.Sprintf("%.0fkt", meteorology.MpsToKnots(wd.WindGusts))
		altimeterStr = fmt.Sprintf("%.2fin", meteorology.HpaToInHg(wd.SeaLevelPressure))
	}

	var p3hAgo float64
	deltaStr := "N/A"
	tendState := "STEADY"
	if len(wd.HourlyPressures) >= 4 {
		p3hAgo = wd.HourlyPressures[len(wd.HourlyPressures)-4]
		delta, state := meteorology.PressureTendency(wd.SeaLevelPressure, p3hAgo)
		tendState = state
		sign := ""
		if delta > 0 {
			sign = "+"
		}
		deltaStr = fmt.Sprintf("%s%.1f", sign, delta)
	}

	cardinalDir := meteorology.WindDirectionCardinal(wd.WindDirection)

	sparkline := ""
	if len(wd.HourlyPressures) > 0 {
		sparkVals := wd.HourlyPressures
		if len(sparkVals) > 12 {
			sparkVals = sparkVals[len(sparkVals)-12:]
		}
		sparkline = renderSparkline(sparkVals)
	}

	// 5-line matrix output with absolutely zero decorations
	fmt.Printf("ISO:%.2f,%.2f|SRC:%s|S:%s|T:%s\n", wd.Latitude, wd.Longitude, source, status, wd.Time.Format("01-02 15:04"))
	fmt.Printf("MET:T:%s RH:%.0f%% DP:%s LCL:%s\n", tempStr, wd.RelativeHumidity, dpStr, lclStr)
	fmt.Printf("BARO:P:%s D:%s (%s) %s\n", altimeterStr, deltaStr, tendState, sparkline)
	fmt.Printf("WIND:W:%s G:%s D:%s(%.0f)\n", speedStr, gustStr, cardinalDir, wd.WindDirection)
	fmt.Printf("CAPE:%.0fJ SOL:%.0fW\n", wd.Cape, wd.SolarIrradiance)
}
