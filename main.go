package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kaelo/config"
	"kaelo/log"
	"kaelo/models"
	"kaelo/services"

	"go.uber.org/zap"
)

func main() {
	// Initialize timezone to Asia/Bangkok
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		panic("Failed to load Asia/Bangkok timezone: " + err.Error())
	}
	time.Local = loc

	// Initialize structured logger
	logger := log.GetInstance()
	defer logger.Sync()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	// Validate required configuration
	if cfg.FirebaseDbUrl == "" || cfg.FirebaseServiceAccountJSON == "" {
		logger.Fatal("Firebase configuration is required")
	}
	if cfg.TelegramBotToken == "" || cfg.TelegramChatID == "" {
		logger.Fatal("Telegram configuration is required")
	}

	// Initialize services
	firebaseService, err := services.NewFirebaseService(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize Firebase service", zap.Error(err))
	}
	defer firebaseService.Close()

	telegramService, err := services.NewTelegramService(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize Telegram service", zap.Error(err))
	}

	anomalyDetector := services.NewAnomalyDetector(cfg)

	// Send startup notification
	if err := telegramService.SendStartupMessage(); err != nil {
		logger.Warn("Failed to send startup message", zap.Error(err))
	}

	logger.Info("KAELO IoT Monitoring Service started",
		zap.Float64("temp_min", cfg.TemperatureMin),
		zap.Float64("temp_max", cfg.TemperatureMax),
		zap.Float64("humidity_min", cfg.HumidityMin),
		zap.Float64("humidity_max", cfg.HumidityMax),
		zap.Float64("dust_max", cfg.DustMax),
	)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Channel to signal when cleanup is complete
	cleanupDone := make(chan bool, 1)

	go func() {
		<-sigChan
		logger.Info("Shutdown signal received, stopping services")

		// Cancel context to stop all goroutines
		cancel()

		// Wait for cleanup to complete or timeout
		select {
		case <-cleanupDone:
			logger.Info("Cleanup completed successfully")
		case <-time.After(5 * time.Second):
			logger.Warn("Cleanup timeout, forcing exit")
		}

		logger.Info("KAELO IoT Monitoring Service stopped")
		os.Exit(0)
	}()

	// Subscribe to Firebase sensor data
	err = firebaseService.SubscribeToSensorData(ctx, func(sensorData *models.SensorData) {
		// Detect anomalies
		anomalies := anomalyDetector.DetectAnomalies(sensorData)

		if len(anomalies) > 0 {
			logger.Warn("Anomalies detected",
				zap.String("device_id", sensorData.DeviceID),
				zap.Int("anomaly_count", len(anomalies)),
				zap.Float64("temperature", sensorData.Temperature),
				zap.Float64("humidity", sensorData.Humidity),
				zap.Float64("dust", sensorData.Dust),
			)

			// Send Telegram notification
			if err := telegramService.SendAnomalyAlert(anomalies, sensorData); err != nil {
				logger.Error("Failed to send Telegram alert",
					zap.String("device_id", sensorData.DeviceID),
					zap.Error(err),
				)
			} else {
				logger.Info("Anomaly alert sent",
					zap.String("device_id", sensorData.DeviceID),
					zap.Int("anomaly_count", len(anomalies)),
				)
			}
		}
	})

	if err != nil {
		logger.Fatal("Failed to subscribe to sensor data", zap.Error(err))
	}

	logger.Info("Monitoring started, waiting for sensor data")

	// Wait for shutdown signal
	<-ctx.Done()

	// Perform cleanup
	logger.Info("Starting cleanup")

	// Close Firebase service
	if err := firebaseService.Close(); err != nil {
		logger.Error("Error closing Firebase service", zap.Error(err))
	} else {
		logger.Info("Firebase service closed")
	}

	// Signal cleanup completion
	cleanupDone <- true
}
