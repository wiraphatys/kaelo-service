# RabbitMQ Setup Guide

## วิธีการ Enable MQTT Plugin

มีหลายวิธีในการ enable MQTT plugin ใน RabbitMQ:

## ✅ วิธีที่ 1: Mount Enabled Plugins File (แนะนำ - วิธีที่เราใช้)

วิธีนี้ใช้ไฟล์ `enabled_plugins` ที่ mount เข้าไปใน container

### Setup

1. สร้างไฟล์ `rabbitmq-enabled-plugins`:
```erlang
[rabbitmq_management,rabbitmq_mqtt].
```

2. Mount ใน docker-compose.yaml:
```yaml
services:
  rabbitmq:
    image: rabbitmq:3.13-management-alpine
    volumes:
      - ./rabbitmq-enabled-plugins:/etc/rabbitmq/enabled_plugins:ro
```

3. Start container:
```bash
docker-compose up -d rabbitmq
```

### ข้อดี:
- ✅ Enable อัตโนมัติทุกครั้งที่ start container
- ✅ ไม่ต้อง rebuild image
- ✅ แก้ไขง่าย (แก้ไฟล์แล้ว restart)
- ✅ Version control friendly

### ตรวจสอบว่า plugin enable แล้ว:
```bash
docker exec kaelo-rabbitmq rabbitmq-plugins list

# Output ควรเห็น:
# [E*] rabbitmq_mqtt
# [E*] rabbitmq_management
```

---

## วิธีที่ 2: Manual Enable (สำหรับทดสอบ)

Enable plugin ด้วยตนเองภายใน container ที่กำลังทำงาน

### คำสั่ง:
```bash
# 1. เข้าไปใน container
docker exec -it kaelo-rabbitmq bash

# 2. Enable plugin
rabbitmq-plugins enable rabbitmq_mqtt

# 3. ตรวจสอบ
rabbitmq-plugins list

# 4. Exit
exit
```

### ข้อเสีย:
- ❌ ต้องทำทุกครั้งที่สร้าง container ใหม่
- ❌ ไม่เหมาะกับ production

---

## วิธีที่ 3: Custom Dockerfile

สร้าง custom Docker image ที่มี plugin enabled

### Dockerfile.rabbitmq:
```dockerfile
FROM rabbitmq:3.13-management-alpine

# Enable MQTT plugin
RUN rabbitmq-plugins enable --offline rabbitmq_mqtt

# Expose MQTT port
EXPOSE 1883
```

### docker-compose.yaml:
```yaml
services:
  rabbitmq:
    build:
      context: .
      dockerfile: Dockerfile.rabbitmq
```

### ข้อดี:
- ✅ Plugin enable อัตโนมัติ
- ✅ เหมาะกับการ distribute image

### ข้อเสีย:
- ❌ ต้อง rebuild image ทุกครั้งที่แก้ไข
- ❌ ใช้เวลา build นานกว่า

---

## วิธีที่ 4: Environment Variable + Custom Entrypoint

ใช้ environment variable กับ custom entrypoint script

### docker-compose.yaml:
```yaml
services:
  rabbitmq:
    image: rabbitmq:3.13-management-alpine
    environment:
      RABBITMQ_SERVER_ADDITIONAL_ERL_ARGS: -rabbitmq_mqtt tcp_listeners [1883]
    command: >
      bash -c "rabbitmq-plugins enable --offline rabbitmq_mqtt &&
               docker-entrypoint.sh rabbitmq-server"
```

### ข้อเสีย:
- ❌ ซับซ้อน
- ❌ อาจมีปัญหา timing ในการ start

---

## 🎯 สรุป: เราใช้วิธีที่ 1 (Recommended)

```
rabbitmq-enabled-plugins file
    ↓
Mount to /etc/rabbitmq/enabled_plugins
    ↓
RabbitMQ reads on startup
    ↓
MQTT plugin enabled automatically
```

## 🧪 การทดสอบว่า MQTT Plugin ทำงาน

### 1. ตรวจสอบ Plugin Status
```bash
docker exec kaelo-rabbitmq rabbitmq-plugins list | grep mqtt
# Output: [E*] rabbitmq_mqtt
```

### 2. ตรวจสอบ MQTT Port
```bash
# ตรวจสอบว่า port 1883 เปิดอยู่
docker exec kaelo-rabbitmq netstat -tlnp | grep 1883

# หรือจากภายนอก
nc -zv localhost 1883
# Output: Connection to localhost 1883 port [tcp/*] succeeded!
```

