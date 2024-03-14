package main

import (
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/event"
)

var (
	mutex           sync.Mutex
	eventCounts     map[string]int
	envName         string
	hostname        string
	errorCategories = []string{"context deadline exceeded"}
)

func initMetrics() {

	eventCounts = make(map[string]int)
	var err error
	envName, err = GetEnvironmentName()
	if err != nil {
		log.Printf("Error getting environment name: %+v", err)
		envName = "unknown"
	}
	log.Printf("Environment name: %+v", envName)
	hostname, _ = os.Hostname()
}

func trackConnectionEvents(e *event.PoolEvent) {
	// Track connection events
	incrementEventCount(e.Type)
}

func trackMongoDBErrors(err error) {
	errorMessage := err.Error()
	categorized := false
	for _, category := range errorCategories {
		if strings.Contains(errorMessage, category) {
			incrementEventCount(category)
			categorized = true
			break
		}
	}
	if !categorized {
		log.Printf("Uncategorized error: %s", errorMessage)
		incrementEventCount("unknown")
	}
}

func incrementEventCount(eventType string) {
	mutex.Lock()
	defer mutex.Unlock()
	eventCounts[eventType]++
}

func getMetrics() bson.M {
	mutex.Lock()
	defer mutex.Unlock()

	// Copy eventCounts to avoid concurrent map access
	counts := make(map[string]int)
	for k, v := range eventCounts {
		counts[k] = v
	}

	// Construct metrics document
	metrics := bson.M{
		"_id":         primitive.NewObjectID(),
		"dt":          time.Now(),
		"env":         envName,
		"hostname":    hostname,
		"eventCounts": counts,
	}

	return metrics
}

func clearEventCounts() {
	mutex.Lock()
	defer mutex.Unlock()

	eventCounts = make(map[string]int)
}
