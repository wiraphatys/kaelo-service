package services

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"kaelo/config"
	"kaelo/models"
	"go.uber.org/zap"
)

type TelegramService struct {
	bot            *tgbotapi.BotAPI
	chatID         int64
	config         *config.Config
	lastAlertTimes map[string]time.Time // Track last alert time per device
	logger         *zap.Logger
}

func NewTelegramService(cfg *config.Config) (*TelegramService, error) {
	logger, _ := zap.NewProduction()
	bot, err := tgbotapi.NewBotAPI(cfg.TelegramBotToken)
	if err != nil {
		return nil, fmt.Errorf("error creating telegram bot: %v", err)
	}

	chatID, err := strconv.ParseInt(cfg.TelegramChatID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing chat ID: %v", err)
	}

	logger.Info("Telegram bot authorized", zap.String("username", bot.Self.UserName))

	ts := &TelegramService{
		bot:            bot,
		chatID:         chatID,
		config:         cfg,
		lastAlertTimes: make(map[string]time.Time),
		logger:         logger,
	}

	// Test Telegram connection with retry
	if err := ts.testConnection(); err != nil {
		logger.Error("Telegram connection test failed", zap.Error(err))
		return nil, fmt.Errorf("telegram connection test failed: %v", err)
	}

	return ts, nil
}

// testConnection tests Telegram connection with retry logic
func (ts *TelegramService) testConnection() error {
	maxRetries := 3
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		ts.logger.Info("Testing Telegram connection", zap.Int("attempt", attempt), zap.Int("max_retries", maxRetries))
		
		// Try to get bot info to test connection
		_, err := ts.bot.GetMe()
		
		if err == nil {
			ts.logger.Info("Telegram connection successful")
			return nil
		}
		
		ts.logger.Warn("Telegram connection failed",
			zap.Int("attempt", attempt),
			zap.Int("max_retries", maxRetries),
			zap.Error(err))
		
		if attempt < maxRetries {
			time.Sleep(time.Duration(attempt) * time.Second) // Exponential backoff
		}
	}
	
	return fmt.Errorf("failed to connect to Telegram after %d attempts", maxRetries)
}

// SendAnomalyAlert sends a beautifully formatted anomaly alert to Telegram with throttling
func (ts *TelegramService) SendAnomalyAlert(anomalies []*models.Anomaly, sensorData *models.SensorData) error {
	if len(anomalies) == 0 {
		return nil
	}

	// Check if we should throttle notifications for this device
	if ts.shouldThrottleAlert(sensorData.DeviceID) {
		ts.logger.Debug("Throttling alert", zap.String("device_id", sensorData.DeviceID))
		return nil
	}

	message := ts.formatAnomalyMessage(anomalies, sensorData)

	msg := tgbotapi.NewMessage(ts.chatID, message)
	msg.ParseMode = "HTML"
	msg.DisableWebPagePreview = true

	_, err := ts.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("error sending telegram message: %v", err)
	}

	// Update last alert time for this device
	ts.lastAlertTimes[sensorData.DeviceID] = time.Now()

	ts.logger.Info("Sent anomaly alert",
		zap.String("device_id", sensorData.DeviceID),
		zap.Int("anomaly_count", len(anomalies)))
	return nil
}

// shouldThrottleAlert checks if we should throttle alerts for a device (within 15 seconds)
func (ts *TelegramService) shouldThrottleAlert(deviceID string) bool {
	lastAlertTime, exists := ts.lastAlertTimes[deviceID]
	if !exists {
		return false // No previous alert, don't throttle
	}
	
	timeSinceLastAlert := time.Since(lastAlertTime)
	return timeSinceLastAlert < 15*time.Second
}

// formatAnomalyMessage creates a mobile-friendly, beautifully formatted message
func (ts *TelegramService) formatAnomalyMessage(anomalies []*models.Anomaly, sensorData *models.SensorData) string {
	var sb strings.Builder

	// Header with alert emoji
	sb.WriteString("🚨 <b>KAELO SENSOR ALERT</b> 🚨\n\n")

	// Device info
	sb.WriteString(fmt.Sprintf("📱 <b>Device:</b> %s\n", sensorData.DeviceID))
	sb.WriteString(fmt.Sprintf("🕐 <b>Time:</b> %s\n\n", sensorData.Timestamp.Format("2006-01-02 15:04:05")))

	// Current readings section
	sb.WriteString("📊 <b>Current Readings:</b>\n")
	sb.WriteString(fmt.Sprintf("🌡️ Temperature: %.1f°C\n", sensorData.Temperature))
	sb.WriteString(fmt.Sprintf("💧 Humidity: %.1f%%\n", sensorData.Humidity))
	sb.WriteString(fmt.Sprintf("💨 Dust: %.1f μg/m³\n\n", sensorData.Dust))

	// Anomalies section
	sb.WriteString("⚠️ <b>Detected Issues:</b>\n")
	for i, anomaly := range anomalies {
		sb.WriteString(fmt.Sprintf("%s %s <b>%s</b>\n", 
			anomaly.GetSeverityColor(), 
			anomaly.GetAnomalyEmoji(), 
			ts.getAnomalyTitle(anomaly)))
		
		sb.WriteString(fmt.Sprintf("   └ %s\n", anomaly.Description))
		
		if i < len(anomalies)-1 {
			sb.WriteString("\n")
		}
	}

	// Footer with action recommendation
	sb.WriteString("\n💡 <b>Recommended Action:</b>\n")
	sb.WriteString("Please check the environment and take appropriate measures to normalize the conditions.\n\n")
	
	// Status indicator
	sb.WriteString("🔴 <b>Status:</b> ATTENTION REQUIRED")

	return sb.String()
}

// getAnomalyTitle returns a user-friendly title for the anomaly
func (ts *TelegramService) getAnomalyTitle(anomaly *models.Anomaly) string {
	switch anomaly.Type {
	case models.TemperatureTooHigh:
		return "High Temperature Alert"
	case models.TemperatureTooLow:
		return "Low Temperature Alert"
	case models.HumidityTooHigh:
		return "High Humidity Alert"
	case models.HumidityTooLow:
		return "Low Humidity Alert"
	case models.DustTooHigh:
		return "High Dust Level Alert"
	default:
		return "Sensor Alert"
	}
}

// SendStatusMessage sends a general status message
func (ts *TelegramService) SendStatusMessage(message string) error {
	msg := tgbotapi.NewMessage(ts.chatID, message)
	msg.ParseMode = "HTML"
	
	_, err := ts.bot.Send(msg)
	return err
}

// SendStartupMessage sends a message when the service starts
func (ts *TelegramService) SendStartupMessage() error {
	message := "🟢 <b>KAELO Monitoring Service Started</b>\n\n" +
		"📡 Connected to Firebase Realtime Database\n" +
		"🤖 Telegram notifications active\n" +
		"👀 Monitoring sensor data for anomalies...\n\n" +
		"✅ System is ready and operational!"

	return ts.SendStatusMessage(message)
}
