package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"kaelo/config"
	"kaelo/models"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// RabbitMQService handles RabbitMQ connection and message consumption
type RabbitMQService struct {
	config    *config.Config
	conn      *amqp.Connection
	channel   *amqp.Channel
	logger    *zap.Logger
	reconnect chan bool
	isClosing bool
}

// NewRabbitMQService creates a new RabbitMQ service instance
func NewRabbitMQService(cfg *config.Config, logger *zap.Logger) (*RabbitMQService, error) {
	service := &RabbitMQService{
		config:    cfg,
		logger:    logger,
		reconnect: make(chan bool),
		isClosing: false,
	}

	if err := service.connect(); err != nil {
		return nil, err
	}

	return service, nil
}

// connect establishes connection to RabbitMQ and declares exchange and queue
func (r *RabbitMQService) connect() error {
	var err error

	r.logger.Info("Connecting to RabbitMQ", zap.String("url", r.config.RabbitMQURL))

	// Connect to RabbitMQ with retry
	maxRetries := 5
	for attempt := 1; attempt <= maxRetries; attempt++ {
		r.conn, err = amqp.Dial(r.config.RabbitMQURL)
		if err == nil {
			break
		}

		r.logger.Warn("Failed to connect to RabbitMQ",
			zap.Int("attempt", attempt),
			zap.Int("max_retries", maxRetries),
			zap.Error(err))

		if attempt < maxRetries {
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ after %d attempts: %w", maxRetries, err)
	}

	r.logger.Info("Connected to RabbitMQ successfully")

	// Create channel
	r.channel, err = r.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}

	// Set QoS (prefetch count)
	err = r.channel.Qos(
		10,    // prefetch count - process 10 messages at a time
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	// Declare exchange (direct exchange for now, can be changed to topic/fanout)
	err = r.channel.ExchangeDeclare(
		r.config.RabbitMQExchange, // name
		"direct",                  // type - direct exchange
		true,                      // durable
		false,                     // auto-deleted
		false,                     // internal
		false,                     // no-wait
		nil,                       // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	r.logger.Info("Exchange declared", zap.String("exchange", r.config.RabbitMQExchange))

	// Declare queue
	queue, err := r.channel.QueueDeclare(
		r.config.RabbitMQQueue, // name
		true,                   // durable
		false,                  // delete when unused
		false,                  // exclusive
		false,                  // no-wait
		nil,                    // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	r.logger.Info("Queue declared", zap.String("queue", queue.Name))

	// Bind queue to exchange with routing key
	err = r.channel.QueueBind(
		queue.Name,                // queue name
		r.config.RabbitMQQueue,    // routing key (same as queue name for simplicity)
		r.config.RabbitMQExchange, // exchange
		false,                     // no-wait
		nil,                       // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	r.logger.Info("Queue bound to exchange",
		zap.String("queue", queue.Name),
		zap.String("exchange", r.config.RabbitMQExchange),
		zap.String("routing_key", r.config.RabbitMQQueue))

	// Bind queue to amq.topic (for MQTT messages)
	err = r.channel.QueueBind(
		queue.Name,             // queue name
		r.config.RabbitMQQueue, // routing key (MQTT topic)
		"amq.topic",            // MQTT default exchange
		false,                  // no-wait
		nil,                    // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue to MQTT exchange: %w", err)
	}

	r.logger.Info("Queue bound to MQTT exchange",
		zap.String("queue", queue.Name),
		zap.String("exchange", "amq.topic"),
		zap.String("routing_key", r.config.RabbitMQQueue))

	// Setup connection close notification
	go r.handleReconnect()

	return nil
}

// handleReconnect handles automatic reconnection when connection is lost
func (r *RabbitMQService) handleReconnect() {
	for {
		closeErr := <-r.conn.NotifyClose(make(chan *amqp.Error))
		if r.isClosing {
			r.logger.Info("RabbitMQ connection closed gracefully")
			return
		}

		r.logger.Error("RabbitMQ connection lost", zap.Error(closeErr))

		// Attempt to reconnect
		for {
			r.logger.Info("Attempting to reconnect to RabbitMQ...")
			err := r.connect()
			if err == nil {
				r.logger.Info("Successfully reconnected to RabbitMQ")
				r.reconnect <- true
				break
			}

			r.logger.Error("Failed to reconnect", zap.Error(err))
			time.Sleep(5 * time.Second)
		}
	}
}

// Consume starts consuming messages from RabbitMQ queue
func (r *RabbitMQService) Consume(ctx context.Context, sensorDataChan chan<- *models.SensorData) error {
	for {
		msgs, err := r.channel.Consume(
			r.config.RabbitMQQueue, // queue
			"kaelo-service",        // consumer tag
			false,                  // auto-ack (false = manual ack)
			false,                  // exclusive
			false,                  // no-local
			false,                  // no-wait
			nil,                    // args
		)
		if err != nil {
			return fmt.Errorf("failed to register consumer: %w", err)
		}

		r.logger.Info("Started consuming messages from RabbitMQ",
			zap.String("queue", r.config.RabbitMQQueue))

	consumeLoop:
		for {
			select {
			case <-ctx.Done():
				r.logger.Info("Stopping RabbitMQ consumer")
				return nil

			case <-r.reconnect:
				r.logger.Info("Reconnection detected, restarting consumer")
				break consumeLoop

			case msg, ok := <-msgs:
				if !ok {
					r.logger.Warn("Message channel closed")
					time.Sleep(1 * time.Second)
					break consumeLoop
				}

				// Process message
				if err := r.processMessage(msg, sensorDataChan); err != nil {
					r.logger.Error("Failed to process message",
						zap.Error(err),
						zap.String("message_id", msg.MessageId))

					// Negative acknowledgment - requeue the message
					msg.Nack(false, true)
				} else {
					// Acknowledge message
					msg.Ack(false)
				}
			}
		}
	}
}

// processMessage parses and forwards sensor data to the channel
func (r *RabbitMQService) processMessage(msg amqp.Delivery, sensorDataChan chan<- *models.SensorData) error {
	// Parse JSON message
	var sensorData models.SensorData
	if err := json.Unmarshal(msg.Body, &sensorData); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Validate sensor data
	if sensorData.DeviceID == "" {
		return fmt.Errorf("invalid sensor data: missing device_id")
	}

	// Set timestamp if not provided
	if sensorData.Timestamp.IsZero() {
		sensorData.Timestamp = time.Now()
	}

	r.logger.Debug("Received sensor data from RabbitMQ",
		zap.String("device_id", sensorData.DeviceID),
		zap.Float64("temperature_dht", sensorData.TemperatureDHT),
		zap.Float64("humidity", sensorData.Humidity),
		zap.String("gas_quality", sensorData.GasQuality),
		zap.Bool("flame_detected", sensorData.FlameDetected),
		zap.Time("timestamp", sensorData.Timestamp))

	// Send to processing channel (non-blocking with timeout)
	select {
	case sensorDataChan <- &sensorData:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout sending to processing channel")
	}
}

// Close gracefully closes RabbitMQ connection
func (r *RabbitMQService) Close() error {
	r.isClosing = true

	r.logger.Info("Closing RabbitMQ connection")

	if r.channel != nil {
		if err := r.channel.Close(); err != nil {
			r.logger.Error("Error closing channel", zap.Error(err))
		}
	}

	if r.conn != nil {
		if err := r.conn.Close(); err != nil {
			r.logger.Error("Error closing connection", zap.Error(err))
			return err
		}
	}

	r.logger.Info("RabbitMQ connection closed")
	return nil
}

// Publish publishes a message to RabbitMQ (useful for testing)
func (r *RabbitMQService) Publish(sensorData *models.SensorData) error {
	body, err := json.Marshal(sensorData)
	if err != nil {
		return fmt.Errorf("failed to marshal sensor data: %w", err)
	}

	err = r.channel.Publish(
		r.config.RabbitMQExchange, // exchange
		r.config.RabbitMQQueue,    // routing key
		false,                     // mandatory
		false,                     // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // persistent message
			Timestamp:    time.Now(),
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	r.logger.Debug("Published sensor data to RabbitMQ",
		zap.String("device_id", sensorData.DeviceID))

	return nil
}
