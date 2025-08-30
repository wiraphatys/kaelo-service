package main

import (
	"context"
	"fmt"
	"log"
	"os"

	firebase "firebase.google.com/go/v4"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

// Acceleration defines the structure for acceleration data.
type Acceleration struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// Gyroscope defines the structure for gyroscope data.
type Gyroscope struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// SensorData matches the structure of a single record in your 'sensor-data' collection.
type SensorData struct {
	Acceleration   Acceleration `json:"acceleration"`
	DeviceID       string       `json:"device_id"`
	FlameDetected  bool         `json:"flame_detected"`
	GasQuality     string       `json:"gas_quality"`
	Gyroscope      Gyroscope    `json:"gyroscope"`
	Humidity       float64      `json:"humidity"`
	TemperatureDHT float64      `json:"temperature_dht"`
	TemperatureMPU float64      `json:"temperature_mpu"`
	Timestamp      string       `json:"timestamp"`
}

var FirebaseServiceAccountJSON string
var FirebaseDbUrl string

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	FirebaseServiceAccountJSON = os.Getenv("FIREBASE_SERVICE_ACCOUNT_JSON")
	FirebaseDbUrl = os.Getenv("FIREBASE_DB_URL")

	// Validate environment variables
	if FirebaseServiceAccountJSON == "" {
		log.Fatal("FIREBASE_SERVICE_ACCOUNT_JSON environment variable is not set")
	}
	if FirebaseDbUrl == "" {
		log.Fatal("FIREBASE_DB_URL environment variable is not set")
	}

	// Parse the service account JSON from environment variable
	serviceAccountJSON := []byte(FirebaseServiceAccountJSON)

	// Initialize Firebase app
	conf := &firebase.Config{
		DatabaseURL: FirebaseDbUrl,
	}

	opt := option.WithCredentialsJSON(serviceAccountJSON)
	app, err := firebase.NewApp(context.Background(), conf, opt)
	if err != nil {
		log.Fatalf("Error initializing Firebase app: %v", err)
	}

	// Get database client
	client, err := app.Database(context.Background())
	if err != nil {
		log.Fatalf("Error getting database client: %v", err)
	}

	// Option 1: Read single sensor data entry (if you know the key)
	// Uncomment this if you want to read a specific entry
	/*
		data := &SensorData{}
		err = client.NewRef("sensor-data/SPECIFIC_KEY_HERE").Get(context.Background(), data)
		if err != nil {
			log.Fatalf("Error reading specific sensor data: %v", err)
		}
		fmt.Printf("Single sensor data: %+v\n", data)
	*/

	// Option 2: Read all sensor data entries
	var allData map[string]SensorData
	err = client.NewRef("sensor-data").Get(context.Background(), &allData)
	if err != nil {
		log.Fatalf("Error reading sensor data: %v", err)
	}

	fmt.Printf("Total entries found: %d\n", len(allData))

	// Print all entries
	for key, data := range allData {
		fmt.Printf("Key: %s\n", key)
		fmt.Printf("Data: %+v\n", data)
		fmt.Println("---")
	}

	// // Option 3: Read the latest entry (if entries are ordered by timestamp)
	// // This assumes your keys are sortable (like timestamps)
	// if len(allData) > 0 {
	// 	var latestKey string
	// 	var latestData SensorData

	// 	for key, data := range allData {
	// 		if latestKey == "" || key > latestKey {
	// 			latestKey = key
	// 			latestData = data
	// 		}
	// 	}

	// 	fmt.Printf("\nLatest entry:\nKey: %s\nData: %+v\n", latestKey, latestData)
	// }
}
