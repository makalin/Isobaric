import SwiftUI

// Struct to hold meteorological state
struct WeatherState {
    var lat: Double = 40.99
    var lon: Double = 29.41
    var provider: String = "open-meteo"
    var isCached: Bool = false
    var isHybrid: Bool = false
    var observationTime: Date = Date()
    
    // Core parameters
    var tempC: Double = 16.0
    var humidity: Double = 92.0
    var dewpointC: Double = 14.7
    var pressureHpa: Double = 997.0
    var altimeterHpa: Double = 1011.9
    var windSpeedMps: Double = 1.6
    var windDirection: Double = 315.0
    var windGustsMps: Double = 5.3
    var cape: Double = 40.0
    var solarFlux: Double = 87.0
    
    // Historical trends (24 hours of pressure)
    var hourlyPressures: [Double] = [
        1009.8, 1010.1, 1009.9, 1010.0, 1009.7, 1010.2, 1009.9, 1010.3,
        1010.1, 1010.8, 1011.0, 1011.3, 1011.7, 1011.5, 1011.6, 1011.0,
        1011.1, 1010.6, 1010.7, 1010.2, 1010.6, 1011.0, 1011.6, 1011.9
    ]
}

struct ContentView: View {
    @State private var weather = WeatherState()
    @State private var useImperial = false
    @State private var simulationMode = false
    @State private var selectedLayout: AppLayoutMode = .auto
    
    enum AppLayoutMode {
        case auto, tallSplit, compactSquare
    }
    
    // Lifting Condensation Level Calculation
    var lclMeters: Double {
        if weather.dewpointC >= weather.tempC { return 0.0 }
        return 125.0 * (weather.tempC - weather.dewpointC)
    }
    
    // Wind Cardinal Conversion
    var windCardinal: String {
        let deg = weather.windDirection.truncatingRemainder(dividingBy: 360)
        let cardinals = ["N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"]
        let idx = Int(floor((deg + 11.25) / 22.5)) % 16
        return cardinals[idx]
    }
    
    // 3h pressure tendency calculation
    var pressureTendency: (delta: Double, state: String) {
        if weather.hourlyPressures.count < 4 { return (0.0, "STEADY") }
        let current = weather.altimeterHpa
        let past3h = weather.hourlyPressures[weather.hourlyPressures.count - 4]
        let diff = current - past3h
        let state: String
        if diff > 1.5 {
            state = "RISING"
        } else if diff < -1.5 {
            state = "FALLING"
        } else {
            state = "STEADY"
        }
        return (diff, state)
    }
    
    var body: some View {
        GeometryReader { geo in
            let isSquare = min(geo.size.width, geo.size.height) / max(geo.size.width, geo.size.height) > 0.8
            let layoutToRender = resolveLayout(isSquare: isSquare)
            
            ZStack {
                // Background dark weather-themed gradient
                LinearGradient(
                    gradient: Gradient(colors: [Color(red: 0.08, green: 0.11, blue: 0.18), Color(red: 0.03, green: 0.04, blue: 0.06)]),
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )
                .ignoresSafeArea()
                
                VStack(spacing: 0) {
                    // Header Toolbar (Only for testing / setup, hidden in pure dashboard mode)
                    HStack {
                        Text("isobaric")
                            .font(.system(.headline, design: .monospaced))
                            .fontWeight(.bold)
                            .foregroundColor(.cyan)
                        
                        Spacer()
                        
                        Picker("Layout", selection: $selectedLayout) {
                            Text("Auto").tag(AppLayoutMode.auto)
                            Text("Split").tag(AppLayoutMode.tallSplit)
                            Text("Compact").tag(AppLayoutMode.compactSquare)
                        }
                        .pickerStyle(SegmentedPickerStyle())
                        .frame(width: 180)
                        
                        Button(action: { useImperial.toggle() }) {
                            Text(useImperial ? "US" : "Metric")
                                .font(.caption)
                                .padding(.horizontal, 8)
                                .padding(.vertical, 4)
                                .background(Color.cyan.opacity(0.2))
                                .cornerRadius(4)
                                .foregroundColor(.cyan)
                        }
                    }
                    .padding(.horizontal)
                    .padding(.vertical, 8)
                    .background(Color.black.opacity(0.3))
                    
                    if layoutToRender == .compactSquare {
                        CompactSquareLayoutView(weather: weather, useImperial: useImperial, lcl: lclMeters, windDir: windCardinal, tendency: pressureTendency)
                    } else {
                        TallSplitLayoutView(weather: weather, useImperial: useImperial, lcl: lclMeters, windDir: windCardinal, tendency: pressureTendency)
                    }
                }
            }
        }
    }
    
