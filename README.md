# KAELO IoT Monitoring Service

A Go-based backend service for monitoring ESP32 sensor data through Firebase Realtime Database and sending Telegram notifications when anomalies are detected.

## 🏗️ System Architecture

```
ESP32 (4 sensors) → Firebase Realtime DB ← Core Backend (this service)
                                              ↓
                                         Telegram Bot
```

## 📊 Monitored Sensors

- **Temperature** (°C)
- **Humidity** (%)
- **Dust** (μg/m³)

## 🚀 Features

- Real-time Firebase Realtime Database subscription
- Configurable anomaly detection thresholds
- Beautiful mobile-friendly Telegram notifications
- Docker containerization with multi-arch support
- Automated CI/CD with GitHub Actions
- Graceful shutdown handling
- Environment-based configuration

## 📋 Prerequisites

- Go 1.23.4 or later
- Firebase project with Realtime Database
- Telegram Bot Token and Chat ID
- Docker (for containerization)

## ⚙️ Configuration

1. Copy the example environment file:
```bash
cp .env.example .env
```

2. Configure your environment variables:

### Firebase Setup
- Create a Firebase project
- Enable Realtime Database
- Generate a service account key
- Copy the entire JSON content as a single line string for `FIREBASE_SERVICE_ACCOUNT_JSON`

### Telegram Setup
- Create a bot via [@BotFather](https://t.me/botfather)
- Get your chat ID (you can use [@userinfobot](https://t.me/userinfobot))

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `FIREBASE_DB_URL` | Firebase Realtime Database URL | Required |
| `FIREBASE_SERVICE_ACCOUNT_JSON` | Service account JSON as string | Required |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token | Required |
| `TELEGRAM_CHAT_ID` | Telegram chat ID | Required |
| `TEMPERATURE_MIN` | Minimum temperature threshold | 15.0 |
| `TEMPERATURE_MAX` | Maximum temperature threshold | 35.0 |
| `HUMIDITY_MIN` | Minimum humidity threshold | 30.0 |
| `HUMIDITY_MAX` | Maximum humidity threshold | 80.0 |
| `DUST_MAX` | Maximum dust threshold | 50.0 |

## 🏃‍♂️ Running the Service

### Local Development

1. Install dependencies:
```bash
go mod download
```

2. Run the service:
```bash
go run main.go
```

### Docker

1. Build and run with Docker Compose:
```bash
docker-compose up -d
```

2. Or build and run manually:
```bash
docker build -t kaelo-service .
docker run --env-file .env kaelo-service
```

### Raspberry Pi Deployment

1. Pull the image from GitHub Container Registry:
```bash
docker pull ghcr.io/yourusername/kaelo-service:latest
```

2. Run on Raspberry Pi:
```bash
docker run -d \
  --name kaelo-monitoring \
  --restart unless-stopped \
  --env-file .env \
  ghcr.io/yourusername/kaelo-service:latest
```

## 📱 Telegram Notifications

The service sends beautifully formatted notifications with:

- 🚨 Alert headers with emojis
- 📱 Device information
- 📊 Current sensor readings
- ⚠️ Detailed anomaly descriptions
- 💡 Recommended actions
- 🔴 Status indicators

Example notification:
```
🚨 KAELO SENSOR ALERT 🚨

📱 Device: ESP32-001
🕐 Time: 2024-01-15 14:30:25

📊 Current Readings:
🌡️ Temperature: 38.5°C
💧 Humidity: 65.2%
💨 Dust: 25.3 μg/m³

⚠️ Detected Issues:
🔴 🔥 High Temperature Alert
   └ Temperature 38.5°C exceeds maximum threshold of 35.0°C

💡 Recommended Action:
Please check the environment and take appropriate measures to normalize the conditions.

🔴 Status: ATTENTION REQUIRED
```

## 🔧 Firebase Data Structure

Expected Firebase Realtime Database structure:

```json
{
  "sensors": {
    "ESP32-001": {
      "temperature": 25.5,
      "humidity": 60.2,
      "dust": 15.3,
      "timestamp": "2024-01-15T14:30:25Z"
    },
    "ESP32-002": {
      "temperature": 26.1,
      "humidity": 58.7,
      "dust": 12.8,
      "timestamp": "2024-01-15T14:30:30Z"
    }
  }
}
```

## 🚀 CI/CD Pipeline

The project includes GitHub Actions for:

- Automated Docker image building
- Multi-architecture support (AMD64, ARM64)
- Publishing to GitHub Container Registry (GHCR)
- Semantic versioning with tags

## 📝 Logs

The service provides detailed logging with emojis for easy monitoring:

- 🚀 Service startup
- 📡 Data reception
- ⚠️ Anomaly detection
- ✅ Successful notifications
- ❌ Error conditions

## 🛠️ Development

### Project Structure

```
kaelo-service/
├── config/          # Configuration management
├── models/          # Data models
├── services/        # Business logic services
├── .github/         # GitHub Actions workflows
├── main.go          # Application entry point
├── Dockerfile       # Container definition
├── docker-compose.yml
└── README.md
```

### Adding New Sensors

1. Update the `SensorData` model in `models/sensor.go`
2. Add new anomaly types and thresholds in `config/config.go`
3. Implement detection logic in `services/anomaly.go`
4. Update Telegram formatting in `services/telegram.go`

## 🔒 Security Notes

- Never commit `.env` files to version control
- Use environment variables for all sensitive data
- The Firebase service account JSON should be stored as a single-line string
- Ensure proper network security when deploying on Raspberry Pi

## 📞 Support

For issues and questions, please create an issue in the GitHub repository.

## 📄 License

This project is licensed under the MIT License.
