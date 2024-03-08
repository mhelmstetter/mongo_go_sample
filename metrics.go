package main

import (
	"go.mongodb.org/mongo-driver/event"
)

func trackConnectionEvents(e *event.PoolEvent) {
	// Track connection events
	incrementEventCount(e.Type)
}

func trackMongoDBErrors(err error) {
	// Track MongoDB errors
	errorMessage := err.Error()

	// Custom error categorization logic
	if errorMessage == "context deadline exceeded" {
		incrementEventCount(errorMessage)
	} else {
		// Check for other error categories
		// ...
		// If not found, categorize as "Other" or something similar
		incrementEventCount("Other")
	}
}

func init() {

	// Initialize eventCounts map
	eventCounts = make(map[string]int)

	// // Setup event listeners for MongoDB errors
	// client.Error = func(_ *mongo.Client, err error) {
	// 	trackMongoDBErrors(err)
	// }
}
