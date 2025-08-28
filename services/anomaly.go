package services

import (
	"fmt"
	"kaelo/config"
	"kaelo/models"
)

type AnomalyDetector struct {
	config *config.Config
}

func NewAnomalyDetector(cfg *config.Config) *AnomalyDetector {
	return &AnomalyDetector{
		config: cfg,
	}
}

// DetectAnomalies analyzes sensor data and returns any detected anomalies
func (ad *AnomalyDetector) DetectAnomalies(data *models.SensorData) []*models.Anomaly {
	var anomalies []*models.Anomaly

	// Check temperature anomalies
	if data.Temperature > ad.config.TemperatureMax {
		anomalies = append(anomalies, &models.Anomaly{
			Type:        models.TemperatureTooHigh,
			Value:       data.Temperature,
			Threshold:   ad.config.TemperatureMax,
			DeviceID:    data.DeviceID,
			Timestamp:   data.Timestamp,
			Description: fmt.Sprintf("Temperature %.1f°C exceeds maximum threshold of %.1f°C", data.Temperature, ad.config.TemperatureMax),
		})
	}

	if data.Temperature < ad.config.TemperatureMin {
		anomalies = append(anomalies, &models.Anomaly{
			Type:        models.TemperatureTooLow,
			Value:       data.Temperature,
			Threshold:   ad.config.TemperatureMin,
			DeviceID:    data.DeviceID,
			Timestamp:   data.Timestamp,
			Description: fmt.Sprintf("Temperature %.1f°C is below minimum threshold of %.1f°C", data.Temperature, ad.config.TemperatureMin),
		})
	}

	// Check humidity anomalies
	if data.Humidity > ad.config.HumidityMax {
		anomalies = append(anomalies, &models.Anomaly{
			Type:        models.HumidityTooHigh,
			Value:       data.Humidity,
			Threshold:   ad.config.HumidityMax,
			DeviceID:    data.DeviceID,
			Timestamp:   data.Timestamp,
			Description: fmt.Sprintf("Humidity %.1f%% exceeds maximum threshold of %.1f%%", data.Humidity, ad.config.HumidityMax),
		})
	}

	if data.Humidity < ad.config.HumidityMin {
		anomalies = append(anomalies, &models.Anomaly{
			Type:        models.HumidityTooLow,
			Value:       data.Humidity,
			Threshold:   ad.config.HumidityMin,
			DeviceID:    data.DeviceID,
			Timestamp:   data.Timestamp,
			Description: fmt.Sprintf("Humidity %.1f%% is below minimum threshold of %.1f%%", data.Humidity, ad.config.HumidityMin),
		})
	}

	// Check dust anomalies
	if data.Dust > ad.config.DustMax {
		anomalies = append(anomalies, &models.Anomaly{
			Type:        models.DustTooHigh,
			Value:       data.Dust,
			Threshold:   ad.config.DustMax,
			DeviceID:    data.DeviceID,
			Timestamp:   data.Timestamp,
			Description: fmt.Sprintf("Dust level %.1f μg/m³ exceeds maximum threshold of %.1f μg/m³", data.Dust, ad.config.DustMax),
		})
	}

	// Check flame sensor anomalies
	if data.Flame > ad.config.FlameThreshold {
		anomalies = append(anomalies, &models.Anomaly{
			Type:        models.FlameDetected,
			Value:       data.Flame,
			Threshold:   ad.config.FlameThreshold,
			DeviceID:    data.DeviceID,
			Timestamp:   data.Timestamp,
			Description: fmt.Sprintf("Flame detected! Sensor value: %.0f (threshold: %.0f)", data.Flame, ad.config.FlameThreshold),
		})
	}

	// Check light sensor anomalies
	if data.Light < ad.config.LightMin {
		anomalies = append(anomalies, &models.Anomaly{
			Type:        models.LightTooLow,
			Value:       data.Light,
			Threshold:   ad.config.LightMin,
			DeviceID:    data.DeviceID,
			Timestamp:   data.Timestamp,
			Description: fmt.Sprintf("Light level %.0f is too low (minimum: %.0f)", data.Light, ad.config.LightMin),
		})
	}

	if data.Light > ad.config.LightMax {
		anomalies = append(anomalies, &models.Anomaly{
			Type:        models.LightTooHigh,
			Value:       data.Light,
			Threshold:   ad.config.LightMax,
			DeviceID:    data.DeviceID,
			Timestamp:   data.Timestamp,
			Description: fmt.Sprintf("Light level %.0f is too high (maximum: %.0f)", data.Light, ad.config.LightMax),
		})
	}

	// Check vibration sensor anomalies
	if data.Vibration > 0 {
		anomalies = append(anomalies, &models.Anomaly{
			Type:        models.VibrationDetected,
			Value:       data.Vibration,
			Threshold:   0,
			DeviceID:    data.DeviceID,
			Timestamp:   data.Timestamp,
			Description: "Vibration detected! Device may be experiencing movement or impact",
		})
	}

	// Check gas sensor anomalies
	if data.Gas > ad.config.GasMax {
		anomalies = append(anomalies, &models.Anomaly{
			Type:        models.GasTooHigh,
			Value:       data.Gas,
			Threshold:   ad.config.GasMax,
			DeviceID:    data.DeviceID,
			Timestamp:   data.Timestamp,
			Description: fmt.Sprintf("Dangerous gas level detected! %.0f PPM (threshold: %.0f PPM)", data.Gas, ad.config.GasMax),
		})
	}

	return anomalies
}

// IsAnomalous returns true if any anomalies are detected
func (ad *AnomalyDetector) IsAnomalous(data *models.SensorData) bool {
	anomalies := ad.DetectAnomalies(data)
	return len(anomalies) > 0
}
