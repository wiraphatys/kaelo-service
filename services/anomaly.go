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

	return anomalies
}

// IsAnomalous returns true if any anomalies are detected
func (ad *AnomalyDetector) IsAnomalous(data *models.SensorData) bool {
	anomalies := ad.DetectAnomalies(data)
	return len(anomalies) > 0
}
