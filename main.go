package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", healthCheck).Methods("GET")
	r.HandleFunc("/upsert", upsertDocument).Methods("GET")
	r.HandleFunc("/find", findDocuments).Methods("GET")
	r.HandleFunc("/agg", aggSampleGroup).Methods("GET")

	// Start server
	fmt.Println("Server listening on port 5000")
	log.Fatal(http.ListenAndServe(":5000", r))

	// Initialize MongoDB and start metrics collection
	initMongoDB()
	go storeMetrics()
}
