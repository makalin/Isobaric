package com.isobaric.app

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlin.math.floor

// Meteorological State
data class WeatherState(
    val lat: Double = 40.99,
    val lon: Double = 29.41,
    val provider: String = "open-meteo",
    val isCached: Boolean = false,
    val isHybrid: Boolean = false,
    val tempC: Double = 16.0,
    val humidity: Double = 92.0,
    val dewpointC: Double = 14.7,
    val pressureHpa: Double = 997.0,
    val altimeterHpa: Double = 1011.9,
    val windSpeedMps: Double = 1.6,
    val windDirection: Double = 315.0,
    val windGustsMps: Double = 5.3,
    val cape: Double = 40.0,
    val solarFlux: Double = 87.0,
    val hourlyPressures: List<Double> = listOf(
        1009.8, 1010.1, 1009.9, 1010.0, 1009.7, 1010.2, 1009.9, 1010.3,
        1010.1, 1010.8, 1011.0, 1011.3, 1011.7, 1011.5, 1011.6, 1011.0,
        1011.1, 1010.6, 1010.7, 1010.2, 1010.6, 1011.0, 1011.6, 1011.9
    )
)

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState: Bundle?)
        setContent {
            MaterialTheme(
                colorScheme = darkColorScheme(
                    primary = Color(0xFF00E5FF), // Cyan
                    background = Color(0xFF141A29),
                    surface = Color(0xFF1E273A)
                )
            ) {
                Surface(
                    modifier = Modifier.fillMaxSize(),
                    color = MaterialTheme.colorScheme.background
                ) {
                    IsobaricScreen()
                }
            }
        }
    }
}

@Composable
fun IsobaricScreen() {
    val weather by remember { mutableStateOf(WeatherState()) }
    var useImperial by remember { mutableStateOf(false) }
    var selectedLayoutMode by remember { mutableStateOf("auto") }

    val configuration = LocalConfiguration.current
    val screenWidth = configuration.screenWidthDp
    val screenHeight = configuration.screenHeightDp
    val aspectRatio = screenWidth.toFloat() / screenHeight.toFloat()
    
    // Auto-detect square displays (aspect ratios between 0.8 and 1.25)
    val isSquareScreen = aspectRatio in 0.8f..1.25f
    val currentLayout = when (selectedLayoutMode) {
        "tallSplit" -> "tallSplit"
        "compactSquare" -> "compactSquare"
        else -> if (isSquareScreen) "compactSquare" else "tallSplit"
    }

    // Meteorological Conversions
    val lclMeters = if (weather.dewpointC >= weather.tempC) 0.0 else 125.0 * (weather.tempC - weather.dewpointC)
    val windDirCardinal = remember(weather.windDirection) {
        val deg = weather.windDirection % 360
        val cardinals = listOf("N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW")
        val idx = floor((deg + 11.25) / 22.5).toInt() % 16
        cardinals[idx]
    }
    
    val pressureTendency = remember(weather.altimeterHpa, weather.hourlyPressures) {
        if (weather.hourlyPressures.size < 4) Pair(0.0, "STEADY")
        else {
            val delta = weather.altimeterHpa - weather.hourlyPressures[weather.hourlyPressures.size - 4]
            val state = when {
                delta > 1.5 -> "RISING"
                delta < -1.5 -> "FALLING"
                else -> "STEADY"
            }
            Pair(delta, state)
        }
    }

    VStackBackground {
        // App Header
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .background(Color.Black.copy(alpha = 0.3f))
                .padding(horizontal = 16.dp, vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.SpaceBetween
        ) {
            Text(
                text = "isobaric",
                fontFamily = FontFamily.Monospace,
                fontWeight = FontWeight.Bold,
                fontSize = 16.sp,
                color = Color(0xFF00E5FF)
            )

            // Manual Mode Selector
            Row(verticalAlignment = Alignment.CenterVertically) {
                Button(
                    onClick = {
                        selectedLayoutMode = when (selectedLayoutMode) {
                            "auto" -> "tallSplit"
                            "tallSplit" -> "compactSquare"
                            else -> "auto"
                        }
                    },
                    modifier = Modifier.padding(end = 8.dp),
                    colors = ButtonDefaults.buttonColors(containerColor = Color.Cyan.copy(alpha = 0.2f))
                ) {
                    Text(
                        text = "Layout: ${selectedLayoutMode.replaceFirstChar { it.uppercase() }}",
                        color = Color(0xFF00E5FF),
                        fontSize = 10.sp
                    )
                }

                Button(
                    onClick = { useImperial = !useImperial },
                    colors = ButtonDefaults.buttonColors(containerColor = Color.Cyan.copy(alpha = 0.2f))
                ) {
                    Text(
                        text = if (useImperial) "US" else "Metric",
                        color = Color(0xFF00E5FF),
                        fontSize = 10.sp
                    )
                }
            }
        }

        if (currentLayout == "compactSquare") {
            CompactSquareLayout(weather, useImperial, lclMeters, windDirCardinal, pressureTendency)
        } else {
            TallSplitLayout(weather, useImperial, lclMeters, windDirCardinal, pressureTendency)
        }
    }
}

