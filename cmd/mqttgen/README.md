# MQTT Mock Data Generator

Generate realistic mock sensor data and publish to RabbitMQ for testing.

## Features

- ✅ Realistic sensor data with natural variations
- ✅ Configurable anomaly probability
- ✅ Adjustable message rate (RPS)
- ✅ Duration limit or infinite run
- ✅ Multiple device simulation
- ✅ Graceful shutdown

## Usage

### Basic Usage

```bash
# Run with default settings (1 RPS)
go run cmd/mqttgen/main.go

# Or build and run
go build -o mqttgen cmd/mqttgen/main.go
./mqttgen
```

### Advanced Options

```bash
# 10 messages per second
go run cmd/mqttgen/main.go -rps 10

# Custom device ID
go run cmd/mqttgen/main.go -device ESP32-TEST-001

# Run for 60 seconds
go run cmd/mqttgen/main.go -duration 60

# Higher anomaly rate (30%)
go run cmd/mqttgen/main.go -anomaly 0.3

# Combine options
go run cmd/mqttgen/main.go -rps 5 -device ESP32-LAB-001 -duration 120 -anomaly 0.2
```

## Command Line Options

| Flag | Default | Description |
|------|---------|-------------|
| `-rps` | 1 | Messages per second (1-1000) |
| `-device` | ESP32-MOCK-001 | Device ID for generated data |
| `-duration` | 0 | Duration in seconds (0 = infinite) |
| `-anomaly` | 0.1 | Probability of anomaly (0.0-1.0) |

## Generated Data

### Normal Data Example

```json
{
  "device_id": "ESP32-MOCK-001",
  "temperature_dht": 27.3,
  "temperature_mpu": 0,
  "humidity": 62.5,
  "gas_quality": "good",
  "flame_detected": false,
  "acceleration": {
    "x": 0.05,
    "y": -0.03,
    "z": 9.82
  },
  "gyroscope": {
    "x": 0.01,
    "y": -0.02,
    "z": 0.00
  },
  "timestamp": "2024-01-15T14:30:25Z"
}
```

### Anomaly Data Example

```json
{
  "device_id": "ESP32-MOCK-001",
  "temperature_dht": 38.7,
  "humidity": 88.2,
  "gas_quality": "poor",
  "flame_detected": true,
  "acceleration": {
    "x": 5.23,
    "y": -3.45,
    "z": 12.10
  },
  "gyroscope": {
    "x": 3.21,
    "y": -2.15,
    "z": 1.34
  },
  "timestamp": "2024-01-15T14:30:26Z"
}
```

## Realistic Sensor Ranges

### Temperature (DHT22)
- **Normal**: 25-29°C (±2°C variation)
- **High Anomaly**: 36-41°C (triggers alert)
- **Low Anomaly**: 10-14°C (triggers alert)

### Humidity (DHT22)
- **Normal**: 55-65% (±5% variation)
- **High Anomaly**: 85-95% (triggers alert)
- **Low Anomaly**: 15-25% (triggers alert)

### Gas Quality (MQ-135)
- **Normal**: "good" (95% of time)
- **Moderate**: "moderate" (4% of time)
- **Poor**: "poor" (1% of time, usually with anomaly)

### Flame Detection
- **Normal**: false (99.9% of time)
- **Anomaly**: true (0.1% of time, critical alert)

### Acceleration (MPU6050)
- **Normal X/Y**: ±0.1 m/s² (small noise)
- **Normal Z**: 9.8 m/s² (gravity)
- **Anomaly**: ±10 m/s² (vibration/movement)

### Gyroscope (MPU6050)
- **Normal**: ±0.05 rad/s (nearly stationary)
- **Anomaly**: ±4.0 rad/s (rotation)

## Examples

### Test Anomaly Detection (10 messages)
```bash
go run cmd/mqttgen/main.go -rps 1 -duration 10 -anomaly 0.5
```

### Load Testing (High Throughput)
```bash
go run cmd/mqttgen/main.go -rps 100 -duration 60
```

### Simulate Multiple Devices
```bash
# Terminal 1
go run cmd/mqttgen/main.go -device ESP32-DEVICE-001 -rps 1

# Terminal 2
go run cmd/mqttgen/main.go -device ESP32-DEVICE-002 -rps 1

# Terminal 3
go run cmd/mqttgen/main.go -device ESP32-DEVICE-003 -rps 1
```

### Continuous Testing (24 hours)
```bash
go run cmd/mqttgen/main.go -rps 1 -duration 86400
```

## Monitoring

### View Generated Data
```bash
# Watch kaelo-service logs
docker-compose logs -f kaelo-service

# Filter for anomalies
docker-compose logs kaelo-service | grep "Anomalies detected"

# Check RabbitMQ queue
open http://localhost:15672/#/queues
```

### RabbitMQ Statistics
- **Management UI**: http://localhost:15672
- **Queue**: sensor_data_queue
- **Message rate**: Should match RPS setting
- **Consumers**: Should show 1 (kaelo-service)

## Troubleshooting

### Cannot Connect to RabbitMQ
```bash
# Check RabbitMQ is running
docker-compose ps rabbitmq

# Check connection string in .env
cat .env | grep RABBITMQ_URL
```

### No Messages in Queue
```bash
# Check service logs
go run cmd/mqttgen/main.go -rps 1

# Verify in RabbitMQ UI
open http://localhost:15672/#/queues/%2F/sensor_data_queue
```

### Rate Limiting
```bash
# If messages not publishing fast enough, check:
# 1. Network latency
# 2. RabbitMQ load
# 3. System resources

# Use lower RPS for testing
go run cmd/mqttgen/main.go -rps 1
```

## Performance

| RPS | CPU Usage | Memory | Network |
|-----|-----------|--------|---------|
| 1 | <1% | ~20MB | <1KB/s |
| 10 | ~2% | ~25MB | ~10KB/s |
| 100 | ~10% | ~30MB | ~100KB/s |
| 1000 | ~30% | ~50MB | ~1MB/s |

## Integration with Testing

### Automated Testing Script
```bash
#!/bin/bash
# test-scenario.sh

echo "Testing normal conditions..."
go run cmd/mqttgen/main.go -rps 1 -duration 30 -anomaly 0.0

echo "Testing high anomaly rate..."
go run cmd/mqttgen/main.go -rps 1 -duration 30 -anomaly 0.5

echo "Testing high throughput..."
go run cmd/mqttgen/main.go -rps 50 -duration 10 -anomaly 0.1

echo "Testing complete!"
```

## Best Practices

1. **Start with low RPS** (1-10) to verify data format
2. **Monitor Telegram alerts** for anomaly detection
3. **Check Firebase** for batch writes
4. **Use duration limit** for automated tests
5. **Graceful shutdown** with Ctrl+C for accurate statistics

## Tips

- Use `-anomaly 0.0` for testing normal conditions only
- Use `-anomaly 0.5` for testing alert systems
- Use `-rps 1` for easy log monitoring
- Use `-duration` for CI/CD automated tests
- Multiple devices simulate real IoT deployment

---

**Happy Testing!** 🚀

For more information, see the main [README.md](../../README-NEW.md)

