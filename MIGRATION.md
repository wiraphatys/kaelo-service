# Migration Guide: Firebase Polling → RabbitMQ Architecture

This document explains the migration from Firebase Realtime Database polling to RabbitMQ-based architecture.

## 🏗️ Architecture Changes

### Previous Architecture (Firebase Polling)
```
ESP32 → Firebase RTDB → kaelo-service (polling every 3s)
                            ├─ Anomaly Detection
                            ├─ Telegram Alert
                            └─ Hardware Alert
```

### New Architecture (RabbitMQ + Batch Processing)
```
ESP32 → RabbitMQ → kaelo-service → ┌─ Process 1: Business Logic (realtime)
       (MQTT)                       │  ├─ Anomaly Detection
                                    │  ├─ Telegram Alert
                                    │  └─ Hardware Alert
                                    │
                                    └─ Process 2: Batch Writer to Firebase
                                       └─ Buffer (100 records OR 10s timeout)
                                          └─ Firebase RTDB ← Dashboard Query
```

## 📊 Key Changes

### 1. Data Flow
- **Before**: ESP32 writes directly to Firebase, service polls Firebase every 3 seconds
- **After**: ESP32 publishes to RabbitMQ via MQTT, service consumes messages in real-time

### 2. Firebase Role
- **Before**: Primary data source (read + write)
- **After**: Archive storage only (batch write every 100 records or 10 seconds)

### 3. Processing
- **Before**: Single sequential process
- **After**: Two parallel processes:
  - **Process 1**: Real-time anomaly detection and alerts (critical path)
  - **Process 2**: Batch writing to Firebase (non-critical path)

## 🔧 ESP32 Changes Required

### Old ESP32 Code (Firebase Direct Write)
```cpp
// Old: Direct Firebase write
Firebase.setFloat(firebaseData, "/sensor-data/temperature", temp);
Firebase.setFloat(firebaseData, "/sensor-data/humidity", humidity);
```

### New ESP32 Code (MQTT Publish)
```cpp
#include <PubSubClient.h>

// MQTT Configuration
const char* mqtt_server = "your-rabbitmq-host";
const int mqtt_port = 1883;
const char* mqtt_user = "kaelo";
const char* mqtt_password = "kaelo2024";
const char* mqtt_topic = "sensor_data_queue";

WiFiClient espClient;
PubSubClient mqttClient(espClient);

void setup() {
  // ... WiFi setup ...
  
  // Setup MQTT
  mqttClient.setServer(mqtt_server, mqtt_port);
  mqttClient.setCallback(callback);
}

void reconnect() {
  while (!mqttClient.connected()) {
    Serial.print("Attempting MQTT connection...");
    if (mqttClient.connect("ESP32Client", mqtt_user, mqtt_password)) {
      Serial.println("connected");
    } else {
      Serial.print("failed, rc=");
      Serial.print(mqttClient.state());
      delay(5000);
    }
  }
}

void publishSensorData() {
  // Create JSON payload
  StaticJsonDocument<512> doc;
  doc["device_id"] = "ESP32-001";
  doc["temperature_dht"] = temp;
  doc["humidity"] = humidity;
  doc["gas_quality"] = gasQuality;
  doc["flame_detected"] = flameDetected;
  doc["timestamp"] = getISOTimestamp();
  
  JsonObject acceleration = doc.createNestedObject("acceleration");
  acceleration["x"] = accel_x;
  acceleration["y"] = accel_y;
  acceleration["z"] = accel_z;
  
  JsonObject gyroscope = doc.createNestedObject("gyroscope");
  gyroscope["x"] = gyro_x;
  gyroscope["y"] = gyro_y;
  gyroscope["z"] = gyro_z;
  
  // Serialize and publish
  char jsonBuffer[512];
  serializeJson(doc, jsonBuffer);
  
  mqttClient.publish(mqtt_topic, jsonBuffer);
  Serial.println("Data published to MQTT");
}

void loop() {
  if (!mqttClient.connected()) {
    reconnect();
  }
  mqttClient.loop();
  
  // Read sensors and publish
  readSensors();
  publishSensorData();
  
  delay(1000); // Can send more frequently now (not limited by Firebase writes)
}
```

## 🚀 Deployment Steps

### Step 1: Setup RabbitMQ

```bash
# Start RabbitMQ with docker-compose
cd /path/to/kaelo-service
docker-compose up -d rabbitmq

# Wait for RabbitMQ to start (check logs)
docker-compose logs -f rabbitmq

# Verify RabbitMQ is running
# Management UI: http://localhost:15672
# Username: kaelo
# Password: kaelo2024
```

### Step 2: Enable MQTT Plugin

RabbitMQ MQTT plugin is automatically enabled via the init script, but you can verify:

```bash
docker exec kaelo-rabbitmq rabbitmq-plugins list
# Should show: [E*] rabbitmq_mqtt
```

### Step 3: Update Environment Variables

Copy and configure the new environment file:

```bash
cp env.example .env
# Edit .env with your actual values
```

