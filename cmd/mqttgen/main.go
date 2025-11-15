package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kaelo/models"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
)

var (
	rps        = flag.Int("rps", 1, "Requests per second (messages to send)")
	deviceID   = flag.String("device", "ESP32-MOCK-001", "Device ID for mock data")
	anomaly    = flag.Float64("anomaly", 0.1, "Probability of anomaly (0.0-1.0)")
	mqttBroker = flag.String("broker", "localhost:1883", "MQTT broker address (host:port)")
	mqttUser   = flag.String("user", "kaelo", "MQTT username")
	mqttPass   = flag.String("pass", "kaelo2024", "MQTT password")
	mqttTopic  = flag.String("topic", "sensor_data_queue", "MQTT topic to publish to")
)

type MockDataGenerator struct {
	deviceID         string
	anomalyProbility float64
	baseTemp         float64
	baseHumidity     float64
	logger           *zap.Logger
}

func NewMockDataGenerator(deviceID string, anomalyProb float64, logger *zap.Logger) *MockDataGenerator {
	return &MockDataGenerator{
		deviceID:         deviceID,
		anomalyProbility: anomalyProb,
		baseTemp:         27.0, // Base temperature ~27°C
		baseHumidity:     60.0, // Base humidity ~60%
		logger:           logger,
	}
}

// GenerateSensorData generates realistic sensor data
func (m *MockDataGenerator) GenerateSensorData() *models.SensorData {
	now := time.Now()

	// Determine if this should be an anomaly
	isAnomaly := rand.Float64() < m.anomalyProbility

	// Temperature with realistic variation
	tempVariation := rand.Float64()*4.0 - 2.0 // ±2°C variation
	temperature := m.baseTemp + tempVariation

	if isAnomaly {
		// Sometimes generate high temperature anomaly
		if rand.Float64() < 0.5 {
			temperature = 36.0 + rand.Float64()*5.0 // 36-41°C (above threshold)
		} else {
			temperature = 10.0 + rand.Float64()*4.0 // 10-14°C (below threshold)
		}
	}

	// Humidity with realistic variation
	humidityVariation := rand.Float64()*10.0 - 5.0 // ±5% variation
	humidity := m.baseHumidity + humidityVariation

	if isAnomaly && rand.Float64() < 0.3 {
		// Sometimes generate humidity anomaly
		if rand.Float64() < 0.5 {
			humidity = 85.0 + rand.Float64()*10.0 // 85-95% (above threshold)
		} else {
			humidity = 15.0 + rand.Float64()*10.0 // 15-25% (below threshold)
		}
	}

	// Gas quality (mostly good, sometimes moderate/poor)
	gasQuality := "good"
	if isAnomaly {
		r := rand.Float64()
		if r < 0.2 {
			gasQuality = "poor"
		} else if r < 0.5 {
			gasQuality = "moderate"
		}
	} else {
		if rand.Float64() < 0.05 {
			gasQuality = "moderate"
		}
	}

	// Flame detection (rare event)
	flameDetected := false
	if isAnomaly && rand.Float64() < 0.1 {
		flameDetected = true
	}

	// Acceleration (with gravity ~9.8 m/s² on Z-axis for stationary device)
	// Add small noise for realistic sensor readings
	accelX := (rand.Float64() - 0.5) * 0.2 // Small noise
	accelY := (rand.Float64() - 0.5) * 0.2
	accelZ := 9.8 + (rand.Float64()-0.5)*0.3 // Gravity ± noise

	if isAnomaly && rand.Float64() < 0.2 {
		// Movement/vibration anomaly
		accelX = (rand.Float64() - 0.5) * 10.0
		accelY = (rand.Float64() - 0.5) * 10.0
		accelZ = 9.8 + (rand.Float64()-0.5)*5.0
	}

	// Gyroscope (near zero for stationary device, in rad/s)
	gyroX := (rand.Float64() - 0.5) * 0.1
	gyroY := (rand.Float64() - 0.5) * 0.1
	gyroZ := (rand.Float64() - 0.5) * 0.1

	if isAnomaly && rand.Float64() < 0.15 {
		// Rotation anomaly
		gyroX = (rand.Float64() - 0.5) * 8.0
		gyroY = (rand.Float64() - 0.5) * 8.0
		gyroZ = (rand.Float64() - 0.5) * 8.0
	}

	return &models.SensorData{
		DeviceID:       m.deviceID,
		TemperatureDHT: math.Round(temperature*10) / 10,
		TemperatureMPU: 0, // Deprecated
		Humidity:       math.Round(humidity*10) / 10,
		GasQuality:     gasQuality,
		FlameDetected:  flameDetected,
		Acceleration: models.AccelerationData{
			X: math.Round(accelX*100) / 100,
			Y: math.Round(accelY*100) / 100,
			Z: math.Round(accelZ*100) / 100,
		},
		Gyroscope: models.GyroscopeData{
			X: math.Round(gyroX*100) / 100,
			Y: math.Round(gyroY*100) / 100,
			Z: math.Round(gyroZ*100) / 100,
		},
		Timestamp: now,
	}
}

