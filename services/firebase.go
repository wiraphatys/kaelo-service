package services

import (
	"context"
	"fmt"
	"time"

	"kaelo/config"
	"kaelo/models"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/db"
	"go.uber.org/zap"
	"google.golang.org/api/option"
)

type FirebaseService struct {
	client *db.Client
	config *config.Config
	logger *zap.Logger
}

func NewFirebaseService(cfg *config.Config) (*FirebaseService, error) {
	logger, _ := zap.NewProduction()
	ctx := context.Background()

	// Parse the service account JSON from environment variable
	serviceAccountJSON := []byte(cfg.FirebaseServiceAccountJSON)

	// Initialize Firebase app
	conf := &firebase.Config{
		DatabaseURL: cfg.FirebaseDbUrl,
	}

	opt := option.WithCredentialsJSON(serviceAccountJSON)
	app, err := firebase.NewApp(ctx, conf, opt)
	if err != nil {
		return nil, fmt.Errorf("error initializing firebase app: %v", err)
	}

	// Get database client
	client, err := app.Database(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting database client: %v", err)
	}

	fs := &FirebaseService{
		client: client,
		config: cfg,
		logger: logger,
	}

	// Test Firebase connection with retry
	if err := fs.testConnection(); err != nil {
		logger.Error("Firebase connection test failed", zap.Error(err))
		return nil, fmt.Errorf("firebase connection test failed: %v", err)
	}

	return fs, nil
}

// testConnection tests Firebase connection with retry logic
func (fs *FirebaseService) testConnection() error {
	ctx := context.Background()
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		fs.logger.Info("Testing Firebase connection", zap.Int("attempt", attempt), zap.Int("max_retries", maxRetries))

		// Try to read from root to test connection
		ref := fs.client.NewRef("/")
		var data interface{}
		err := ref.Get(ctx, &data)

		if err == nil {
			fs.logger.Info("Firebase connection successful")
			return nil
		}

		fs.logger.Warn("Firebase connection failed",
			zap.Int("attempt", attempt),
			zap.Int("max_retries", maxRetries),
			zap.Error(err))

		if attempt < maxRetries {
			time.Sleep(time.Duration(attempt) * time.Second) // Exponential backoff
		}
	}

	return fmt.Errorf("failed to connect to Firebase after %d attempts", maxRetries)
}

// SubscribeToSensorData subscribes to sensor data updates from Firebase using optimized polling
func (fs *FirebaseService) SubscribeToSensorData(ctx context.Context, callback func(*models.SensorData)) error {
	ref := fs.client.NewRef("sensor-data")

	// Track last read timestamp and processed records
	lastReadTime := time.Now().Add(-1 * time.Minute)
	processedRecords := make(map[string]bool)

	// Start optimized polling goroutine
	go func() {
		defer fs.logger.Info("Firebase polling stopped")

		ticker := time.NewTicker(3 * time.Second) // Reduced to 3 seconds for better responsiveness
		defer ticker.Stop()

		fs.logger.Info("Starting Firebase optimized polling")

		for {
			select {
			case <-ctx.Done():
				fs.logger.Info("Firebase polling received shutdown signal")
				return
			case <-ticker.C:
				// Query records newer than lastReadTime using orderBy + startAt (index enabled)
				query := ref.OrderByChild("timestamp").StartAt(lastReadTime.Format(time.RFC3339))

				var data map[string]interface{}
				if err := query.Get(ctx, &data); err != nil {
					fs.logger.Error("Error getting sensor data", zap.Error(err))
					continue
				}

				if len(data) == 0 {
					continue // No new data
				}

				newRecordsCount := 0
				latestTimestamp := lastReadTime

				// Process each record
				for randomID, randomData := range data {
					if randomMap, ok := randomData.(map[string]interface{}); ok {
						// Skip if already processed
						if processedRecords[randomID] {
							continue
						}

						sensorData := fs.parseSensorData(randomID, randomMap)
						if sensorData != nil {
							// Only process if timestamp is newer
							if sensorData.Timestamp.After(lastReadTime) {
								processedRecords[randomID] = true
								callback(sensorData)
								newRecordsCount++

								// Update latest timestamp
								if sensorData.Timestamp.After(latestTimestamp) {
									latestTimestamp = sensorData.Timestamp
								}

								fs.logger.Debug("New sensor data received",
									zap.String("record_id", randomID),
									zap.String("device_id", sensorData.DeviceID),
									zap.Float64("temperature_dht", sensorData.TemperatureDHT),
									zap.Float64("temperature_mpu", sensorData.TemperatureMPU),
									zap.Float64("humidity", sensorData.Humidity),
									zap.String("gas_quality", sensorData.GasQuality),
									zap.Bool("flame_detected", sensorData.FlameDetected),
								)
							}
						}
					}
				}

				// Update lastReadTime
				if newRecordsCount > 0 {
					lastReadTime = latestTimestamp
					fs.logger.Info("Processed new records",
						zap.Int("count", newRecordsCount),
						zap.Time("checkpoint", lastReadTime),
					)
				}

				// Cleanup processed records cache
				if len(processedRecords) > 500 {
					newCache := make(map[string]bool)
					count := 0
					for id := range processedRecords {
						if count < 250 {
							newCache[id] = true
							count++
						}
					}
					processedRecords = newCache
					fs.logger.Debug("Cleaned processed records cache")
				}
			}
		}
	}()

	return nil
}

