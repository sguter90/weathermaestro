package models

// SensorType constants for standard sensor types
const (
	SensorTypeTemperature        = "Temperature"
	SensorTypeHumidity           = "Humidity"
	SensorTypePressure           = "Pressure"
	SensorTypeWindDirection      = "WindDirection"
	SensorTypeWindSpeed          = "WindSpeed"
	SensorTypeWindSpeedMaxDaily  = "WindSpeedMaxDaily"
	SensorTypeWindGust           = "WindGust"
	SensorTypeWindGustAngle      = "WindGustAngle"
	SensorTypeWindGustMaxDaily   = "WindGustMaxDaily"
	SensorTypeSolarRadiation     = "SolarRadiation"
	SensorTypeUVIndex            = "UVIndex"
	SensorTypeRainfallRate       = "RainfallRate"
	SensorTypeRainfallEvent      = "RainfallEvent"
	SensorTypeRainfallHourly     = "RainfallHourly"
	SensorTypeRainfallDaily      = "RainfallDaily"
	SensorTypeRainfallWeekly     = "RainfallWeekly"
	SensorTypeRainfallMonthly    = "RainfallMonthly"
	SensorTypeRainfallYearly     = "RainfallYearly"
	SensorTypeRainfallTotal      = "RainfallTotal"
	SensorTypeVPD                = "VPD"
	SensorTypeBattery            = "Battery"
	SensorTypePressureRelative   = "PressureRelative"
	SensorTypePressureAbsolute   = "PressureAbsolute"
	SensorTypeTemperatureOutdoor = "TemperatureOutdoor"
	SensorTypeHumidityOutdoor    = "HumidityOutdoor"
	SensorTypeSignalStrength     = "SignalStrength"
	SensorTypeCO2                = "CO2"
	SensorTypeNoise              = "Noise"
)

// SensorCategory constants for standard sensor categories
const (
	SensorCategoryTemperature = "Temperature"
	SensorCategoryHumidity    = "Humidity"
	SensorCategoryPressure    = "Pressure"
	SensorCategoryWind        = "Wind"
	SensorCategoryRain        = "Rainfall"
	SensorCategorySolar       = "Solar"
	SensorCategoryVapor       = "Vapor"
	SensorCategorySystem      = "System"
	SensorCategoryC02         = "CO2"
	SensorCategoryNoise       = "Noise"
)

// SensorType represents a standardized sensor type
type SensorType struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Unit     string `json:"unit"`
}

// SensorTypeInfo holds metadata about sensor types
type SensorTypeInfo struct {
	Name     string
	Category string
	Unit     string
}

// SensorTypeRegistry maps sensor type IDs to their information
var SensorTypeRegistry = map[string]SensorTypeInfo{
	SensorTypeTemperature: {
		Name:     SensorTypeTemperature,
		Category: SensorCategoryTemperature,
		Unit:     "°C",
	},
	SensorTypeHumidity: {
		Name:     SensorTypeHumidity,
		Category: SensorCategoryHumidity,
		Unit:     "%",
	},
	SensorTypePressure: {
		Name:     SensorTypePressure,
		Category: SensorCategoryPressure,
		Unit:     "hPa",
	},
	SensorTypeWindSpeed: {
		Name:     SensorTypeWindSpeed,
		Category: SensorCategoryWind,
		Unit:     "m/s",
	},
	SensorTypeWindSpeedMaxDaily: {
		Name:     SensorTypeWindSpeedMaxDaily,
		Category: SensorCategoryWind,
		Unit:     "m/s",
	},
	SensorTypeWindDirection: {
		Name:     SensorTypeWindDirection,
		Category: SensorCategoryWind,
		Unit:     "°",
	},
	SensorTypeWindGust: {
		Name:     SensorTypeWindGust,
		Category: SensorCategoryWind,
		Unit:     "m/s",
	},
	SensorTypeWindGustAngle: {
		Name:     SensorTypeWindGustAngle,
		Category: SensorCategoryWind,
		Unit:     "°",
	},
	SensorTypeWindGustMaxDaily: {
		Name:     SensorTypeWindGustMaxDaily,
		Category: SensorCategoryWind,
		Unit:     "m/s",
	},
	SensorTypeSolarRadiation: {
		Name:     SensorTypeSolarRadiation,
		Category: SensorCategorySolar,
		Unit:     "W/m²",
	},
	SensorTypeUVIndex: {
		Name:     SensorTypeUVIndex,
		Category: SensorCategorySolar,
		Unit:     "index",
	},
	SensorTypeRainfallRate: {
		Name:     SensorTypeRainfallRate,
		Category: SensorCategoryRain,
		Unit:     "mm/h",
	},
	SensorTypeRainfallEvent: {
		Name:     SensorTypeRainfallEvent,
		Category: SensorCategoryRain,
		Unit:     "mm",
	},
	SensorTypeRainfallHourly: {
		Name:     SensorTypeRainfallHourly,
		Category: SensorCategoryRain,
		Unit:     "mm",
	},
	SensorTypeRainfallDaily: {
		Name:     SensorTypeRainfallDaily,
		Category: SensorCategoryRain,
		Unit:     "mm",
	},
	SensorTypeRainfallWeekly: {
		Name:     SensorTypeRainfallWeekly,
		Category: SensorCategoryRain,
		Unit:     "mm",
	},
	SensorTypeRainfallMonthly: {
		Name:     SensorTypeRainfallMonthly,
		Category: SensorCategoryRain,
		Unit:     "mm",
	},
	SensorTypeRainfallYearly: {
		Name:     SensorTypeRainfallYearly,
		Category: SensorCategoryRain,
		Unit:     "mm",
	},
	SensorTypeRainfallTotal: {
		Name:     SensorTypeRainfallTotal,
		Category: SensorCategoryRain,
		Unit:     "mm",
	},
	SensorTypeVPD: {
		Name:     SensorTypeVPD,
		Category: SensorCategoryVapor,
		Unit:     "kPa",
	},
	SensorTypeBattery: {
		Name:     SensorTypeBattery,
		Category: SensorCategorySystem,
		Unit:     "%",
	},
	SensorTypePressureRelative: {
		Name:     SensorTypePressureRelative,
		Category: SensorCategoryPressure,
		Unit:     "hPa",
	},
	SensorTypePressureAbsolute: {
		Name:     SensorTypePressureAbsolute,
		Category: SensorCategoryPressure,
		Unit:     "hPa",
	},
	SensorTypeTemperatureOutdoor: {
		Name:     SensorTypeTemperatureOutdoor,
		Category: SensorCategoryTemperature,
		Unit:     "°C",
	},
	SensorTypeHumidityOutdoor: {
		Name:     SensorTypeHumidityOutdoor,
		Category: SensorCategoryHumidity,
		Unit:     "%",
	},
	SensorTypeSignalStrength: {
		Name:     SensorTypeSignalStrength,
		Category: SensorCategorySystem,
		Unit:     "dBm",
	},
	SensorTypeCO2: {
		Name:     SensorTypeCO2,
		Category: SensorCategoryC02,
		Unit:     "ppm",
	},
	SensorTypeNoise: {
		Name:     SensorTypeNoise,
		Category: SensorCategoryNoise,
		Unit:     "dB",
	},
}
