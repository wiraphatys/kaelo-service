#!/bin/bash
# Quick script to run MQTT generator with proper settings

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Starting MQTT Mock Data Generator${NC}"
echo ""
echo "Configuration:"
echo "  - Broker: localhost:1883"
echo "  - Topic: sensor_data_queue"  
echo "  - Rate: 1 RPS"
echo "  - Anomaly: 30%"
echo ""

go run cmd/mqttgen/main.go \
  -broker localhost:1883 \
  -user kaelo \
  -pass kaelo2024 \
  -topic sensor_data_queue \
  -device ESP32-MOCK-001 \
  -rps 1 \
  -anomaly 0.3 \
  "$@"

