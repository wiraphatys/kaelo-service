package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	FirebaseDbUrl              string
	FirebaseServiceAccountJSON string
	TelegramBotToken           string
	TelegramChatID             string
	HardwareAlertURL           string
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
}

func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	config := &Config{
		FirebaseDbUrl:              getEnv("FIREBASE_DB_URL", ""),
		FirebaseServiceAccountJSON: getEnv("FIREBASE_SERVICE_ACCOUNT_JSON", ""),
		TelegramBotToken:           getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:             getEnv("TELEGRAM_CHAT_ID", ""),
		HardwareAlertURL:           getEnv("HARDWARE_ALERT_URL", ""),
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
