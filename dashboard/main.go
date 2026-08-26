// Start HTTP Server on :8080
// Connects to Postgres
// Register routes: GET /, GET /items/{id}, POST /items, POST /items/{id}/delete
// Parses templates once at startup

package main

import (
	"fmt"
	"log"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World! path: %s", r.URL.Path[1:])
}
func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)

	log.Fatal(http.ListenAndServe(":8080", mux))
}
