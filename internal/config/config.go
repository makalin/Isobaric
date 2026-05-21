package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Engine    EngineConfig
	Providers ProvidersConfig
	Display   DisplayConfig
}

type EngineConfig struct {
	Units            string // "metric", "imperial", "aviation"
	CacheTTLMinutes  int
	FallbackToSensor bool
}

type ProvidersConfig struct {
	Primary   string
	Secondary string
}

type DisplayConfig struct {
	Mode              string // "high_density", "compact"
	RenderAsciiCharts bool
}

// DefaultConfig returns default configuration parameters.
func DefaultConfig() *Config {
	return &Config{
		Engine: EngineConfig{
			Units:            "metric",
			CacheTTLMinutes:  15,
			FallbackToSensor: true,
		},
		Providers: ProvidersConfig{
			Primary:   "noaa",
			Secondary: "open-meteo",
		},
		Display: DisplayConfig{
			Mode:              "high_density",
			RenderAsciiCharts: true,
		},
	}
}

// LoadConfig attempts to load the configuration from ~/.config/isobaric/config.toml.
// If the file does not exist, it returns the default configuration.
func LoadConfig(customPath string) (*Config, error) {
	configPath := customPath
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return DefaultConfig(), fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = filepath.Join(home, ".config", "isobaric", "config.toml")
	}

	file, err := os.Open(configPath)
	if os.IsNotExist(err) {
		// Return default silently
		return DefaultConfig(), nil
	} else if err != nil {
		return DefaultConfig(), fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	cfg := DefaultConfig()
	scanner := bufio.NewScanner(file)
	currentSection := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Handle sections: [section]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.ToLower(line[1 : len(line)-1])
			continue
		}

		// Handle key-value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		// Strip trailing comments from value if present
		if idx := strings.Index(val, "#"); idx != -1 {
			val = strings.TrimSpace(val[:idx])
		}
		if idx := strings.Index(val, ";"); idx != -1 {
			val = strings.TrimSpace(val[:idx])
		}

		// Unquote string values
		if strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"") {
			val = val[1 : len(val)-1]
		} else if strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'") {
			val = val[1 : len(val)-1]
		}

		switch currentSection {
		case "engine":
			switch key {
			case "units":
				cfg.Engine.Units = val
			case "cache_ttl_minutes":
				if i, err := strconv.Atoi(val); err == nil {
					cfg.Engine.CacheTTLMinutes = i
				}
			case "fallback_to_sensor":
				if b, err := strconv.ParseBool(val); err == nil {
					cfg.Engine.FallbackToSensor = b
				}
			}
		case "providers":
			switch key {
			case "primary":
				cfg.Providers.Primary = val
			case "secondary":
				cfg.Providers.Secondary = val
			}
		case "display":
			switch key {
			case "mode":
				cfg.Display.Mode = val
			case "render_ascii_charts":
				if b, err := strconv.ParseBool(val); err == nil {
					cfg.Display.RenderAsciiCharts = b
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return cfg, fmt.Errorf("error reading config file: %w", err)
	}

	return cfg, nil
}
