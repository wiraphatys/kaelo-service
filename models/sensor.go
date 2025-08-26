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
	default:
		return "⚠️"
	}
}

// GetSeverityColor returns color for Telegram formatting
func (a *Anomaly) GetSeverityColor() string {
	// Return HTML color codes for Telegram
	switch a.Type {
	case TemperatureTooHigh, DustTooHigh:
		return "🔴" // Red for high severity
	case TemperatureTooLow, HumidityTooLow:
		return "🟡" // Yellow for medium severity
	case HumidityTooHigh:
		return "🔵" // Blue for humidity issues
	default:
		return "⚪" // White for unknown
	}
}
