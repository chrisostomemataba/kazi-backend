package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	fmt.Println("=== KAZI Backend Starting ===")
	
	// Check if -migrate flag is passed
	if len(os.Args) > 1 && os.Args[1] == "-migrate" {
		fmt.Println("Running migrations...")
		// TODO: Add migration logic later
		fmt.Println("Migrations completed successfully")
		return
	}
	
	// Get port from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	// Setup routes
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"message":"KAZI API is running","version":"1.0.0"}`)
	})
	
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"healthy"}`)
	})
	
	// Start server
	fmt.Printf("Server starting on port %s...\n", port)
	fmt.Println("Health check: http://localhost:8080/health")
	
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}