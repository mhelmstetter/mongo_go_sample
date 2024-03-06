package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	client     *mongo.Client
	collection *mongo.Collection
)

type Document struct {
	ID    string `json:"id" bson:"_id"`
	Key   string `json:"key" bson:"key"`
	Value string `json:"value" bson:"value"`
	X     int    `json:"x" bson:"x"`
}

func init() {
	// Get MongoDB connection string from environment variable
	connString := os.Getenv("MONGODB_CONNECTION_STRING")
	if connString == "" {
		log.Fatal("MONGODB_CONNECTION_STRING environment variable is not set")
	}

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	clientOptions := options.Client().ApplyURI(connString)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	collection = client.Database("testdb").Collection("documents")
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/upsert", upsertDocument).Methods("GET")
	r.HandleFunc("/find", findDocuments).Methods("GET")

	// Start server
	fmt.Println("Server listening on port 5000")
	log.Fatal(http.ListenAndServe(":5000", r))
}

func upsertDocument(w http.ResponseWriter, r *http.Request) {
	var doc Document
	doc.ID = uuid.New().String()
	doc.Key = uuid.New().String()
	doc.X = rand.Intn(500000) + 1

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Upsert document
	filter := bson.M{"key": doc.Key}
	opts := options.Update().SetUpsert(true)
	update := bson.M{"$set": doc}
	_, err := collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	fmt.Fprintf(w, "Created _id: %s\n", doc.Key)
}

func findDocuments(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Generate a random X value between 1 and 500,000
	randomX := rand.Intn(500000) + 1

	// Find documents
	filter := bson.M{"x": randomX}
	opts := options.Find().SetProjection(bson.M{"_id": 1})
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	// Count the number of documents found
	var count int
	for cursor.Next(ctx) {
		count++
		if count >= 1000 {
			break // Limit to 1000 documents
		}
	}

	// Return count
	fmt.Fprintf(w, "Number of documents found: %d\n", count)
}
