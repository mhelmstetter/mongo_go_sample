package main

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	client      *mongo.Client
	collection  *mongo.Collection
	metricsColl *mongo.Collection
	configColl  *mongo.Collection
	config      Config
	configMutex sync.Mutex
	mutex       sync.Mutex
	eventCounts map[string]int
)

func initMongoDB() {
	connString := os.Getenv("MONGODB_CONNECTION_STRING")
	if connString == "" {
		log.Fatal("MONGODB_CONNECTION_STRING environment variable is not set")
	}

	serverMonitor := &event.ServerMonitor{
		ServerDescriptionChanged: func(evt *event.ServerDescriptionChangedEvent) {
			log.Printf("Server description changed: %+v", evt)
		},
		TopologyDescriptionChanged: func(evt *event.TopologyDescriptionChangedEvent) {
			log.Printf("Topology description changed: %+v", evt)
		},
		ServerHeartbeatFailed: func(evt *event.ServerHeartbeatFailedEvent) {
			log.Printf("Server heartbeat failed: %+v", evt)
		},
	}

	poolMonitor := &event.PoolMonitor{
		Event: func(e *event.PoolEvent) {
			log.Printf("Pool Event Type: %v", e.Type)
			log.Printf("Pool Event: %v", e)
		},
	}

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	clientOptions := options.Client().
		ApplyURI(connString).
		SetServerMonitor(serverMonitor).
		SetPoolMonitor(poolMonitor)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	// Accessing the "testdb" database and collections
	db := client.Database("testdb")
	collection = db.Collection("documents")
	configColl = db.Collection("config")
	metricsColl = client.Database("testdb").Collection("metrics")

	// Ensure the config document exists and insert default config if it doesn't
	initConfig(ctx)

	// Load initial config
	refreshConfig()

	// Create indexes for documents collection
	_, err = collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "x", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "key", Value: 1}},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Start background task to periodically refresh config
	go refreshConfigPeriodically()
}

func incrementEventCount(eventType string) {
	mutex.Lock()
	defer mutex.Unlock()
	eventCounts[eventType]++
}

func storeMetrics() {
	for {
		// Collect metrics
		metrics := getMetrics()

		// Store metrics in MongoDB
		storeMetricsInDB(metrics)

		// Sleep for 60 seconds
		time.Sleep(60 * time.Second)
	}
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
		"env":         "Mongogosample2-env",
		"eventCounts": counts,
	}

	return metrics
}

func storeMetricsInDB(metrics bson.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := metricsColl.InsertOne(ctx, metrics)
	if err != nil {
		log.Printf("Error storing metrics: %v\n", err)
	}
}
