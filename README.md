# Isobaric

`isobaric` is a zero-bloat, local-first, highly detailed professional weather engine designed for developers, outdoor enthusiasts, and hardware pragmatists. It rejects the slow, ad-ridden, subscription-based models of modern weather apps in favor of raw atmospheric telemetry, extreme data density, and adaptive multi-surface layouts.

Whether running on an ultra-tall folding display, a compact square QWERTY screen, or directly inside a terminal, `isobaric` delivers pure, unadulterated meteorological data without the corporate fluff.

## Architecture & Philosophy

* **Zero Bloat:** No modern tracking SDKs, no animations that crush battery life, and absolutely no premium paywalls.
* **Engineered for Form Factors:** Native grid layouts that adapt seamlessly to unique mobile displays—from long folding panels to tight, square physical keyboard interfaces.
* **Geophysical Depth:** Moves beyond basic "rain/sunny" icons. Includes barometric pressure tendencies, solar radiation metrics, CAPE index (Convective Available Potential Energy) for storm severity, dewpoint spreads, and wind shear vectors.
* **Local-First & Offline Capable:** Caches raw JSON payloads locally. If you lose connection on a trail or at sea, it computes short-term trend deltas using your device's built-in barometric sensor.

---

## Technical Stack

The core engine is built to be blisteringly fast, light on memory, and completely independent of bloated web frameworks.

* **Core Logic:** Go / Rust (compiled to native binaries or highly optimized WebAssembly cores).
* **UI Layer:** Vanilla, performance-tuned layouts utilizing standard platform-native rendering pipelines (zero reliance on heavy runtime engines or memory-hogging web views).
* **Data Sources:** Multi-source fallback pipeline querying public, highly accurate meteorological nodes including NOAA (National Oceanic and Atmospheric Administration), ECMWF (European Centre for Medium-Range Weather Forecasts), and open telemetry APIs.

---

## Feature Roadmap

### 1. The Micro-Dashboard

A highly scannable grid showing critical surface conditions instantly.

* True Barometric Pressure + 3-hour tendency vector (Rising/Steady/Falling).
* Relative Humidity vs. Ambient Dewpoint (calculating the precise condensation ceiling).
* Wind Vector mapping (Sustained velocity + maximum gust spectrum).

### 2. Atmospheric Profiling

Detailed vertical telemetry for advanced forecasting.

| Metric | Resolution | Purpose |
| --- | --- | --- |
| **CAPE Index** | $J/kg$ | Evaluates convective instability and localized thunderstorm potential. |
| **Solar Irradiance** | $W/m^2$ | Real-time solar energy flux reaching the surface. |
| **Altimeter Setting** | $inHg / hPa$ | Raw pressure data corrected to sea level for precise trend tracking. |

### 3. Responsive Adaptations

* **Tall Split Display:** Utilizes upper and lower viewport segments on folding screens to stack deep planetary data visualizations above live-updating microcharts.
* **Compact Square Matrix:** Aggresses padding and omits decorative graphics entirely, fitting a comprehensive 48-hour data matrix onto tight displays without scrolling.

---

## Local Development & Configuration

The configuration is handled via a single flat, predictable file.

```toml
# ~/.config/isobaric/config.toml

[engine]
units = "metric" # metric, imperial, or aviation (hPa/knots)
cache_ttl_minutes = 15
fallback_to_sensor = true

[providers]
primary = "noaa"
secondary = "open-meteo"

[display]
mode = "high_density"
render_ascii_charts = true

```

### Quick Start (Engine Compilation)

To build the native binary core locally:

```bash
# Clone the repository
git clone https://github.com/makalin/isobaric.git
cd isobaric

# Build the high-performance release binary
make build-release

# Run the local telemetry test tool
./bin/isobaric --coords=40.99,29.41 --verbose

```

---

## License

MIT License — Copyright (c) 2026 Mehmet T. AKALIN

See [LICENSE](LICENSE) for the full text.

---

> **Meteorological Note:** By tracking the raw pressure trend delta ($\Delta P / \Delta t$) locally via native device sensors, `isobaric` can warn you of approaching convective fronts and sudden microbursts up to 45 minutes before public consumer APIs update their servers.