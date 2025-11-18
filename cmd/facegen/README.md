# Face Recognition Test Generator

Tool สำหรับทดสอบส่ง face recognition message ไปยัง RabbitMQ

## Usage

### Basic (ไม่มีรูป)

```bash
go run cmd/facegen/main.go
```

UUID v4 จะถูก generate ใหม่ทุกครั้งโดยอัตโนมัติ

### With Image (มีรูป)

```bash
go run cmd/facegen/main.go \
  -image /path/to/photo.jpg
```

### Custom RabbitMQ URL

```bash
go run cmd/facegen/main.go \
  -image photo.jpg \
  -rabbitmq amqp://kaelo:kaelo2024@192.168.1.100:5672/
```

## Parameters

| Flag | Default | Description |
|------|---------|-------------|
| `-image` | "" | Path to image file (JPG, PNG) |
| `-rabbitmq` | from config | RabbitMQ connection URL |

**Note**: UID จะถูก generate เป็น UUID v4 ใหม่ทุกครั้งโดยอัตโนมัติ

## Example

```bash
# ทดสอบส่ง unknown person alert พร้อมรูป
go run cmd/facegen/main.go -image profile_001.jpg

# UUID จะถูก generate อัตโนมัติ เช่น:
# d7f4c1b3-4e8a-4c2f-9a7b-1234567890ab

# ตรวจสอบ Telegram ควรได้รับ:
# - ข้อความแจ้งเตือน
# - UUID ที่ generate ใหม่
# - รูปภาพที่แนบมา
```

## Data Structure

```json
{
  "base64": "iVBORw0KGgoAAAANSUhEUgAA...",
  "uid": "d7f4c1b3-4e8a-4c2f-9a7b-1234567890ab",
  "timestamp": "2024-01-15T14:30:25Z"
}
```

UID จะเป็น UUID v4 format (xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx)

## Testing

```bash
# 1. Start kaelo-service
docker-compose up -d

# 2. ส่ง test message (UUID จะ generate อัตโนมัติ)
go run cmd/facegen/main.go -image profile_001.jpg

# 3. Check Telegram notification
# Should receive alert with:
# - Auto-generated UUID
# - Photo attachment
```

## Notes

- รองรับ JPG, PNG, และ image formats อื่นๆ
- รูปจะถูก encode เป็น base64 ก่อนส่ง
- ขนาดรูปไม่ควรเกิน 10MB (Telegram limit)
- Timestamp จะถูกตั้งเป็นเวลาปัจจุบันอัตโนมัติ

