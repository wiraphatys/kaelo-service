package services

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"kaelo/config"
	"kaelo/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

type TelegramService struct {
	bot                 *tgbotapi.BotAPI
	chatID              int64
	config              *config.Config
	lastAlertTimes      map[string]time.Time // Track last alert time per device
	lastFlameAlertTimes map[string]time.Time // Track last flame alert time per device
	logger              *zap.Logger
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
		bot:                 bot,
		chatID:              chatID,
		config:              cfg,
		lastAlertTimes:      make(map[string]time.Time),
		lastFlameAlertTimes: make(map[string]time.Time),
		logger:              logger,
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

	// Check for flame detection - special case handling
	hasFlameDetection := ts.hasFlameDetection(anomalies)

	if hasFlameDetection {
		// For flame detection: check flame-specific throttling
		if ts.shouldThrottleFlameAlert(sensorData.DeviceID) {
			ts.logger.Debug("Throttling flame alert", zap.String("device_id", sensorData.DeviceID))
			return nil
		}
	} else {
		// For non-flame anomalies: use regular throttling
		if ts.shouldThrottleAlert(sensorData.DeviceID) {
			ts.logger.Debug("Throttling alert", zap.String("device_id", sensorData.DeviceID))
			return nil
		}
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
	if hasFlameDetection {
		ts.lastFlameAlertTimes[sensorData.DeviceID] = time.Now()
	} else {
		ts.lastAlertTimes[sensorData.DeviceID] = time.Now()
	}

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

// shouldThrottleFlameAlert checks if we should throttle flame alerts for a device (within 15 seconds)
func (ts *TelegramService) shouldThrottleFlameAlert(deviceID string) bool {
	lastFlameAlertTime, exists := ts.lastFlameAlertTimes[deviceID]
	if !exists {
		return false // No previous flame alert, don't throttle
	}

	timeSinceLastFlameAlert := time.Since(lastFlameAlertTime)
	return timeSinceLastFlameAlert < 15*time.Second
}

// hasFlameDetection checks if any of the anomalies contains flame detection
func (ts *TelegramService) hasFlameDetection(anomalies []*models.Anomaly) bool {
	for _, anomaly := range anomalies {
		if anomaly.Type == models.FlameDetected {
			return true
		}
	}
	return false
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
	sb.WriteString(fmt.Sprintf("🌡️ DHT Temperature: %.1f°C\n", sensorData.TemperatureDHT))
	sb.WriteString(fmt.Sprintf("💧 Humidity: %.1f%%\n", sensorData.Humidity))
	sb.WriteString(fmt.Sprintf("💨 Gas Quality: %s\n", sensorData.GasQuality))
	sb.WriteString(fmt.Sprintf("🔥 Flame: %t\n\n", sensorData.FlameDetected))

	// deprecated
	// sb.WriteString(fmt.Sprintf("🌡️ MPU Temperature: %.1f°C\n", sensorData.TemperatureMPU))

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
	case models.GasQualityPoor:
		return "Poor Air Quality Alert"
	case models.GasQualityModerate:
		return "Moderate Air Quality Alert"
	case models.FlameDetected:
		return "Flame Detection Alert"
	case models.AccelerationAbnormal:
		return "Abnormal Movement Alert"
	case models.GyroscopeAbnormal:
		return "Abnormal Rotation Alert"
	case models.TemperatureDifferential:
		return "Temperature Sensor Mismatch"
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

// SendUnknownPersonAlert sends alert when unknown person is detected with photo
func (ts *TelegramService) SendUnknownPersonAlert(uid string, imageBase64 string, timestamp string) error {
	// Format message
	message := fmt.Sprintf(
		"🚨 <b>UNKNOWN PERSON DETECTED</b> 🚨\n\n"+
			"👤 <b>Person ID:</b> <code>%s</code>\n"+
			"🕐 <b>Time:</b> %s\n\n"+
			"⚠️ An unrecognized person has entered the premises.\n"+
			"Please check the attached photo and take appropriate action.",
		uid,
		timestamp,
	)

	// If photo is provided, send photo with caption
	if imageBase64 != "" {
		// Clean base64 string (remove whitespace and newlines)
		cleanBase64 := strings.ReplaceAll(imageBase64, " ", "")
		cleanBase64 = strings.ReplaceAll(cleanBase64, "\n", "")
		cleanBase64 = strings.ReplaceAll(cleanBase64, "\r", "")
		cleanBase64 = strings.ReplaceAll(cleanBase64, "\t", "")

		// Decode base64 image
		imageData, err := base64.StdEncoding.DecodeString(cleanBase64)
		if err != nil {
			ts.logger.Error("Failed to decode base64 image",
				zap.Error(err),
				zap.Int("base64_length", len(imageBase64)),
				zap.String("uid", uid))

			// Send text-only message if image decode fails
			msg := tgbotapi.NewMessage(ts.chatID, message+"\n\n❌ Failed to decode image")
			msg.ParseMode = "HTML"
			ts.bot.Send(msg)

			return fmt.Errorf("error decoding image: %v", err)
		}

		ts.logger.Info("Decoded image successfully",
			zap.Int("image_size_bytes", len(imageData)),
			zap.String("uid", uid))

		// Check image size (Telegram limit is 10MB)
		const maxSize = 10 * 1024 * 1024 // 10MB
		if len(imageData) > maxSize {
			ts.logger.Warn("Image size exceeds Telegram limit",
				zap.Int("size_bytes", len(imageData)),
				zap.Int("max_bytes", maxSize),
				zap.String("uid", uid))

			// Send text-only message if image is too large
			msg := tgbotapi.NewMessage(ts.chatID, message+"\n\n❌ Image too large to send")
			msg.ParseMode = "HTML"
			ts.bot.Send(msg)

			return fmt.Errorf("image size %d bytes exceeds Telegram limit of %d bytes", len(imageData), maxSize)
		}

		// Create photo message with caption
		photoBytes := tgbotapi.FileBytes{
			Name:  fmt.Sprintf("unknown_person_%s.jpg", uid),
			Bytes: imageData,
		}

		photoMsg := tgbotapi.NewPhoto(ts.chatID, photoBytes)
		photoMsg.Caption = message
		photoMsg.ParseMode = "HTML"

		_, err = ts.bot.Send(photoMsg)
		if err != nil {
			ts.logger.Error("Failed to send photo",
				zap.Error(err),
				zap.Int("image_size", len(imageData)),
				zap.String("uid", uid))

			// Fallback: send text-only message
			msg := tgbotapi.NewMessage(ts.chatID, message+"\n\n❌ Failed to send photo")
			msg.ParseMode = "HTML"
			ts.bot.Send(msg)

			return fmt.Errorf("error sending photo: %v", err)
		}

		ts.logger.Info("Unknown person alert with photo sent successfully",
			zap.String("uid", uid),
			zap.String("timestamp", timestamp),
			zap.Int("image_size", len(imageData)))
	} else {
		// Send text-only message if no photo
		msg := tgbotapi.NewMessage(ts.chatID, message)
		msg.ParseMode = "HTML"
		msg.DisableWebPagePreview = true

		_, err := ts.bot.Send(msg)
		if err != nil {
			ts.logger.Error("Failed to send text message", zap.Error(err))
			return fmt.Errorf("error sending text message: %v", err)
		}

		ts.logger.Info("Unknown person alert (text only) sent",
			zap.String("uid", uid),
			zap.String("timestamp", timestamp))
	}

	return nil
}

// SendHealthCheckTimeoutAlert sends an alert when a device fails to send health check within timeout
func (ts *TelegramService) SendHealthCheckTimeoutAlert(deviceID string, lastSeen time.Time, timeSinceLastSeen time.Duration, lastHealthCheck *models.HealthCheckData) error {
	var sb strings.Builder

	// Header
	sb.WriteString("⚠️ <b>DEVICE HEALTH CHECK TIMEOUT</b> ⚠️\n\n")

	// Device info
	sb.WriteString(fmt.Sprintf("📱 <b>Device:</b> %s\n", deviceID))
	sb.WriteString(fmt.Sprintf("🕐 <b>Last Seen:</b> %s\n", lastSeen.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("⏱️ <b>Time Since Last Check:</b> %s\n\n", formatDuration(timeSinceLastSeen)))

	// Last known status (if available)
	if lastHealthCheck != nil {
		sb.WriteString("📊 <b>Last Known Status:</b>\n")
		sb.WriteString(fmt.Sprintf("📡 WiFi: %s\n", formatConnectionStatus(lastHealthCheck.WiFiConnected)))
		sb.WriteString(fmt.Sprintf("🔌 MQTT: %s\n", formatConnectionStatus(lastHealthCheck.MQTTConnected)))
		sb.WriteString(fmt.Sprintf("⏰ Uptime: %s\n\n", formatUptime(lastHealthCheck.UptimeMs)))

		// Sensor status
		sb.WriteString("🔧 <b>Sensors:</b>\n")
		sb.WriteString(fmt.Sprintf("  • DHT11: %s\n", formatSensorStatus(lastHealthCheck.Sensors.DHT11)))
		sb.WriteString(fmt.Sprintf("  • MPU6050: %s\n", formatSensorStatus(lastHealthCheck.Sensors.MPU6050)))
		sb.WriteString(fmt.Sprintf("  • Flame: %s\n", formatSensorStatus(lastHealthCheck.Sensors.Flame)))
		sb.WriteString(fmt.Sprintf("  • Gas: %s\n\n", formatSensorStatus(lastHealthCheck.Sensors.Gas)))
	}

	// Action required
	sb.WriteString("💡 <b>Action Required:</b>\n")
	sb.WriteString("Device may be offline or experiencing connectivity issues. Please check the device status.\n\n")

	// Status indicator
	sb.WriteString("🔴 <b>Status:</b> DEVICE TIMEOUT")

	msg := tgbotapi.NewMessage(ts.chatID, sb.String())
	msg.ParseMode = "HTML"
	msg.DisableWebPagePreview = true

	_, err := ts.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("error sending health check timeout alert: %v", err)
	}

	ts.logger.Info("Sent health check timeout alert",
		zap.String("device_id", deviceID),
		zap.Duration("time_since_last_seen", timeSinceLastSeen))

	return nil
}

// SendHealthCheckRecoveryAlert sends an alert when a device recovers from timeout
func (ts *TelegramService) SendHealthCheckRecoveryAlert(deviceID string, downDuration time.Duration) error {
	var sb strings.Builder

	// Header
	sb.WriteString("✅ <b>DEVICE RECOVERED</b> ✅\n\n")

	// Device info
	sb.WriteString(fmt.Sprintf("📱 <b>Device:</b> %s\n", deviceID))
	sb.WriteString(fmt.Sprintf("🕐 <b>Recovery Time:</b> %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("⏱️ <b>Downtime:</b> %s\n\n", formatDuration(downDuration)))

	// Status indicator
	sb.WriteString("🟢 <b>Status:</b> DEVICE ONLINE")

	msg := tgbotapi.NewMessage(ts.chatID, sb.String())
	msg.ParseMode = "HTML"
	msg.DisableWebPagePreview = true

	_, err := ts.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("error sending health check recovery alert: %v", err)
	}

	ts.logger.Info("Sent health check recovery alert",
		zap.String("device_id", deviceID),
		zap.Duration("down_duration", downDuration))

	return nil
}

// Helper functions for formatting

func formatConnectionStatus(connected bool) string {
	if connected {
		return "✅ Connected"
	}
	return "❌ Disconnected"
}

func formatSensorStatus(working bool) string {
	if working {
		return "✅ OK"
	}
	return "❌ Failed"
}

func formatUptime(uptimeMs int64) string {
	duration := time.Duration(uptimeMs) * time.Millisecond
	return formatDuration(duration)
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0f seconds", d.Seconds())
	} else if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%d min %d sec", minutes, seconds)
	} else if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%d hr %d min", hours, minutes)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%d days %d hr", days, hours)
}
