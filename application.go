package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

var (
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
	AggInQuerySize        int           `bson:"aggInQuerySize"`
	DefaultContextTimeout time.Duration `bson:"defaultContextTimeout"`
	UpdateInterval        time.Duration `bson:"-"`
	FindMaxTimeMS         int           `bson:"findMaxTimeMs"`
	FindTimeoutMS         int           `bson:"findTimeoutMs"`
	AggMaxTimeMS          int           `bson:"aggMaxTimeMs"`
	AggTimeoutMS          int           `bson:"aggTimeoutMs"`
}

func init() {
	initMetrics()
	initMongoDB()

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

func refreshConfigPeriodically() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			refreshConfig()
			// Collect metrics
			metrics := getMetrics()
			fmt.Println("Got metrics %v", metrics)
			// Store metrics in MongoDB
			storeMetricsInDB(metrics)
			clearEventCounts()
		}
	}
}
