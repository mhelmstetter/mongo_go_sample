package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
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

	envName, _ := GetEnvironmentName("/var/lib/cfn-init/data/metadata.json")

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