// MARK: - WRAPPERS & UTILITIES
@Composable
fun VStackBackground(content: @Composable ColumnScope.() -> Unit) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(
                Brush.verticalGradient(
                    colors = listOf(Color(0xFF141B2D), Color(0xFF070B12))
                )
            ),
        content = content
    )
}

// MARK: - TALL SPLIT LAYOUT
@Composable
fun TallSplitLayout(
    weather: WeatherState,
    useImperial: Boolean,
    lcl: Double,
    windDir: String,
    tendency: Pair<Double, String>
) {
    Column(modifier = Modifier.fillMaxSize()) {
        // Upper Segment: Sparkline and observation location
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .weight(1f)
                .padding(16.dp),
            verticalArrangement = Arrangement.Center
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Column {
                    Text("OBSERVATION TELEMETRY", fontSize = 10.sp, fontFamily = FontFamily.Monospace, color = Color.Gray)
                    Text("${weather.lat}°N, ${weather.lon}°E", fontSize = 14.sp, fontFamily = FontFamily.Monospace, fontWeight = FontWeight.Medium, color = Color.White)
                }
                Text(
                    "LIVE",
                    fontSize = 10.sp,
                    fontFamily = FontFamily.Monospace,
                    fontWeight = FontWeight.Bold,
                    color = Color.Green,
                    modifier = Modifier
                        .background(Color.Green.copy(alpha = 0.15f), RoundedCornerShape(4.dp))
                        .padding(horizontal = 8.dp, vertical = 2.dp)
                )
            }

            Spacer(modifier = Modifier.height(16.dp))

            // Sparkline Panel
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .background(Color.White.copy(alpha = 0.03f), RoundedCornerShape(8.dp))
                    .padding(12.dp)
            ) {
                Text("BAROMETRIC TENDENCY (24H)", fontSize = 10.sp, fontFamily = FontFamily.Monospace, color = Color(0xFF00E5FF))
                Spacer(modifier = Modifier.height(8.dp))
                Sparkline(
                    values = weather.hourlyPressures,
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(80.dp)
                )
                Spacer(modifier = Modifier.height(6.dp))
                Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                    Text("${weather.hourlyPressures.first()} hPa", fontSize = 10.sp, fontFamily = FontFamily.Monospace, color = Color.Gray)
                    Text("${weather.hourlyPressures.last()} hPa", fontSize = 10.sp, fontFamily = FontFamily.Monospace, color = Color.Gray)
                }
            }
        }

        Divider(color = Color.Cyan.copy(alpha = 0.2f))

        // Lower Segment: Grid of details
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .weight(1.2f)
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            MetricSection("SURFACE METRICS") {
                MetricRow("Ambient Temp", formatTemp(weather.tempC, useImperial))
                MetricRow("Relative Humidity", "${weather.humidity}%")
                MetricRow("Dewpoint Spread", formatTemp(weather.dewpointC, useImperial))
                MetricRow("Condensation Ceiling", formatDistance(lcl, useImperial))
            }

            MetricSection("BAROMETRIC TELEMETRY") {
                MetricRow("Altimeter Setting", formatPressure(weather.altimeterHpa, useImperial))
                MetricRow("Station Pressure", formatPressure(weather.pressureHpa, useImperial))
                MetricRow("3h Trend Delta", String.format("%+.1f hPa", tendency.first))
                MetricRow("Pressure Vector", tendency.second)
            }

            MetricSection("WIND & KINETICS") {
                MetricRow("Sustained Velocity", formatSpeed(weather.windSpeedMps, useImperial))
                MetricRow("Maximum Gusts", formatSpeed(weather.windGustsMps, useImperial))
                MetricRow("Vector Direction", "$windDir (${weather.windDirection.toInt()}°)")
            }

            MetricSection("CONVECTIVE INSTABILITY") {
                MetricRow("CAPE Index", String.format("%.0f J/kg", weather.cape))
                MetricRow("Solar Irradiance", String.format("%.0f W/m²", weather.solarFlux))
            }
        }
    }
}

