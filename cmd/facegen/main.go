package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kaelo/config"
	"kaelo/models"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

var (
	imagePath   = flag.String("image", "", "Path to image file (will be converted to base64)")
	rabbitMQURL = flag.String("rabbitmq", "", "RabbitMQ URL (default from config)")
)

func main() {
	flag.Parse()

	// Initialize logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	// Use provided RabbitMQ URL or default from config
	mqttURL := cfg.RabbitMQURL
	if *rabbitMQURL != "" {
		mqttURL = *rabbitMQURL
	}

	logger.Info("Face Recognition Test Generator",
		zap.String("rabbitmq_url", mqttURL),
		zap.String("image_path", *imagePath))

	// Read and encode image if provided
	var imageBase64 string
	if *imagePath != "" {
		imageData, err := os.ReadFile(*imagePath)
		if err != nil {
			logger.Fatal("Failed to read image file", zap.Error(err))
		}
		imageBase64 = base64.StdEncoding.EncodeToString(imageData)
		logger.Info("Image loaded and encoded",
			zap.Int("size_bytes", len(imageData)),
			zap.Int("base64_length", len(imageBase64)))
	} else {
		logger.Warn("No image provided, will send without photo")
	}

	// Connect to RabbitMQ
	conn, err := amqp.Dial(mqttURL)
	if err != nil {
		logger.Fatal("Failed to connect to RabbitMQ", zap.Error(err))
	}
	defer conn.Close()

	channel, err := conn.Channel()
	if err != nil {
		logger.Fatal("Failed to open channel", zap.Error(err))
	}
	defer channel.Close()

	logger.Info("Connected to RabbitMQ successfully")

	// Generate new UUID v4 for unknown person
	personUID := uuid.New().String()

	// Create face recognition data
	faceData := models.FaceRecognitionData{
		Base64:    imageBase64,
		UID:       personUID,
		Timestamp: time.Now(),
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(faceData)
	if err != nil {
		logger.Fatal("Failed to marshal face data", zap.Error(err))
	}

	// Publish to face_recognition_queue via sensors exchange
	err = channel.Publish(
		cfg.RabbitMQExchange,     // exchange
		"face_recognition_queue", // routing key
		false,                    // mandatory
		false,                    // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         jsonData,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		},
	)
	if err != nil {
		logger.Fatal("Failed to publish message", zap.Error(err))
	}

	logger.Info("✅ Face recognition message published successfully",
		zap.String("uid", personUID),
		zap.String("exchange", cfg.RabbitMQExchange),
		zap.String("routing_key", "face_recognition_queue"),
		zap.Int("message_size", len(jsonData)))

	// Pretty print the sent data
	var prettyData interface{}
	json.Unmarshal(jsonData, &prettyData)
	prettyJSON, _ := json.MarshalIndent(prettyData, "", "  ")

	// Truncate base64 for display
	displayData := string(prettyJSON)
	if len(imageBase64) > 100 {
		displayData = fmt.Sprintf("{\n  \"base64\": \"<truncated %d chars>\",\n  \"uid\": \"%s\",\n  \"timestamp\": \"%s\"\n}",
			len(imageBase64), personUID, faceData.Timestamp.Format(time.RFC3339))
	}

	logger.Info("Sent data:\n" + displayData)

	// Wait a bit for processing
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-time.After(2 * time.Second):
		logger.Info("Message should be processed. Check Telegram for notification!")
	case <-sigChan:
		logger.Info("Interrupted")
	}
}
