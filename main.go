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
	if cfg.RabbitMQURL == "" {
		logger.Fatal("RabbitMQ configuration is required")
	}
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

	anomalyDetector := services.NewAnomalyDetectionService(cfg)

	// Initialize hardware alert service
	var hardwareAlertService *services.HardwareAlertService
	if cfg.HardwareAlertURL != "" {
		hardwareAlertService = services.NewHardwareAlertService(logger, cfg.HardwareAlertURL)
		logger.Info("Hardware alert service initialized", zap.String("url", cfg.HardwareAlertURL))
	}

	// Initialize RabbitMQ service
	rabbitMQService, err := services.NewRabbitMQService(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to initialize RabbitMQ service", zap.Error(err))
	}
	defer rabbitMQService.Close()

	// Initialize batch writer service
	batchWriterService := services.NewBatchWriterService(cfg, firebaseService, logger)

	// Initialize face recognition service
	faceRecognitionService := services.NewFaceRecognitionService(telegramService, logger)

	// Initialize health check monitoring service
	healthCheckService := services.NewHealthCheckService(cfg, telegramService, logger)

	// Send startup notification
	if err := telegramService.SendStartupMessage(); err != nil {
		logger.Warn("Failed to send startup message", zap.Error(err))
	}

	logger.Info("KAELO IoT Monitoring Service started",
		zap.String("rabbitmq_url", cfg.RabbitMQURL),
		zap.String("rabbitmq_queue", cfg.RabbitMQQueue),
		zap.Int("firebase_batch_size", cfg.FirebaseBatchSize),
		zap.Int("firebase_batch_timeout", cfg.FirebaseBatchTimeout),
		zap.Float64("temp_min", cfg.TemperatureMin),
		zap.Float64("temp_max", cfg.TemperatureMax),
		zap.Float64("humidity_min", cfg.HumidityMin),
		zap.Float64("humidity_max", cfg.HumidityMax),
		zap.Float64("dust_max", cfg.DustMax),
		zap.Float64("flame_threshold", cfg.FlameThreshold),
		zap.Float64("light_min", cfg.LightMin),
		zap.Float64("light_max", cfg.LightMax),
		zap.Float64("gas_max", cfg.GasMax),
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
		case <-time.After(10 * time.Second):
			logger.Warn("Cleanup timeout, forcing exit")
		}

		logger.Info("KAELO IoT Monitoring Service stopped")
		os.Exit(0)
	}()

	// Create channels for sensor data processing
	// Buffer size should be large enough to handle burst traffic
	businessLogicChan := make(chan *models.SensorData, 200)
	batchWriterChan := make(chan *models.SensorData, 200)
	faceRecognitionChan := make(chan *models.FaceRecognitionData, 100)
	healthCheckChan := make(chan *models.HealthCheckData, 100)

	// Start Process 1: Business Logic Processing (Anomaly Detection + Alerts)
	go func() {
		logger.Info("Starting business logic processor")
		for {
			select {
			case <-ctx.Done():
				logger.Info("Business logic processor stopped")
				return
			case sensorData, ok := <-businessLogicChan:
				if !ok {
					logger.Info("Business logic channel closed")
					return
				}

				// Detect anomalies
				anomalies := anomalyDetector.DetectAnomalies(sensorData)

				if len(anomalies) > 0 {
					logger.Warn("Anomalies detected",
						zap.String("device_id", sensorData.DeviceID),
						zap.Int("anomaly_count", len(anomalies)),
						zap.Float64("temperature_dht", sensorData.TemperatureDHT),
						zap.Float64("temperature_mpu", sensorData.TemperatureMPU),
						zap.Float64("humidity", sensorData.Humidity),
						zap.String("gas_quality", sensorData.GasQuality),
						zap.Bool("flame_detected", sensorData.FlameDetected),
						zap.Any("acceleration", sensorData.Acceleration),
						zap.Any("gyroscope", sensorData.Gyroscope),
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

					// Send hardware alert if service is configured
					if hardwareAlertService != nil {
						if err := hardwareAlertService.SendHardwareAlert(anomalies, sensorData); err != nil {
							logger.Error("Failed to send hardware alert",
								zap.String("device_id", sensorData.DeviceID),
								zap.Error(err),
							)
						} else {
							logger.Info("Hardware alert sent",
								zap.String("device_id", sensorData.DeviceID),
								zap.Int("anomaly_count", len(anomalies)),
							)
						}
					}
				}
			}
		}
	}()

	// Start Process 2: Batch Writer for Firebase
	go batchWriterService.Start(ctx, batchWriterChan)

	// Start Process 3: Face Recognition Processor
	go faceRecognitionService.Start(ctx, faceRecognitionChan)

	// Start Process 4: Health Check Monitoring
	go healthCheckService.Start(ctx, healthCheckChan)

	// Start RabbitMQ consumers
	go func() {
		logger.Info("Starting RabbitMQ sensor data consumer and message distributor")

		// Create a single channel for RabbitMQ sensor messages
		rabbitMQChan := make(chan *models.SensorData, 100)

		// Start RabbitMQ sensor data consumer
		go func() {
			if err := rabbitMQService.ConsumeSensorData(ctx, rabbitMQChan); err != nil {
				logger.Error("RabbitMQ sensor consumer error", zap.Error(err))
			}
		}()

		// Distribute messages to both processing channels
		for {
			select {
			case <-ctx.Done():
				logger.Info("Message distributor stopped")
				close(businessLogicChan)
				close(batchWriterChan)
				return
			case sensorData, ok := <-rabbitMQChan:
				if !ok {
					logger.Info("RabbitMQ channel closed")
					close(businessLogicChan)
					close(batchWriterChan)
					return
				}

				// Send to both processes (non-blocking with timeout)
				// Process 1: Business Logic
				select {
				case businessLogicChan <- sensorData:
				case <-time.After(1 * time.Second):
					logger.Warn("Timeout sending to business logic channel",
						zap.String("device_id", sensorData.DeviceID))
				}

				// Process 2: Batch Writer
				select {
				case batchWriterChan <- sensorData:
				case <-time.After(1 * time.Second):
					logger.Warn("Timeout sending to batch writer channel",
						zap.String("device_id", sensorData.DeviceID))
				}
			}
		}
	}()

	// Start Face Recognition Consumer
	go func() {
		logger.Info("Starting RabbitMQ face recognition consumer")

		if err := rabbitMQService.ConsumeFaceRecognitionData(ctx, faceRecognitionChan); err != nil {
			logger.Error("RabbitMQ face recognition consumer error", zap.Error(err))
		}
	}()

	// Start Health Check Consumer
	go func() {
		logger.Info("Starting RabbitMQ health check consumer")

		if err := rabbitMQService.ConsumeHealthCheck(ctx, healthCheckChan); err != nil {
			logger.Error("RabbitMQ health check consumer error", zap.Error(err))
		}
	}()

	logger.Info("All services started, waiting for messages from RabbitMQ")

	// Wait for shutdown signal
	<-ctx.Done()

	// Perform cleanup
	logger.Info("Starting cleanup")

	// Wait for batch writer to finish flushing
	logger.Info("Waiting for batch writer to flush remaining data")
	if batchWriterService.WaitForShutdown(5 * time.Second) {
		logger.Info("Batch writer shutdown completed")
	} else {
		logger.Warn("Batch writer shutdown timeout")
	}

	// Close RabbitMQ service (will close all consumers)
	if err := rabbitMQService.Close(); err != nil {
		logger.Error("Error closing RabbitMQ service", zap.Error(err))
	} else {
		logger.Info("RabbitMQ service closed")
	}

	// Close Firebase service
	if err := firebaseService.Close(); err != nil {
		logger.Error("Error closing Firebase service", zap.Error(err))
	} else {
		logger.Info("Firebase service closed")
	}

	// Signal cleanup completion
	cleanupDone <- true
}