// MARK: - COMPACT SQUARE LAYOUT
@Composable
fun CompactSquareLayout(
    weather: WeatherState,
    useImperial: Boolean,
    lcl: Double,
    windDir: String,
    tendency: Pair<Double, String>
) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(4.dp),
        verticalArrangement = Arrangement.spacedBy(4.dp)
    ) {
        // Location row
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 4.dp),
            horizontalArrangement = Arrangement.SpaceBetween
        ) {
            Text("ISO:${weather.lat},${weather.lon}", fontSize = 10.sp, fontFamily = FontFamily.Monospace, color = Color.LightGray)
            Text("SRC:${weather.provider.uppercase()}", fontSize = 10.sp, fontFamily = FontFamily.Monospace, color = Color.LightGray)
            Text("HYBRID", fontSize = 10.sp, fontFamily = FontFamily.Monospace, color = Color.Green)
        }

        Divider(color = Color.Gray.copy(alpha = 0.3f))

        // Grid Rows
        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(2.dp)) {
            CompactCell("TEMP", formatTemp(weather.tempC, useImperial), Color.Green, Modifier.weight(1f))
            CompactCell("HUMID", "${weather.humidity.toInt()}%", Color.White, Modifier.weight(1f))
            CompactCell("DEWPT", formatTemp(weather.dewpointC, useImperial), Color.Green, Modifier.weight(1f))
        }

        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(2.dp)) {
            CompactCell("BARO", formatPressure(weather.altimeterHpa, useImperial), Color(0xFF00E5FF), Modifier.weight(1f))
            CompactCell("3H CHNG", String.format("%+.1f", tendency.first), if (tendency.first >= 0) Color(0xFF00E5FF) else Color.Red, Modifier.weight(1f))
            CompactCell("LCL CEIL", formatDistance(lcl, useImperial), Color(0xFF00E5FF), Modifier.weight(1f))
        }

        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(2.dp)) {
            CompactCell("WIND", formatSpeed(weather.windSpeedMps, useImperial), Color.Yellow, Modifier.weight(1f))
            CompactCell("GUST", formatSpeed(weather.windGustsMps, useImperial), Color.Yellow, Modifier.weight(1f))
            CompactCell("VECTOR", "$windDir ${weather.windDirection.toInt()}°", Color.Yellow, Modifier.weight(1f))
        }

        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(2.dp)) {
            CompactCell("CAPE", String.format("%.0fJ", weather.cape), if (weather.cape > 100) Color.Red else Color.Green, Modifier.weight(1f))
            CompactCell("SOLAR", String.format("%.0fW", weather.solarFlux), Color(0xFFFF9800), Modifier.weight(1f))
            CompactCell("TREND", tendency.second, Color.Magenta, Modifier.weight(1f))
        }

        Spacer(modifier = Modifier.weight(1f))

        // Small bar trend chart in Compact Mode
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .height(16.dp)
                .padding(horizontal = 4.dp),
            horizontalArrangement = Arrangement.spacedBy(1.dp),
            verticalAlignment = Alignment.Bottom
        ) {
            val maxPress = weather.hourlyPressures.maxOrNull() ?: 1.0
            val minPress = weather.hourlyPressures.minOrNull() ?: 0.0
            val span = if (maxPress - minPress == 0.0) 1.0 else maxPress - minPress
            
            weather.hourlyPressures.forEach { pressure ->
                val ratio = (pressure - minPress) / span
                Box(
                    modifier = Modifier
                        .weight(1f)
                        .fillMaxHeight(ratio.toFloat().coerceIn(0.1f, 1.0f))
                        .background(Color(0xFF00E5FF))
                )
            }
        }
    }
}

