package services

import (
	"context"
	"sync"
	"time"

	"kaelo/config"
	"kaelo/models"

	"go.uber.org/zap"
)

// BatchWriterService handles batching sensor data and writing to Firebase
type BatchWriterService struct {
	config          *config.Config
	firebaseService *FirebaseService
	logger          *zap.Logger
	buffer          []*models.SensorData
	bufferMutex     sync.Mutex
	flushTimer      *time.Timer
	maxBatchSize    int
	batchTimeout    time.Duration
	shutdownChan    chan bool
}

// NewBatchWriterService creates a new batch writer service
func NewBatchWriterService(cfg *config.Config, firebaseService *FirebaseService, logger *zap.Logger) *BatchWriterService {
	return &BatchWriterService{
		config:          cfg,
		firebaseService: firebaseService,
		logger:          logger,
		buffer:          make([]*models.SensorData, 0, cfg.FirebaseBatchSize),
		maxBatchSize:    cfg.FirebaseBatchSize,
		batchTimeout:    time.Duration(cfg.FirebaseBatchTimeout) * time.Second,
		shutdownChan:    make(chan bool, 1),
	}
}

// Start begins the batch writer service
func (bw *BatchWriterService) Start(ctx context.Context, sensorDataChan <-chan *models.SensorData) {
	bw.logger.Info("Starting batch writer service",
		zap.Int("max_batch_size", bw.maxBatchSize),
		zap.Duration("batch_timeout", bw.batchTimeout))

	// Initialize flush timer
	bw.flushTimer = time.NewTimer(bw.batchTimeout)

	for {
		select {
		case <-ctx.Done():
			bw.logger.Info("Batch writer received shutdown signal")
			bw.flushBuffer(ctx)
			bw.shutdownChan <- true
			return

		case sensorData, ok := <-sensorDataChan:
			if !ok {
				bw.logger.Warn("Sensor data channel closed")
				bw.flushBuffer(ctx)
				return
			}

			// Add to buffer
			bw.bufferMutex.Lock()
			bw.buffer = append(bw.buffer, sensorData)
			currentSize := len(bw.buffer)
			bw.bufferMutex.Unlock()

			bw.logger.Debug("Added sensor data to buffer",
				zap.String("device_id", sensorData.DeviceID),
				zap.Int("buffer_size", currentSize),
				zap.Int("max_batch_size", bw.maxBatchSize))

			// Check if buffer is full
			if currentSize >= bw.maxBatchSize {
				bw.logger.Info("Buffer full, flushing to Firebase",
					zap.Int("buffer_size", currentSize))

				// Stop and reset timer
				if !bw.flushTimer.Stop() {
					// Drain the timer channel if it hasn't been drained
					select {
					case <-bw.flushTimer.C:
					default:
					}
				}

				bw.flushBuffer(ctx)
				bw.flushTimer.Reset(bw.batchTimeout)
			}

		case <-bw.flushTimer.C:
			// Timeout reached, flush buffer
			bw.bufferMutex.Lock()
			currentSize := len(bw.buffer)
			bw.bufferMutex.Unlock()

			if currentSize > 0 {
				bw.logger.Info("Batch timeout reached, flushing to Firebase",
					zap.Int("buffer_size", currentSize))
				bw.flushBuffer(ctx)
			}

			// Reset timer
			bw.flushTimer.Reset(bw.batchTimeout)
		}
	}
}

// flushBuffer writes the current buffer to Firebase and clears it
func (bw *BatchWriterService) flushBuffer(ctx context.Context) {
	bw.bufferMutex.Lock()

	if len(bw.buffer) == 0 {
		bw.bufferMutex.Unlock()
		return
	}

	// Copy buffer for writing (to avoid holding lock during write)
	batch := make([]*models.SensorData, len(bw.buffer))
	copy(batch, bw.buffer)

	// Clear buffer
	bw.buffer = bw.buffer[:0]

	bw.bufferMutex.Unlock()

	// Write batch to Firebase with retry
	maxRetries := 3
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err = bw.firebaseService.WriteBatch(ctx, batch)
		if err == nil {
			bw.logger.Info("Successfully flushed batch to Firebase",
				zap.Int("batch_size", len(batch)))
			return
		}

		bw.logger.Error("Failed to flush batch to Firebase",
			zap.Int("attempt", attempt),
			zap.Int("max_retries", maxRetries),
			zap.Int("batch_size", len(batch)),
			zap.Error(err))

		// Exponential backoff
		if attempt < maxRetries {
			backoff := time.Duration(attempt) * time.Second
			time.Sleep(backoff)
		}
	}

	// If all retries failed, log error (data will be lost)
	bw.logger.Error("Failed to flush batch after all retries, data lost",
		zap.Int("batch_size", len(batch)),
		zap.Error(err))
}

// WaitForShutdown waits for the batch writer to complete shutdown
func (bw *BatchWriterService) WaitForShutdown(timeout time.Duration) bool {
	select {
	case <-bw.shutdownChan:
		return true
	case <-time.After(timeout):
		return false
	}
}

// GetBufferSize returns the current buffer size (for monitoring)
func (bw *BatchWriterService) GetBufferSize() int {
	bw.bufferMutex.Lock()
	defer bw.bufferMutex.Unlock()
	return len(bw.buffer)
}
