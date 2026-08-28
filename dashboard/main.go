// Start HTTP Server on :8080
// Connects to Postgres
// Register routes: GET /, GET /items/{id}, POST /items, POST /items/{id}/delete
// Parses templates once at startup

package main

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"
)

var tmpl *template.Template

func initTemplates() {
	tmpl = template.Must(template.ParseGlob("dashboard/templates/*go.html"))
}

// func handler(w http.ResponseWriter, r *http.Request) {
// 	fmt.Fprintf(w, "Hello World! path: %s", r.URL.Path[1:])
// }

func main() {
	initTemplates()

	db, err := connectDB(10, 2*time.Second)
	if err != nil {
		log.Fatalf("Database connection error: %v", err)
	}
	defer db.Close()

	log.Println("Successfully connected to Database!")

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		items, err := listItems(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		tmpl.ExecuteTemplate(w, "layout", map[string]any{"Items": items})
	})

	mux.HandleFunc("POST /items", func(w http.ResponseWriter, r *http.Request) {
		title := r.FormValue("title")

		url := r.FormValue("url")

		if title == "" || url == "" {
			http.Error(w, "Title and URL required", http.StatusBadRequest)

			return
		}

		id, err := insertItem(db, title, url)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		item, err := getItem(db, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		tmpl.ExecuteTemplate(w, "item-row", item)
	})

	mux.HandleFunc("POST /items/{id}/delete", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = deleteItem(db, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	mux.HandleFunc("GET /items/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		item, err := getItem(db, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		tmpl.ExecuteTemplate(w, "layout", map[string]any{"Item": item})
	})
}