// parseSensorData converts Firebase data to SensorData struct
func (fs *FirebaseService) parseSensorData(randomID string, data map[string]interface{}) *models.SensorData {
	// Extract device_id from the nested data
	deviceID, deviceOk := data["device_id"].(string)
	temperatureDHT, tempDHTOk := data["temperature_dht"].(float64)
	temperatureMPU, tempMPUOk := data["temperature_mpu"].(float64)
	humidity, humOk := data["humidity"].(float64)
	gasQuality, gasOk := data["gas_quality"].(string)
	flameDetected, flameOk := data["flame_detected"].(bool)
	timestampStr, timeOk := data["timestamp"].(string)

	if !deviceOk || !tempDHTOk || !humOk || !gasOk || !timeOk {
		fs.logger.Warn("Invalid sensor data format", zap.String("record_id", randomID))
		return nil
	}

	// Set defaults for optional fields
	if !tempMPUOk {
		temperatureMPU = 0
	}
	if !flameOk {
		flameDetected = false
	}

	// Parse timestamp
	timestamp, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		fs.logger.Warn("Invalid timestamp format",
			zap.String("record_id", randomID),
			zap.Error(err))
		return nil
	}

	return &models.SensorData{
		DeviceID:       deviceID,
		TemperatureDHT: temperatureDHT,
		TemperatureMPU: temperatureMPU,
		Humidity:       humidity,
		GasQuality:     gasQuality,
		FlameDetected:  flameDetected,
		Timestamp:      timestamp,
	}
}

// GetLatestSensorData retrieves the latest sensor data for a device
func (fs *FirebaseService) GetLatestSensorData(ctx context.Context, deviceID string) (*models.SensorData, error) {
	ref := fs.client.NewRef("sensor-data")

	var data map[string]interface{}
	if err := ref.Get(ctx, &data); err != nil {
		return nil, fmt.Errorf("error getting sensor data: %v", err)
	}

	if data == nil {
		return nil, fmt.Errorf("no data found")
	}

	// Find the latest entry for the specified device
	var latestData *models.SensorData
	var latestTime time.Time

	for randomID, randomData := range data {
		if randomMap, ok := randomData.(map[string]interface{}); ok {
			sensorData := fs.parseSensorData(randomID, randomMap)
			if sensorData != nil && sensorData.DeviceID == deviceID {
				if latestData == nil || sensorData.Timestamp.After(latestTime) {
					latestData = sensorData
					latestTime = sensorData.Timestamp
				}
			}
		}
	}

	if latestData == nil {
		return nil, fmt.Errorf("no data found for device %s", deviceID)
	}

	return latestData, nil
}

// Close closes the Firebase connection
func (fs *FirebaseService) Close() error {
	fs.logger.Info("Closing Firebase service")
	// Firebase client doesn't require explicit closing but we log it
	return nil
}
