package main

import (
	"context"
	"log"

	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func trackConnectionEvents(e *event.PoolEvent) {
	// Track connection events
	incrementEventCount(e.Type.String())
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
	// Setup event listeners for connection events
	poolMonitor := &event.PoolMonitor{
		Event: trackConnectionEvents,
	}
	clientOptions := options.Client().SetPoolMonitor(poolMonitor)
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize eventCounts map
	eventCounts = make(map[string]int)

	// Setup event listeners for MongoDB errors
	client.Error = func(_ *mongo.Client, err error) {
		trackMongoDBErrors(err)
	}
}
