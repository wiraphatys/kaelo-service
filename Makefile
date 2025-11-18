mqttgen:
	go run cmd/mqttgen/main.go -rps 1 -anomaly 0.3

facegen:
	go run cmd/facegen/main.go -image ./image/profile_001.jpg