package models

import "time"

// FaceRecognitionData represents face recognition data from camera
type FaceRecognitionData struct {
	Base64    string    `json:"base64"`
	UID       string    `json:"uid"`
	Timestamp time.Time `json:"timestamp"`
}