    private func resolveLayout(isSquare: Bool) -> AppLayoutMode {
        switch selectedLayout {
        case .tallSplit: return .tallSplit
        case .compactSquare: return .compactSquare
        case .auto:
            return isSquare ? .compactSquare : .tallSplit
        }
    }
}

// MARK: - TALL SPLIT LAYOUT
struct TallSplitLayoutView: View {
    let weather: WeatherState
    let useImperial: Bool
    let lcl: Double
    let windDir: String
    let tendency: (delta: Double, state: String)
    
    var body: some View {
        VStack(spacing: 0) {
            // Upper segment: Graphical planetary charts & live sparkline
            VStack(spacing: 12) {
                HStack {
                    VStack(alignment: .leading) {
                        Text("OBSERVATION TELEMETRY")
                            .font(.system(.caption, design: .monospaced))
                            .foregroundColor(.gray)
                        Text("\(String(format: "%.4f", weather.lat))°N, \(String(format: "%.4f", weather.lon))°E")
                            .font(.system(.subheadline, design: .monospaced))
                            .fontWeight(.medium)
                            .foregroundColor(.white)
                    }
                    Spacer()
                    Text("LIVE")
                        .font(.system(.caption, design: .monospaced))
                        .fontWeight(.bold)
                        .padding(.horizontal, 8)
                        .padding(.vertical, 2)
                        .background(Color.green.opacity(0.2))
                        .cornerRadius(4)
                        .foregroundColor(.green)
                }
                .padding(.horizontal)
                
                // Sparkline Microchart for Barometric Pressure
                VStack(alignment: .leading, spacing: 4) {
                    Text("BAROMETRIC TENDENCY (24H)")
                        .font(.system(.caption2, design: .monospaced))
                        .foregroundColor(.cyan)
                        .padding(.horizontal)
                    
                    SparklineView(values: weather.hourlyPressures)
                        .frame(height: 80)
                        .padding(.horizontal)
                    
                    HStack {
                        Text("\(String(format: "%.1f hPa", weather.hourlyPressures.first ?? 0))")
                        Spacer()
                        Text("\(String(format: "%.1f hPa", weather.hourlyPressures.last ?? 0))")
                    }
                    .font(.system(.caption2, design: .monospaced))
                    .foregroundColor(.gray)
                    .padding(.horizontal)
                }
                .padding(.vertical, 8)
                .background(Color.white.opacity(0.03))
                .cornerRadius(8)
                .padding(.horizontal)
            }
            .padding(.top)
            .frame(maxHeight: .infinity)
            
            Divider()
                .background(Color.cyan.opacity(0.3))
            
            // Lower segment: Deep technical grid stack
            ScrollView {
                VStack(spacing: 16) {
                    // Surface Metrics
                    MetricSectionView(title: "SURFACE PARAMETERS") {
                        MetricRowView(label: "Temperature", val: formatTemp(weather.tempC))
                        MetricRowView(label: "Humidity", val: String(format: "%.1f%%", weather.humidity))
                        MetricRowView(label: "Dewpoint", val: formatTemp(weather.dewpointC))
                        MetricRowView(label: "LCL Ceiling", val: formatDistance(lcl))
                    }
                    
                    // Barometer Details
                    MetricSectionView(title: "BAROMETRIC TELEMETRY") {
                        MetricRowView(label: "Sea-Level Baro", val: formatPressure(weather.altimeterHpa))
                        MetricRowView(label: "Station Pressure", val: formatPressure(weather.pressureHpa))
                        MetricRowView(label: "3h Tendency", val: String(format: "%+.1f hPa", tendency.delta))
                        MetricRowView(label: "Trend State", val: tendency.state)
                    }
                    
                    // Wind & Kinetics
                    MetricSectionView(title: "WIND & KINETICS") {
                        MetricRowView(label: "Sustained Speed", val: formatSpeed(weather.windSpeedMps))
                        MetricRowView(label: "Max Gusts", val: formatSpeed(weather.windGustsMps))
                        MetricRowView(label: "Directional Vector", val: "\(windDir) (\(Int(weather.windDirection))°)")
                    }
                    
                    // Convective stability
                    MetricSectionView(title: "CONVECTIVE PROFILING") {
                        MetricRowView(label: "CAPE Index", val: String(format: "%.0f J/kg", weather.cape))
                        MetricRowView(label: "Solar Irradiance", val: String(format: "%.0f W/m²", weather.solarFlux))
                    }
                }
                .padding()
            }
            .frame(maxHeight: .infinity)
        }
    }
    
