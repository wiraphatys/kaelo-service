#!/bin/bash
# Test script to publish a sample sensor data message to RabbitMQ

# Configuration
RABBITMQ_HOST="localhost"
RABBITMQ_PORT="15672"
RABBITMQ_USER="kaelo"
RABBITMQ_PASS="kaelo2024"
EXCHANGE="sensors"
ROUTING_KEY="sensor_data_queue"

# Sample sensor data JSON
SENSOR_DATA='{
  "device_id": "ESP32-TEST-001",
  "temperature_dht": 28.5,
  "temperature_mpu": 0,
  "humidity": 65.2,
  "gas_quality": "good",
  "acceleration": {
    "x": 0.1,
    "y": 0.2,
    "z": 9.8
  },
  "gyroscope": {
    "x": 0.01,
    "y": 0.02,
    "z": 0.03
  },
  "flame_detected": false,
  "timestamp": "'$(date -u +"%Y-%m-%dT%H:%M:%SZ")'"
}'

echo "Publishing test message to RabbitMQ..."
echo "Exchange: $EXCHANGE"
echo "Routing Key: $ROUTING_KEY"
echo "Payload:"
echo "$SENSOR_DATA" | jq .

# Publish message using RabbitMQ HTTP API
curl -i -u $RABBITMQ_USER:$RABBITMQ_PASS \
  -H "Content-Type: application/json" \
  -X POST "http://$RABBITMQ_HOST:$RABBITMQ_PORT/api/exchanges/%2F/$EXCHANGE/publish" \
  -d '{
    "properties": {
      "delivery_mode": 2,
      "content_type": "application/json"
    },
    "routing_key": "'$ROUTING_KEY'",
    "payload": "'"$(echo $SENSOR_DATA | jq -c .)"'",
    "payload_encoding": "string"
  }'

echo ""
echo "Message published!"
echo ""
echo "To check if message was received, check the RabbitMQ Management UI:"
echo "http://$RABBITMQ_HOST:$RABBITMQ_PORT"
echo "Username: $RABBITMQ_USER"
echo "Password: $RABBITMQ_PASS"

