package services

import (
	"context"
	"sync"
	"time"

	"kaelo/config"
	"kaelo/models"

	"go.uber.org/zap"
)

// HealthCheckService monitors device health checks and sends alerts for timeouts
type HealthCheckService struct {
	config          *config.Config
	telegramService *TelegramService
	logger          *zap.Logger
	devices         map[string]*models.DeviceHealth
	mu              sync.RWMutex
}

// NewHealthCheckService creates a new health check monitoring service
func NewHealthCheckService(cfg *config.Config, telegram *TelegramService, logger *zap.Logger) *HealthCheckService {
	return &HealthCheckService{
		config:          cfg,
		telegramService: telegram,
		logger:          logger,
		devices:         make(map[string]*models.DeviceHealth),
	}
}

// Start begins the health check monitoring process
func (h *HealthCheckService) Start(ctx context.Context, healthCheckChan <-chan *models.HealthCheckData) {
	h.logger.Info("Starting health check monitoring service",
		zap.String("queue", h.config.HealthCheckQueue),
		zap.Int("timeout_seconds", h.config.HealthCheckTimeout))

	// Start the timeout checker goroutine
	go h.runTimeoutChecker(ctx)

	// Process incoming health checks
	for {
		select {
		case <-ctx.Done():
			h.logger.Info("Health check monitoring service stopped")
			return
		case healthCheck, ok := <-healthCheckChan:
			if !ok {
				h.logger.Info("Health check channel closed")
				return
			}
			h.updateHealthCheck(healthCheck)
		}
	}
}

// updateHealthCheck updates the health status for a device
func (h *HealthCheckService) updateHealthCheck(data *models.HealthCheckData) {
	h.mu.Lock()
	defer h.mu.Unlock()

	deviceID := data.DeviceID
	now := time.Now()

	// Get or create device health record
	device, exists := h.devices[deviceID]
	if !exists {
		device = &models.DeviceHealth{
			DeviceID: deviceID,
			Status:   models.DeviceHealthy,
		}
		h.devices[deviceID] = device
		h.logger.Info("New device registered for health monitoring",
			zap.String("device_id", deviceID))
	}

	// Check if device was previously in timeout state
	wasTimeout := device.Status == models.DeviceTimeout

	// Update device health
	device.LastHealthCheck = data
	device.LastSeen = now
	device.Status = models.DeviceHealthy

	h.logger.Debug("Health check received",
		zap.String("device_id", deviceID),
		zap.Bool("wifi_connected", data.WiFiConnected),
		zap.Bool("mqtt_connected", data.MQTTConnected),
		zap.Int64("uptime_ms", data.UptimeMs),
		zap.Any("sensors", data.Sensors))

	// If device recovered from timeout, send recovery alert
	if wasTimeout {
		downDuration := now.Sub(device.TimeoutAt)
		h.logger.Info("Device recovered from timeout",
			zap.String("device_id", deviceID),
			zap.Duration("down_duration", downDuration))

		if err := h.telegramService.SendHealthCheckRecoveryAlert(deviceID, downDuration); err != nil {
			h.logger.Error("Failed to send recovery alert",
				zap.String("device_id", deviceID),
				zap.Error(err))
		}
	}
}

// runTimeoutChecker periodically checks for device timeouts
func (h *HealthCheckService) runTimeoutChecker(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	h.logger.Info("Health check timeout checker started")

	for {
		select {
		case <-ctx.Done():
			h.logger.Info("Health check timeout checker stopped")
			return
		case <-ticker.C:
			h.checkTimeouts()
		}
	}
}

// checkTimeouts checks all devices for timeout conditions
func (h *HealthCheckService) checkTimeouts() {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	timeoutDuration := time.Duration(h.config.HealthCheckTimeout) * time.Second

	for deviceID, device := range h.devices {
		// Skip if already in timeout state
		if device.Status == models.DeviceTimeout {
			continue
		}

		// Check if device has timed out
		timeSinceLastSeen := now.Sub(device.LastSeen)
		if timeSinceLastSeen > timeoutDuration {
			h.logger.Warn("Device health check timeout detected",
				zap.String("device_id", deviceID),
				zap.Time("last_seen", device.LastSeen),
				zap.Duration("time_since_last_seen", timeSinceLastSeen))

			// Update device status
			device.Status = models.DeviceTimeout
			device.TimeoutAt = now

			// Send timeout alert
			if err := h.telegramService.SendHealthCheckTimeoutAlert(
				deviceID,
				device.LastSeen,
				timeSinceLastSeen,
				device.LastHealthCheck,
			); err != nil {
				h.logger.Error("Failed to send timeout alert",
					zap.String("device_id", deviceID),
					zap.Error(err))
			}
		}
	}
}

// GetDeviceHealth returns the current health status of a device (for testing/debugging)
func (h *HealthCheckService) GetDeviceHealth(deviceID string) (*models.DeviceHealth, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	device, exists := h.devices[deviceID]
	return device, exists
}
