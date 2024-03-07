package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	client         *mongo.Client
	collection     *mongo.Collection
	configColl     *mongo.Collection
	config         Config
	configMutex    sync.Mutex
	updateInterval time.Duration = 60 * time.Second // Default update interval
	numRetries     int           = 2
)

type Document struct {
	ID    string `json:"id" bson:"_id"`
	Key   string `json:"key" bson:"key"`
	Value string `json:"value" bson:"value"`
	X     int    `json:"x" bson:"x"`
}

type Config struct {
	UpsertContextTimeout  time.Duration `bson:"upsertContextTimeout"`
	FindContextTimeout    time.Duration `bson:"findContextTimeout"`
	AggContextTimeout     time.Duration `bson:"aggContextTimeout"`
	DefaultContextTimeout time.Duration `bson:"defaultContextTimeout"`
	UpdateInterval        time.Duration `bson:"-"`
}

func init() {
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
		Event: func(evt *event.PoolEvent) {
			switch evt.Type {
			case event.ConnectionCreated:
				log.Printf("Connection created: %+v", evt)
			case event.ConnectionClosed:
				log.Printf("Connection closed: %+v", evt)
			case event.PoolCleared:
				log.Printf("Connection pool cleared: %+v", evt)
			}
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

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", healthCheck).Methods("GET")
	r.HandleFunc("/upsert", upsertDocument).Methods("GET")
	r.HandleFunc("/find", findDocuments).Methods("GET")
	r.HandleFunc("/agg", aggSampleGroup).Methods("GET")

	// Start server
	fmt.Println("Server listening on port 5000")
	log.Fatal(http.ListenAndServe(":5000", r))
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

func upsertDocument(w http.ResponseWriter, r *http.Request) {
	var doc Document
	doc.ID = uuid.New().String()
	doc.Key = uuid.New().String()
	doc.X = rand.Intn(500000) + 1

	ctx, cancel := context.WithTimeout(context.Background(), config.UpsertContextTimeout*time.Millisecond)
	defer cancel()

	filter := bson.M{"key": doc.Key}
	opts := options.Update().SetUpsert(true)
	update := bson.M{"$set": doc}

	var err error
	for i := 1; i <= numRetries; i++ {
		_, err = collection.UpdateOne(ctx, filter, update, opts)
		if err == nil {
			break
		}
		log.Printf("upsert error: %+v, attempt %v", err, i)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Created _id: %s\n", doc.Key)
}

func findDocuments(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), config.FindContextTimeout*time.Millisecond)
	defer cancel()

	randomX := rand.Intn(500000) + 1

	filter := bson.M{"x": randomX}
	opts := options.Find().SetProjection(bson.M{"_id": 1})

	var err error
	var cursor *mongo.Cursor
	for i := 1; i <= numRetries; i++ {
		cursor, err = collection.Find(ctx, filter, opts)
		if err == nil {
			break
		}
		log.Printf("find error: %+v, attempt %v", err, i)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var count int
	for cursor.Next(ctx) {
		count++
		if count >= 1000 {
			break
		}
	}

	fmt.Fprintf(w, "Number of documents found: %d\n", count)
}

func aggSampleGroup(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), config.AggContextTimeout*time.Millisecond)
	defer cancel()

	pipeline := bson.A{
		bson.D{{"$sample", bson.D{{"size", 100000}}}},
		bson.D{
			{"$group",
				bson.D{
					{"_id", 1},
					{"minid", bson.D{{"$min", "$_id"}}},
					{"maxid", bson.D{{"$max", "$_id"}}},
					{"minkey", bson.D{{"$min", "$key"}}},
					{"maxkey", bson.D{{"$max", "$key"}}},
					{"xavg", bson.D{{"$avg", "$x"}}},
				},
			},
		},
	}

	var err error
	var cursor *mongo.Cursor
	for i := 1; i <= numRetries; i++ {
		cursor, err = collection.Aggregate(ctx, pipeline)
		if err == nil {
			break
		}
		log.Printf("agg error: %+v, attempt %v", err, i)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var count int
	for cursor.Next(ctx) {
		count++
		if count >= 1000 {
			break
		}
	}

	fmt.Fprintf(w, "Aggregation returned: %d\n", count)
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "All good here at %s\n", time.Now().String())
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

func refreshConfigPeriodically() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			refreshConfig()
		}
	}
}
