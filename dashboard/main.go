// Start HTTP Server on :8080
// Connects to Postgres
// Register routes: GET /, GET /items/{id}, POST /items, POST /items/{id}/delete
// Parses templates once at startup

package main

import (
	"database/sql"
	"errors"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	dbpkg "github.com/Bommel48/go-scraper-notifier/pkg/db"
)

var (
	indexTmpl *template.Template
	itemTmpl  *template.Template
)

func initTemplates() {
	baseDir := "dashboard/templates"
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		baseDir = "templates"
	}

	layoutPath := filepath.Join(baseDir, "layout.go.html")
	indexPath := filepath.Join(baseDir, "index.go.html")
	itemPath := filepath.Join(baseDir, "item.go.html")

	indexTmpl = template.Must(template.ParseFiles(layoutPath, indexPath))
	itemTmpl = template.Must(template.ParseFiles(layoutPath, itemPath))
}

func main() {
	initTemplates()

	db, err := connectDB(10, 2*time.Second)
	if err != nil {
		log.Fatalf("Database connection error: %v", err)
	}
	defer db.Close()

	if err := dbpkg.InitSchema(db); err != nil {
		log.Fatalf("Schema setup error: %v", err)
	}

	log.Println("Successfully connected to Database!")

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		items, err := listItems(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		indexTmpl.ExecuteTemplate(w, "layout", map[string]any{"Items": items})
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

		indexTmpl.ExecuteTemplate(w, "item-row", item)
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
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "Item not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		itemTmpl.ExecuteTemplate(w, "layout", map[string]any{"Item": item})
	})

	log.Println("Dashboard running on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
