# Build stage
FROM golang:1.23.4-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY .env .env
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o kaelo-service .

# Production stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /app/kaelo-service /kaelo-service

CMD ["/kaelo-service"]