    // Helpers
    private func formatTemp(_ c: Double) -> String {
        if useImperial { return String(format: "%.1f°F", c * 1.8 + 32.0) }
        return String(format: "%.1f°C", c)
    }
    
    private func formatDistance(_ m: Double) -> String {
        if useImperial { return String(format: "%.0fft", m * 3.28084) }
        return String(format: "%.0fm", m)
    }
    
    private func formatPressure(_ hpa: Double) -> String {
        if useImperial { return String(format: "%.2finHg", hpa * 0.02953) }
        return String(format: "%.1fhPa", hpa)
    }
    
    private func formatSpeed(_ mps: Double) -> String {
        if useImperial { return String(format: "%.1fmph", mps * 2.23694) }
        return String(format: "%.1fm/s", mps)
    }
}

// MARK: - COMPACT SQUARE MATRIX LAYOUT
struct CompactSquareLayoutView: View {
    let weather: WeatherState
    let useImperial: Bool
    let lcl: Double
    let windDir: String
    let tendency: (delta: Double, state: String)
    
    var body: some View {
        VStack(spacing: 4) {
            // Header line
            HStack {
                Text("ISO:\(String(format: "%.2f", weather.lat)),\(String(format: "%.2f", weather.lon))")
                Spacer()
                Text("SRC:\(weather.provider.prefix(4).uppercased())")
                Spacer()
                Text("LIVE")
                    .foregroundColor(.green)
            }
            .font(.system(size: 10, weight: .bold, design: .monospaced))
            .foregroundColor(.gray)
            .padding(.horizontal, 4)
            
            Divider()
                .background(Color.gray.opacity(0.3))
            
            // Grid layout 4x2 matrix
            VStack(spacing: 2) {
                // Row 1
                HStack(spacing: 2) {
                    CompactCell(title: "TEMP", val: formatTemp(weather.tempC), color: .green)
                    CompactCell(title: "HUMID", val: String(format: "%.0f%%", weather.humidity), color: .white)
                    CompactCell(title: "DEWPT", val: formatTemp(weather.dewpointC), color: .green)
                }
                
                // Row 2
                HStack(spacing: 2) {
                    CompactCell(title: "BARO", val: formatPressure(weather.altimeterHpa), color: .cyan)
                    CompactCell(title: "3H DELTA", val: String(format: "%+.1f", tendency.delta), color: tendency.delta >= 0 ? .cyan : .red)
                    CompactCell(title: "LCL HGHT", val: formatDistance(lcl), color: .cyan)
                }
                
                // Row 3
                HStack(spacing: 2) {
                    CompactCell(title: "WIND", val: formatSpeed(weather.windSpeedMps), color: .yellow)
                    CompactCell(title: "GUST", val: formatSpeed(weather.windGustsMps), color: .yellow)
                    CompactCell(title: "VECTOR", val: "\(windDir) \(Int(weather.windDirection))°", color: .yellow)
                }
                
                // Row 4
                HStack(spacing: 2) {
                    CompactCell(title: "CAPE", val: String(format: "%.0fJ", weather.cape), color: weather.cape > 100 ? .red : .green)
                    CompactCell(title: "SOLAR", val: String(format: "%.0fW", weather.solarFlux), color: .orange)
                    CompactCell(title: "TREND", val: tendency.state, color: .purple)
                }
            }
            
            // Bottom small trend visualizer
            HStack(spacing: 1) {
                ForEach(weather.hourlyPressures, id: \.self) { pressure in
                    Rectangle()
                        .fill(Color.cyan)
                        .frame(height: CGFloat(normalize(pressure) * 12.0 + 2.0))
                }
            }
            .frame(height: 16)
            .padding(.top, 2)
        }
        .padding(4)
        .background(Color.black.opacity(0.8))
        .cornerRadius(6)
        .padding(4)
    }
    
