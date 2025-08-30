package services

import (
	"fmt"
	"kaelo/config"
	"kaelo/models"
	"math"
	"time"
)

type AnomalyDetectionService struct {
	config *config.Config
}

func NewAnomalyDetectionService(cfg *config.Config) *AnomalyDetectionService {
	return &AnomalyDetectionService{
		config: cfg,
	}
}

// DetectAnomalies analyzes sensor data and returns any detected anomalies
func (s *AnomalyDetectionService) DetectAnomalies(data *models.SensorData) []*models.Anomaly {
	var anomalies []*models.Anomaly

	// Temperature anomalies for DHT sensor
	if data.TemperatureDHT > s.config.TemperatureMax {
		anomalies = append(anomalies, &models.Anomaly{
			Type:        models.TemperatureTooHigh,
			Value:       data.TemperatureDHT,
			Threshold:   s.config.TemperatureMax,
			DeviceID:    data.DeviceID,
			Description: fmt.Sprintf("DHT Temperature %.1f°C exceeds threshold %.1f°C", data.TemperatureDHT, s.config.TemperatureMax),
			Timestamp:   time.Now(),
		})
	}

	if data.TemperatureDHT < s.config.TemperatureMin {
		anomalies = append(anomalies, &models.Anomaly{
			Type:        models.TemperatureTooLow,
			Value:       data.TemperatureDHT,
			Threshold:   s.config.TemperatureMin,
			DeviceID:    data.DeviceID,
			Description: fmt.Sprintf("DHT Temperature %.1f°C below threshold %.1f°C", data.TemperatureDHT, s.config.TemperatureMin),
			Timestamp:   time.Now(),
		})
	}

	// TODO: remove this because TemperatureMPU is deprecated
	// Temperature differential detection
	// if data.TemperatureDHT > 0 && data.TemperatureMPU > 0 {
	// 	diff := math.Abs(data.TemperatureDHT - data.TemperatureMPU)
	// 	avg := (data.TemperatureDHT + data.TemperatureMPU) / 2
	// 	if avg > 0 && (diff/avg)*100 > 20 { // 20% difference threshold
	// 		anomalies = append(anomalies, &models.Anomaly{
	// 			Type:        models.TemperatureDifferential,
	// 			Value:       diff,
	// 			Threshold:   20.0,
	// 			DeviceID:    data.DeviceID,
	// 			Description: fmt.Sprintf("Temperature differential %.1f°C between DHT (%.1f°C) and MPU (%.1f°C)", diff, data.TemperatureDHT, data.TemperatureMPU),
	// 			Timestamp:   time.Now(),
	// 		})
	// 	}
	// }

	// Check humidity anomalies
	if data.Humidity > s.config.HumidityMax {
		anomalies = append(anomalies, &models.Anomaly{
			Type:        models.HumidityTooHigh,
			Value:       data.Humidity,
			Threshold:   s.config.HumidityMax,
			DeviceID:    data.DeviceID,
			Timestamp:   data.Timestamp,
			Description: fmt.Sprintf("Humidity %.1f%% exceeds maximum threshold of %.1f%%", data.Humidity, s.config.HumidityMax),
		})
	}

	if data.Humidity < s.config.HumidityMin {
		anomalies = append(anomalies, &models.Anomaly{
			Type:        models.HumidityTooLow,
			Value:       data.Humidity,
			Threshold:   s.config.HumidityMin,
			DeviceID:    data.DeviceID,
			Timestamp:   data.Timestamp,
			Description: fmt.Sprintf("Humidity %.1f%% is below minimum threshold of %.1f%%", data.Humidity, s.config.HumidityMin),
		})
	}

	// Gas quality anomalies
	switch data.GasQuality {
	case "poor":
		anomalies = append(anomalies, &models.Anomaly{
			Type:        models.GasQualityPoor,
			DeviceID:    data.DeviceID,
			Description: "Air quality is poor - immediate attention required",
			Timestamp:   time.Now(),
		})
	case "moderate":
		anomalies = append(anomalies, &models.Anomaly{
			Type:        models.GasQualityModerate,
			DeviceID:    data.DeviceID,
			Description: "Air quality is moderate - monitor closely",
			Timestamp:   time.Now(),
		})
	}

	// Flame detection
	if data.FlameDetected {
		anomalies = append(anomalies, &models.Anomaly{
			Type:        models.FlameDetected,
			DeviceID:    data.DeviceID,
			Description: "Flame detected - emergency response required",
			Timestamp:   time.Now(),
		})
	}

	// Gyroscope anomaly detection
	gyroMagnitude := math.Sqrt(data.Gyroscope.X*data.Gyroscope.X + data.Gyroscope.Y*data.Gyroscope.Y + data.Gyroscope.Z*data.Gyroscope.Z)
	if gyroMagnitude > 5.0 { // Threshold for abnormal angular velocity
		anomalies = append(anomalies, &models.Anomaly{
			Type:        models.GyroscopeAbnormal,
			Value:       gyroMagnitude,
			Threshold:   5.0,
			DeviceID:    data.DeviceID,
			Description: fmt.Sprintf("Abnormal gyroscope reading: %.2f rad/s", gyroMagnitude),
			Timestamp:   time.Now(),
		})
	}

	// Acceleration anomaly detection
	accMagnitude := math.Sqrt(data.Acceleration.X*data.Acceleration.X + data.Acceleration.Y*data.Acceleration.Y + data.Acceleration.Z*data.Acceleration.Z)
	if accMagnitude > 15.0 { // Threshold for abnormal acceleration
		anomalies = append(anomalies, &models.Anomaly{
			Type:        models.AccelerationAbnormal,
			Value:       accMagnitude,
			Threshold:   15.0,
			DeviceID:    data.DeviceID,
			Description: fmt.Sprintf("Abnormal acceleration detected: %.2f m/s²", accMagnitude),
			Timestamp:   time.Now(),
		})
	}

	return anomalies
}

// IsAnomalous returns true if any anomalies are detected
func (s *AnomalyDetectionService) IsAnomalous(data *models.SensorData) bool {
	anomalies := s.DetectAnomalies(data)
	return len(anomalies) > 0
}