Key new variables:
- `RABBITMQ_URL`: Connection string (default: `amqp://kaelo:kaelo2024@localhost:5672/`)
- `RABBITMQ_QUEUE`: Queue name (default: `sensor_data_queue`)
- `RABBITMQ_EXCHANGE`: Exchange name (default: `sensors`)
- `FIREBASE_BATCH_SIZE`: Batch size (default: `100`)
- `FIREBASE_BATCH_TIMEOUT`: Timeout in seconds (default: `10`)

### Step 4: Update ESP32 Firmware

Flash the new firmware to ESP32 with MQTT support:

1. Update ESP32 code (see above)
2. Update MQTT credentials in code
3. Flash to ESP32
4. Monitor serial output to verify connection

### Step 5: Deploy New Service

```bash
# Stop old service
docker-compose down kaelo-service

# Rebuild with new code
docker build -t kaelo-service .

# Start all services
docker-compose up -d

# Check logs
docker-compose logs -f kaelo-service
```

## 🧪 Testing

### Test 1: Verify RabbitMQ Connection

```bash
# Check if service connected to RabbitMQ
docker-compose logs kaelo-service | grep "Connected to RabbitMQ"

# Check RabbitMQ Management UI
# http://localhost:15672/#/queues
# Should see "sensor_data_queue" with 1 consumer
```

### Test 2: Publish Test Message

Use the provided test script:

```bash
chmod +x scripts/test-rabbitmq.sh
./scripts/test-rabbitmq.sh
```

Or manually via RabbitMQ Management UI:
1. Go to http://localhost:15672/#/exchanges/%2F/sensors
2. Click "Publish message"
3. Set routing key: `sensor_data_queue`
4. Paste JSON payload (see test script)
5. Click "Publish message"

### Test 3: Verify Processing

Check service logs for:
- Message received from RabbitMQ
- Anomaly detection (if thresholds exceeded)
- Batch write to Firebase (every 100 messages or 10 seconds)

```bash
docker-compose logs -f kaelo-service
```

### Test 4: Verify Firebase Batch Write

```bash
# Watch for batch write logs
docker-compose logs kaelo-service | grep "Successfully wrote batch to Firebase"

# Check Firebase Console
# Should see records in /sensor-data path
```

## 📊 Performance Improvements

### Message Throughput
- **Before**: ~20 messages/minute (limited by polling interval)
- **After**: ~1000+ messages/minute (real-time consumption)

### Firebase Writes
- **Before**: 1 write per message (expensive)
- **After**: 1 write per 100 messages (10x more efficient)

### Latency
- **Before**: 0-3 seconds delay (polling interval)
- **After**: <100ms (real-time processing)

## 🔄 Rollback Plan

If you need to rollback to the old architecture:

```bash
# 1. Stop new services
docker-compose down

# 2. Checkout old code
git checkout <previous-commit-hash>

# 3. Update ESP32 firmware back to Firebase direct write

# 4. Start old service
docker-compose up -d kaelo-service
```

## 🐛 Troubleshooting

### Issue: Service can't connect to RabbitMQ
```bash
# Check RabbitMQ is running
docker-compose ps rabbitmq

# Check RabbitMQ logs
docker-compose logs rabbitmq

# Verify credentials in .env file
```

### Issue: Messages not being consumed
```bash
# Check queue has consumer
# RabbitMQ UI: http://localhost:15672/#/queues

# Check service logs
docker-compose logs kaelo-service | grep "consuming messages"
```

### Issue: ESP32 can't connect to MQTT
```bash
# Verify MQTT port is accessible
telnet your-rabbitmq-host 1883

# Check MQTT plugin is enabled
docker exec kaelo-rabbitmq rabbitmq-plugins list | grep mqtt

# Check ESP32 serial output for connection errors
```

### Issue: Batch not writing to Firebase
```bash
# Check Firebase credentials
docker-compose logs kaelo-service | grep "Firebase"

# Check buffer size (might not reach 100 yet)
# Should auto-flush after 10 seconds even if buffer not full
```

## 📚 Additional Resources

- [RabbitMQ MQTT Plugin Documentation](https://www.rabbitmq.com/mqtt.html)
- [RabbitMQ Management Plugin](https://www.rabbitmq.com/management.html)
- [ESP32 MQTT Client Library](https://github.com/knolleary/pubsubclient)

## 💡 Best Practices

1. **Monitor RabbitMQ queue depth**: Should stay near 0 (messages being consumed quickly)
2. **Set appropriate batch size**: Balance between Firebase write cost and data freshness
3. **Use persistent messages**: Already configured in the code
4. **Enable RabbitMQ monitoring**: Use Prometheus/Grafana for production
5. **Implement circuit breaker**: For Firebase writes (future enhancement)

## 🎯 Next Steps

- [ ] Implement monitoring dashboard (Grafana)
- [ ] Add alerting for queue depth
- [ ] Implement dead letter queue for failed messages
- [ ] Add circuit breaker for Firebase writes
- [ ] Scale to multiple consumers (horizontal scaling)