### 3. ตรวจสอบใน Management UI
1. เปิด http://localhost:15672
2. Login: kaelo / kaelo2024
3. ไปที่ tab "Overview"
4. ควรเห็น "MQTT plugin enabled" หรือ port 1883 ใน "Ports and contexts"

### 4. Test MQTT Connection (ใช้ mosquitto client)
```bash
# Install mosquitto client
# macOS:
brew install mosquitto

# Ubuntu/Debian:
sudo apt-get install mosquitto-clients

# Test publish
mosquitto_pub -h localhost -p 1883 -u kaelo -P kaelo2024 \
  -t "sensor_data_queue" -m '{"test": "message"}'

# Test subscribe
mosquitto_sub -h localhost -p 1883 -u kaelo -P kaelo2024 \
  -t "sensor_data_queue" -v
```

## 🔧 Troubleshooting

### ปัญหา: Plugin ไม่ enable

**อาการ:**
```bash
docker exec kaelo-rabbitmq rabbitmq-plugins list | grep mqtt
# Output: [ ] rabbitmq_mqtt
```

**วิธีแก้:**
1. ตรวจสอบไฟล์ `rabbitmq-enabled-plugins`:
```bash
cat rabbitmq-enabled-plugins
# ต้องมี: [rabbitmq_management,rabbitmq_mqtt].
```

2. ตรวจสอบ docker-compose.yaml volume mount:
```bash
docker inspect kaelo-rabbitmq | grep enabled_plugins
```

3. Restart container:
```bash
docker-compose down
docker-compose up -d rabbitmq
```

### ปัญหา: MQTT Port ไม่เปิด

**อาการ:**
```bash
nc -zv localhost 1883
# Connection refused
```

**วิธีแก้:**
1. ตรวจสอบ plugin enabled:
```bash
docker exec kaelo-rabbitmq rabbitmq-plugins list | grep mqtt
```

2. ตรวจสอบ logs:
```bash
docker-compose logs rabbitmq | grep mqtt
# ควรเห็น: "MQTT plugin started"
```

3. ตรวจสอบ port binding:
```bash
docker-compose ps
# ควรเห็น: 0.0.0.0:1883->1883/tcp
```

### ปัญหา: Permission Denied

**อาการ:**
```bash
docker-compose up -d
# Error: permission denied for rabbitmq-enabled-plugins
```

**วิธีแก้:**
```bash
chmod 644 rabbitmq-enabled-plugins
```

## 📚 เอกสารเพิ่มเติม

- [RabbitMQ MQTT Plugin Official Docs](https://www.rabbitmq.com/mqtt.html)
- [RabbitMQ Plugins Guide](https://www.rabbitmq.com/plugins.html)
- [MQTT Protocol Specification](https://mqtt.org/mqtt-specification/)

## 🎓 ความรู้เพิ่มเติม

### ทำไมต้อง enable plugin?

RabbitMQ มี core ที่รองรับ AMQP protocol แต่ MQTT เป็น protocol คนละตัว ดังนั้น:
- ต้องมี plugin แปลง MQTT → AMQP
- Plugin จะทำหน้าที่เป็น bridge ระหว่าง 2 protocols
- ESP32 ใช้ MQTT (ง่าย สำหรับ IoT)
- kaelo-service ใช้ AMQP (powerful สำหรับ backend)

### MQTT vs AMQP

| Feature | MQTT | AMQP |
|---------|------|------|
| Protocol | Lightweight | Feature-rich |
| Use case | IoT devices | Enterprise messaging |
| Message size | Small | Any size |
| QoS levels | 3 (0, 1, 2) | Complex guarantees |
| Header overhead | Very low | Higher |
| Battery friendly | ✅ Yes | ❌ No |

ESP32 ใช้ MQTT เพราะ:
- ✅ น้ำหนักเบา (save battery)
- ✅ Library มีเยอะ
- ✅ Easy to implement
- ✅ ประหยัด bandwidth

## 🚀 Quick Reference

```bash
# Start RabbitMQ with MQTT
docker-compose up -d rabbitmq

# Check plugin status
docker exec kaelo-rabbitmq rabbitmq-plugins list

# Enable plugin manually (if needed)
docker exec kaelo-rabbitmq rabbitmq-plugins enable rabbitmq_mqtt

# Check logs
docker-compose logs rabbitmq | grep -i mqtt

# Test MQTT connection
mosquitto_pub -h localhost -p 1883 -u kaelo -P kaelo2024 \
  -t "test" -m "hello"

# Access Management UI
open http://localhost:15672
```

---

**Questions?** Check the main [README.md](../README-NEW.md) or [MIGRATION.md](../MIGRATION.md)

