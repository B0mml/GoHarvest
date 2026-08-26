// Start HTTP Server on :8080
// Connects to Postgres
// Register routes: GET /, GET /items/{id}, POST /items, POST /items/{id}/delete
// Parses templates once at startup

package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World! path: %s", r.URL.Path[1:])
}

func main() {
	db, err := connectDB(10, 2*time.Second)
	if err != nil {
		log.Fatalf("Database connection error: %v", err)
	}
	defer db.Close()

	log.Println("Successfully connected to Database!")

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)

	log.Println("Dashboard server listening on :8080...")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
