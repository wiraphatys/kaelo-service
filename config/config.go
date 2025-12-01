package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	// RabbitMQ Configuration
	RabbitMQURL      string
	RabbitMQQueue    string
	RabbitMQExchange string

	// Firebase Configuration
	FirebaseDbUrl              string
	FirebaseServiceAccountJSON string
	FirebaseBatchSize          int
	FirebaseBatchTimeout       int // in seconds

	// Telegram Configuration
	TelegramBotToken string
	TelegramChatID   string

	// Hardware Alert Configuration
	HardwareAlertURL string

	// Thresholds for anomaly detection
	TemperatureMin float64
	TemperatureMax float64
	HumidityMin    float64
	HumidityMax    float64
	DustMax        float64
	FlameThreshold float64
	LightMin       float64
	LightMax       float64
	GasMax         float64

	// Health Check Configuration
	HealthCheckQueue   string
	HealthCheckTimeout int // in seconds
}

func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	config := &Config{
		// RabbitMQ Configuration
		RabbitMQURL:      getEnv("RABBITMQ_URL", "amqp://kaelo:kaelo2024@172.20.10.12:5672/"),
		RabbitMQQueue:    getEnv("RABBITMQ_QUEUE", "sensor_data_queue"),
		RabbitMQExchange: getEnv("RABBITMQ_EXCHANGE", "sensors"),

		// Firebase Configuration
		FirebaseDbUrl:              getEnv("FIREBASE_DB_URL", ""),
		FirebaseServiceAccountJSON: getEnv("FIREBASE_SERVICE_ACCOUNT_JSON", ""),
		FirebaseBatchSize:          getEnvInt("FIREBASE_BATCH_SIZE", 100),
		FirebaseBatchTimeout:       getEnvInt("FIREBASE_BATCH_TIMEOUT", 10),

		// Telegram Configuration
		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:   getEnv("TELEGRAM_CHAT_ID", ""),

		// Hardware Alert Configuration
		HardwareAlertURL: getEnv("HARDWARE_ALERT_URL", ""),

		// Default thresholds - can be overridden by env vars
		TemperatureMin: getEnvFloat("TEMPERATURE_MIN", 15.0),
		TemperatureMax: getEnvFloat("TEMPERATURE_MAX", 35.0),
		HumidityMin:    getEnvFloat("HUMIDITY_MIN", 30.0),
		HumidityMax:    getEnvFloat("HUMIDITY_MAX", 80.0),
		DustMax:        getEnvFloat("DUST_MAX", 50.0),
		FlameThreshold: getEnvFloat("FLAME_THRESHOLD", 500.0),
		LightMin:       getEnvFloat("LIGHT_MIN", 100.0),
		LightMax:       getEnvFloat("LIGHT_MAX", 800.0),
		GasMax:         getEnvFloat("GAS_MAX", 400.0),

		// Health Check Configuration
		HealthCheckQueue:   getEnv("HEALTH_CHECK_QUEUE", "health_check_queue"),
		HealthCheckTimeout: getEnvInt("HEALTH_CHECK_TIMEOUT", 60),
	}

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		// Simple conversion - in production you might want better error handling
		if f, err := parseFloat(value); err == nil {
			return f
		}
	}
	return defaultValue
}

func parseFloat(s string) (float64, error) {
	// Simple float parsing
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		// Simple conversion - in production you might want better error handling
		if i, err := parseInt(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func parseInt(s string) (int, error) {
	// Simple int parsing
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}
