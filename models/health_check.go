package models

import (
	"time"
)

// DeviceHealthStatus represents the health status of a device
type DeviceHealthStatus string

const (
	DeviceHealthy   DeviceHealthStatus = "healthy"
	DeviceTimeout   DeviceHealthStatus = "timeout"
	DeviceRecovered DeviceHealthStatus = "recovered"
)

// SensorStatus represents the status of individual sensors on the device
type SensorStatus struct {
	DHT11   bool `json:"dht11"`
	MPU6050 bool `json:"mpu6050"`
	Flame   bool `json:"flame"`
	Gas     bool `json:"gas"`
}

// HealthCheckData represents health check data from ESP32 devices
type HealthCheckData struct {
	DeviceID      string       `json:"device_id"`
	Timestamp     time.Time    `json:"timestamp"`
	WiFiConnected bool         `json:"wifi_connected"`
	MQTTConnected bool         `json:"mqtt_connected"`
	UptimeMs      int64        `json:"uptime_ms"`
	Sensors       SensorStatus `json:"sensors"`
}

// DeviceHealth tracks the health state of a device
type DeviceHealth struct {
	DeviceID        string
	LastHealthCheck *HealthCheckData
	LastSeen        time.Time
	Status          DeviceHealthStatus
	TimeoutAt       time.Time // When the device timed out (if applicable)
}
