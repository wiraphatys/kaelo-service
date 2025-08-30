package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"kaelo/models"

	"go.uber.org/zap"
)

// HardwareAlertService handles hardware alert notifications
type HardwareAlertService struct {
	logger     *zap.Logger
	apiURL     string
	httpClient *http.Client
}

// HardwareAlertPayload represents the payload sent to hardware alert API
type HardwareAlertPayload struct {
	SensorData *models.SensorData `json:"sensor_data"`
	Severity   string             `json:"severity"`
	AlertType  string             `json:"alert_type"`
}

// NewHardwareAlertService creates a new hardware alert service
func NewHardwareAlertService(logger *zap.Logger, apiURL string) *HardwareAlertService {
	return &HardwareAlertService{
		logger: logger,
		apiURL: apiURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendHardwareAlert sends alert to hardware service via HTTP POST
func (h *HardwareAlertService) SendHardwareAlert(anomalies []*models.Anomaly, sensorData *models.SensorData) error {
	if len(anomalies) == 0 {
		return nil
	}

	// Determine severity based on anomaly types
	severity := h.determineSeverity(anomalies)

	payload := HardwareAlertPayload{
		SensorData: sensorData,
		Severity:   severity,
		AlertType:  "sensor_anomaly",
	}

	// Convert payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		h.logger.Error("Failed to marshal hardware alert payload",
			zap.Error(err),
			zap.String("device_id", sensorData.DeviceID),
		)
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Build endpoint
	endpoint := fmt.Sprintf("%s/api/v1/hardware-alert", h.apiURL)

	// Create HTTP request
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		h.logger.Error("Failed to create HTTP request",
			zap.Error(err),
			zap.String("url", endpoint),
		)
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "KAELO-IoT-Service/1.0")

	// Send request
	resp, err := h.httpClient.Do(req)
	if err != nil {
		h.logger.Error("Failed to send hardware alert",
			zap.Error(err),
			zap.String("device_id", sensorData.DeviceID),
			zap.String("url", endpoint),
		)
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		h.logger.Info("Hardware alert sent successfully",
			zap.String("device_id", sensorData.DeviceID),
			zap.Int("anomaly_count", len(anomalies)),
			zap.String("severity", severity),
			zap.Int("status_code", resp.StatusCode),
		)
		return nil
	}

	h.logger.Error("Hardware alert API returned error",
		zap.String("device_id", sensorData.DeviceID),
		zap.Int("status_code", resp.StatusCode),
		zap.String("status", resp.Status),
	)
	return fmt.Errorf("hardware alert API error: %s", resp.Status)
}

// determineSeverity determines alert severity based on anomaly types
func (h *HardwareAlertService) determineSeverity(anomalies []*models.Anomaly) string {
	// Check if this is a critical anomaly that needs hardware alert
	for _, anomaly := range anomalies {
		switch anomaly.Type {
		case models.FlameDetected, models.GasQualityPoor:
			return "critical"
		case models.AccelerationAbnormal:
			return "high"
		case models.GyroscopeAbnormal:
			return "high"
		}
	}

	hasHighSeverity := false
	hasMediumSeverity := false

	for _, anomaly := range anomalies {
		switch anomaly.Type {
		case models.TemperatureTooHigh, models.GasQualityModerate:
			hasHighSeverity = true
		case models.TemperatureTooLow, models.HumidityTooLow, models.TemperatureDifferential:
			hasMediumSeverity = true
		}
	}

	if hasHighSeverity {
		return "high"
	}
	if hasMediumSeverity {
		return "medium"
	}
	return "low"
}