func main() {
	flag.Parse()

	// Initialize logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	logger.Info("MQTT Mock Data Generator started",
		zap.String("device_id", *deviceID),
		zap.Int("rps", *rps),
		zap.Float64("anomaly_probability", *anomaly),
		zap.String("mqtt_broker", *mqttBroker),
		zap.String("mqtt_topic", *mqttTopic),
	)
	logger.Info("Press Ctrl+C to stop gracefully")

	// Initialize MQTT client (simulating ESP32/Arduino)
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s", *mqttBroker))
	opts.SetClientID(fmt.Sprintf("%s-generator", *deviceID))
	opts.SetUsername(*mqttUser)
	opts.SetPassword(*mqttPass)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.SetAutoReconnect(true)

	// Connection handler
	opts.OnConnect = func(client mqtt.Client) {
		logger.Info("Connected to MQTT broker",
			zap.String("broker", *mqttBroker))
	}

	// Connection lost handler
	opts.OnConnectionLost = func(client mqtt.Client, err error) {
		logger.Error("MQTT connection lost", zap.Error(err))
	}

	// Create and connect MQTT client
	mqttClient := mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		logger.Fatal("Failed to connect to MQTT broker", zap.Error(token.Error()))
	}
	defer mqttClient.Disconnect(250)

	// Initialize mock data generator
	mockGen := NewMockDataGenerator(*deviceID, *anomaly, logger)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Shutdown signal received, stopping generator")
		cancel()
	}()

	// Calculate interval between messages
	interval := time.Second / time.Duration(*rps)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logger.Info("Starting to generate mock data (infinite mode)",
		zap.Duration("interval", interval),
		zap.String("rate", fmt.Sprintf("%d msg/s", *rps)))

	messageCount := 0
	anomalyCount := 0
	startTime := time.Now()

	// Print stats every 60 seconds
	statsTicker := time.NewTicker(60 * time.Second)
	defer statsTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Graceful shutdown
			elapsed := time.Since(startTime)
			avgRate := float64(messageCount) / elapsed.Seconds()

			logger.Info("🛑 Shutting down gracefully...",
				zap.Int("total_messages", messageCount),
				zap.Int("anomalies_generated", anomalyCount),
				zap.Duration("total_uptime", elapsed),
				zap.Float64("avg_rate", avgRate),
			)

			// Disconnect MQTT client gracefully
			logger.Info("Disconnecting from MQTT broker...")
			mqttClient.Disconnect(250)

			logger.Info("✅ Shutdown complete. Goodbye!")
			return

		case <-ticker.C:
			// Generate sensor data
			sensorData := mockGen.GenerateSensorData()

			// Check if this is an anomaly (for statistics)
			isAnomaly := sensorData.TemperatureDHT > 35 || sensorData.TemperatureDHT < 15 ||
				sensorData.Humidity > 80 || sensorData.Humidity < 30 ||
				sensorData.GasQuality == "poor" || sensorData.GasQuality == "moderate" ||
				sensorData.FlameDetected

			if isAnomaly {
				anomalyCount++
			}

			// Convert to JSON (like ESP32 would do)
			jsonData, err := json.Marshal(sensorData)
			if err != nil {
				logger.Error("Failed to marshal sensor data", zap.Error(err))
				continue
			}

			// Publish to MQTT (simulating ESP32/Arduino)
			token := mqttClient.Publish(*mqttTopic, 0, false, jsonData)
			if token.Wait() && token.Error() != nil {
				logger.Error("Failed to publish MQTT message",
					zap.Error(token.Error()),
					zap.Int("message_count", messageCount))
			} else {
				messageCount++

				// Log every 100 messages
				if messageCount%100 == 0 {
					logger.Info("📊 MQTT messages published",
						zap.Int("count", messageCount),
						zap.Int("anomalies", anomalyCount),
						zap.Float64("rate", float64(messageCount)/time.Since(startTime).Seconds()),
					)
				}

				// Log message details in debug mode
				prettyJSON, _ := json.MarshalIndent(sensorData, "", "  ")
				logger.Debug("Published MQTT message",
					zap.String("device_id", sensorData.DeviceID),
					zap.String("topic", *mqttTopic),
					zap.Bool("is_anomaly", isAnomaly),
					zap.String("data", string(prettyJSON)))
			}

		case <-statsTicker.C:
			// Print statistics every 60 seconds
			recentRate := float64(messageCount) / time.Since(startTime).Seconds()
			anomalyRate := 0.0
			if messageCount > 0 {
				anomalyRate = float64(anomalyCount) / float64(messageCount) * 100
			}

			logger.Info("📈 Statistics",
				zap.Int("total_messages", messageCount),
				zap.Int("anomalies", anomalyCount),
				zap.Float64("anomaly_rate_percent", anomalyRate),
				zap.Float64("avg_rate_msg_per_sec", recentRate),
				zap.Duration("uptime", time.Since(startTime)),
			)
		}
	}
}