// MARK: - COMPACT CELL
@Composable
fun CompactCell(title: String, valStr: String, color: Color, modifier: Modifier = Modifier) {
    Column(
        modifier = modifier
            .background(Color.White.copy(alpha = 0.04f), RoundedCornerShape(3.dp))
            .padding(4.dp),
        horizontalAlignment = Alignment.Start
    ) {
        Text(title, fontSize = 7.sp, fontFamily = FontFamily.Monospace, color = Color.Gray)
        Text(valStr, fontSize = 11.sp, fontFamily = FontFamily.Monospace, fontWeight = FontWeight.Bold, color = color, maxLines = 1)
    }
}

// MARK: - SUBCOMPONENTS
@Composable
fun MetricSection(title: String, content: @Composable ColumnScope.() -> Unit) {
    Column(modifier = Modifier.fillMaxWidth()) {
        Text(
            title,
            fontSize = 10.sp,
            fontFamily = FontFamily.Monospace,
            fontWeight = FontWeight.Bold,
            color = Color(0xFF00E5FF),
            modifier = Modifier.padding(bottom = 6.dp)
        )
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .background(Color.White.copy(alpha = 0.04f), RoundedCornerShape(8.dp))
                .padding(10.dp),
            verticalArrangement = Arrangement.spacedBy(6.dp),
            content = content
        )
    }
}

@Composable
fun MetricRow(label: String, value: String) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween
    ) {
        Text(label, fontSize = 13.sp, fontFamily = FontFamily.Monospace, color = Color.Gray)
        Text(value, fontSize = 13.sp, fontFamily = FontFamily.Monospace, fontWeight = FontWeight.Bold, color = Color.White)
    }
}

// Draw line path sparkline using canvas
@Composable
fun Sparkline(values: List<Double>, modifier: Modifier = Modifier) {
    Canvas(modifier = modifier) {
        if (values.size < 2) return@Canvas
        val maxVal = values.maxOrNull() ?: 1.0
        val minVal = values.minOrNull() ?: 0.0
        val range = if (maxVal - minVal == 0.0) 1.0 else maxVal - minVal
        
        val width = size.width
        val height = size.height
        val stepX = width / (values.size - 1)
        
        val path = Path()
        values.forEachIndexed { index, value ->
            val x = index * stepX
            val norm = (value - minVal) / range
            val y = height - (norm.toFloat() * height)
            
            if (index == 0) {
                path.moveTo(x, y)
            } else {
                path.lineTo(x, y)
            }
        }
        
        // Draw path line
        drawPath(
            path = path,
            color = Color(0xFF00E5FF),
            style = Stroke(width = 4f)
        )
        
        // Draw fill gradient area
        val fillPath = Path().apply {
            addPath(path)
            lineTo(width, height)
            lineTo(0f, height)
            close()
        }
        
        drawPath(
            path = fillPath,
            brush = Brush.verticalGradient(
                colors = listOf(Color(0xFF00E5FF).copy(alpha = 0.3f), Color.Transparent),
                startY = 0f,
                endY = height
            )
        )
    }
}

// Unit formatters
private fun formatTemp(c: Double, useImperial: Boolean): String {
    return if (useImperial) String.format("%.1f°F", c * 1.8 + 32.0) else String.format("%.1f°C", c)
}

private fun formatDistance(m: Double, useImperial: Boolean): String {
    return if (useImperial) String.format("%.0fft", m * 3.28084) else String.format("%.0fm", m)
}

private fun formatPressure(hpa: Double, useImperial: Boolean): String {
    return if (useImperial) String.format("%.2finHg", hpa * 0.02953) else String.format("%.1fhPa", hpa)
}

private fun formatSpeed(mps: Double, useImperial: Boolean): String {
    return if (useImperial) String.format("%.1fmph", mps * 2.23694) else String.format("%.1fm/s", mps)
}
