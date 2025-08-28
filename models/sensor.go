package models

import (
	"time"
)

// SensorData represents the data structure from ESP32 sensors
type SensorData struct {
	DeviceID    string    `json:"device_id"`
	Temperature float64   `json:"temperature"`
	Humidity    float64   `json:"humidity"`
	Dust        float64   `json:"dust"`
	Flame       float64   `json:"flame"`     // KY-026 Flame sensor (0-1023)
	Light       float64   `json:"light"`     // KY-018 Photo resistor (0-1023)
	Vibration   float64   `json:"vibration"` // KY-002 Vibration switch (0/1)
	Gas         float64   `json:"gas"`       // MQ135 Gas sensor (0-1023 PPM)
	Timestamp   time.Time `json:"timestamp"`
}

// AnomalyType represents different types of anomalies
type AnomalyType string

const (
	TemperatureTooHigh AnomalyType = "temperature_high"
	TemperatureTooLow  AnomalyType = "temperature_low"
	HumidityTooHigh    AnomalyType = "humidity_high"
	HumidityTooLow     AnomalyType = "humidity_low"
	DustTooHigh        AnomalyType = "dust_high"
	FlameDetected      AnomalyType = "flame_detected"
	LightTooLow        AnomalyType = "light_low"
	LightTooHigh       AnomalyType = "light_high"
	VibrationDetected  AnomalyType = "vibration_detected"
	GasTooHigh         AnomalyType = "gas_high"
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
	case HumidityTooHigh:
		return "💧"
	case HumidityTooLow:
		return "🏜️"
	case DustTooHigh:
		return "💨"
	case FlameDetected:
		return "🚨"
	case LightTooLow:
		return "🌑"
	case LightTooHigh:
		return "☀️"
	case VibrationDetected:
		return "📳"
	case GasTooHigh:
		return "☠️"
	default:
		return "⚠️"
	}
}

// GetSeverityColor returns color for Telegram formatting
func (a *Anomaly) GetSeverityColor() string {
	// Return HTML color codes for Telegram
	switch a.Type {
	case TemperatureTooHigh, DustTooHigh, FlameDetected, GasTooHigh:
		return "🔴" // Red for high severity
	case TemperatureTooLow, HumidityTooLow, VibrationDetected:
		return "🟡" // Yellow for medium severity
	case HumidityTooHigh, LightTooLow, LightTooHigh:
		return "🔵" // Blue for environmental issues
	default:
		return "⚪" // White for unknown
	}
}
