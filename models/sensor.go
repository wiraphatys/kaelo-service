package models

import (
	"time"
)

// AccelerationData represents acceleration data from MPU sensor
type AccelerationData struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// GyroscopeData represents gyroscope data from MPU sensor
type GyroscopeData struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// SensorData represents the data structure from ESP32 sensors
type SensorData struct {
	DeviceID       string           `json:"device_id"`
	TemperatureDHT float64          `json:"temperature_dht"`
	Humidity       float64          `json:"humidity"`
	GasQuality     string           `json:"gas_quality"` // "good", "moderate", "poor"
	Acceleration   AccelerationData `json:"acceleration"`
	Gyroscope      GyroscopeData    `json:"gyroscope"`
	FlameDetected  bool             `json:"flame_detected"`
	Timestamp      time.Time        `json:"timestamp"`

	// deprecated
	TemperatureMPU float64 `json:"temperature_mpu"`
}

// AnomalyType represents different types of anomalies
type AnomalyType string

const (
	TemperatureTooHigh      AnomalyType = "temperature_high"
	TemperatureTooLow       AnomalyType = "temperature_low"
	TemperatureDifferential AnomalyType = "temperature_differential"
	HumidityTooHigh         AnomalyType = "humidity_high"
	HumidityTooLow          AnomalyType = "humidity_low"
	GasQualityPoor          AnomalyType = "gas_quality_poor"
	GasQualityModerate      AnomalyType = "gas_quality_moderate"
	FlameDetected           AnomalyType = "flame_detected"
	AccelerationAbnormal    AnomalyType = "acceleration_abnormal"
	GyroscopeAbnormal       AnomalyType = "gyroscope_abnormal"
)

// Anomaly represents a detected anomaly
type Anomaly struct {
	Type        AnomalyType `json:"type"`
	Value       float64     `json:"value"`
	Threshold   float64     `json:"threshold"`
	DeviceID    string      `json:"device_id"`
	Timestamp   time.Time   `json:"timestamp"`
	Description string      `json:"description"`
}

// GetAnomalyEmoji returns appropriate emoji for anomaly type
func (a *Anomaly) GetAnomalyEmoji() string {
	switch a.Type {
	case TemperatureTooHigh:
		return "🔥"
	case TemperatureTooLow:
		return "🧊"
	case TemperatureDifferential:
		return "🌡️"
	case HumidityTooHigh:
		return "💧"
	case HumidityTooLow:
		return "🏜️"
	case GasQualityPoor:
		return "☠️"
	case GasQualityModerate:
		return "⚠️"
	case FlameDetected:
		return "🚨"
	case AccelerationAbnormal:
		return "📳"
	case GyroscopeAbnormal:
		return "🌀"
	default:
		return "⚠️"
	}
}

// GetSeverityColor returns color for Telegram formatting
func (a *Anomaly) GetSeverityColor() string {
	// Return HTML color codes for Telegram
	switch a.Type {
	case TemperatureTooHigh, FlameDetected, GasQualityPoor:
		return "🔴" // Red for high severity
	case TemperatureTooLow, HumidityTooLow, AccelerationAbnormal:
		return "🟡" // Yellow for medium severity
	case HumidityTooHigh, TemperatureDifferential, GasQualityModerate, GyroscopeAbnormal:
		return "🔵" // Blue for environmental issues
	default:
		return "⚪" // White for unknown
	}
}
