package main

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	client      *mongo.Client
	collection  *mongo.Collection
	configColl  *mongo.Collection
	metricsColl *mongo.Collection
)

func initMongoDB() {
	// Get MongoDB connection string from environment variable
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
			//log.Printf("Pool Event Type: %v", e.Type)
			log.Printf("Pool Event: %v", e)
			trackConnectionEvents(e)
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
}

func initConfig(ctx context.Context) {
	// Check if config document exists, if not, insert default config
	count, err := configColl.CountDocuments(ctx, bson.M{})
	if err != nil {
		log.Fatal(err)
	}
	if count == 0 {
		defaultConfig := Config{
			UpsertContextTimeout:  500,
			FindContextTimeout:    500,
			AggContextTimeout:     500,
			DefaultContextTimeout: 500,
			UpdateInterval:        updateInterval,
		}
		_, err := configColl.InsertOne(ctx, defaultConfig)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func refreshConfig() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	configMutex.Lock()
	defer configMutex.Unlock()

	err := configColl.FindOne(ctx, bson.M{}).Decode(&config)
	if err != nil {
		log.Printf("Error refreshing config: %v\n", err)
	}
}

func storeMetricsInDB(metrics bson.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := metricsColl.InsertOne(ctx, metrics)
	if err != nil {
		log.Printf("Error storing metrics: %v\n", err)
	}
}