    // Normalization helper
    private func normalize(_ val: Double) -> Double {
        guard let maxVal = weather.hourlyPressures.max(),
              let minVal = weather.hourlyPressures.min() else { return 0.5 }
        let diff = maxVal - minVal
        if diff == 0 { return 0.5 }
        return (val - minVal) / diff
    }
    
    // Helpers
    private func formatTemp(_ c: Double) -> String {
        if useImperial { return String(format: "%.0f°F", c * 1.8 + 32.0) }
        return String(format: "%.1f°C", c)
    }
    
    private func formatDistance(_ m: Double) -> String {
        if useImperial { return String(format: "%.0fft", m * 3.28084) }
        return String(format: "%.0fm", m)
    }
    
    private func formatPressure(_ hpa: Double) -> String {
        if useImperial { return String(format: "%.2fin", hpa * 0.02953) }
        return String(format: "%.1fhP", hpa)
    }
    
    private func formatSpeed(_ mps: Double) -> String {
        if useImperial { return String(format: "%.0fmph", mps * 2.23694) }
        return String(format: "%.1fm/s", mps)
    }
}

// MARK: - COMPACT CELL
struct CompactCell: View {
    let title: String
    let val: String
    let color: Color
    
    var body: some View {
        VStack(alignment: .leading, spacing: 1) {
            Text(title)
                .font(.system(size: 7, weight: .semibold, design: .monospaced))
                .foregroundColor(.gray)
            Text(val)
                .font(.system(size: 11, weight: .bold, design: .monospaced))
                .foregroundColor(color)
                .lineLimit(1)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(4)
        .background(Color.white.opacity(0.04))
        .cornerRadius(3)
    }
}

// MARK: - SUBVIEWS & RENDERERS
struct MetricSectionView<Content: View>: View {
    let title: String
    let content: Content
    
    init(title: String, @ViewBuilder content: () -> Content) {
        self.title = title
        self.content = content()
    }
    
    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(title)
                .font(.system(.caption, design: .monospaced))
                .fontWeight(.bold)
                .foregroundColor(.cyan)
            
            VStack(spacing: 6) {
                content
            }
            .padding(10)
            .background(Color.white.opacity(0.04))
            .cornerRadius(8)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }
}

struct MetricRowView: View {
    let label: String
    let val: String
    
    var body: some View {
        HStack {
            Text(label)
                .font(.system(.subheadline, design: .monospaced))
                .foregroundColor(.gray)
            Spacer()
            Text(val)
                .font(.system(.subheadline, design: .monospaced))
                .fontWeight(.bold)
                .foregroundColor(.white)
        }
    }
}

// Draw line path sparkline using canvas
struct SparklineView: View {
    let values: [Double]
    
    var body: some View {
        Canvas { context, size in
            guard values.count > 1 else { return }
            let maxVal = values.max() ?? 1.0
            let minVal = values.min() ?? 0.0
            let range = maxVal - minVal == 0 ? 1.0 : maxVal - minVal
            
            let width = size.width
            let height = size.height
            let stepX = width / CGFloat(values.count - 1)
            
            var path = Path()
            for idx in 0..<values.count {
                let x = CGFloat(idx) * stepX
                let normalizedVal = (values[idx] - minVal) / range
                let y = height - (CGFloat(normalizedVal) * height)
                
                if idx == 0 {
                    path.move(to: CGPoint(x: x, y: y))
                } else {
                    path.addLine(to: CGPoint(x: x, y: y))
                }
            }
            
            context.stroke(path, with: .color(.cyan), lineWidth: 2)
            
            // Draw gradient area under the path
            var fillPath = path
            fillPath.addLine(to: CGPoint(x: width, y: height))
            fillPath.addLine(to: CGPoint(x: 0, y: height))
            fillPath.closeSubpath()
            
            context.fill(fillPath, with: .linearGradient(
                Gradient(colors: [Color.cyan.opacity(0.3), Color.cyan.opacity(0.0)]),
                startPoint: CGPoint(x: 0, y: 0),
                endPoint: CGPoint(x: 0, y: height)
            ))
        }
    }
}
