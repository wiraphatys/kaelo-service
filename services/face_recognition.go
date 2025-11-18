package services

import (
	"context"

	"kaelo/models"

	"go.uber.org/zap"
)

// FaceRecognitionService handles face recognition processing
type FaceRecognitionService struct {
	logger          *zap.Logger
	telegramService *TelegramService
}

// NewFaceRecognitionService creates a new face recognition service
func NewFaceRecognitionService(telegramService *TelegramService, logger *zap.Logger) *FaceRecognitionService {
	return &FaceRecognitionService{
		telegramService: telegramService,
		logger:          logger,
	}
}

// Start starts processing face recognition messages from the channel
func (f *FaceRecognitionService) Start(ctx context.Context, faceDataChan <-chan *models.FaceRecognitionData) {
	f.logger.Info("Starting face recognition processor")

	for {
		select {
		case <-ctx.Done():
			f.logger.Info("Stopping face recognition processor")
			return

		case faceData, ok := <-faceDataChan:
			if !ok {
				f.logger.Info("Face recognition channel closed")
				return
			}

			// Process the face recognition data
			f.processFaceData(faceData)
		}
	}
}

// processFaceData processes face recognition data and sends alerts
func (f *FaceRecognitionService) processFaceData(faceData *models.FaceRecognitionData) {
	f.logger.Info("Processing face recognition data",
		zap.String("uid", faceData.UID),
		zap.Time("timestamp", faceData.Timestamp),
		zap.Bool("has_image", faceData.Base64 != ""))

	// Format timestamp for display
	timestampStr := faceData.Timestamp.Format("2006-01-02 15:04:05")

	// Send Telegram notification with photo
	if err := f.telegramService.SendUnknownPersonAlert(faceData.UID, faceData.Base64, timestampStr); err != nil {
		f.logger.Error("Failed to send unknown person alert",
			zap.String("uid", faceData.UID),
			zap.Error(err))
		return
	}

	f.logger.Info("Unknown person alert sent successfully",
		zap.String("uid", faceData.UID))
}
