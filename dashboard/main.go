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
	"strings"
	"time"

	dbpkg "github.com/Bommel48/go-scraper-notifier/pkg/db"
	"github.com/Bommel48/go-scraper-notifier/pkg/models"
	rbmq "github.com/Bommel48/go-scraper-notifier/pkg/rabbitmq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	indexTmpl *template.Template
	itemTmpl  *template.Template
)

func initTemplates() {
	baseDir := "templates"
	if _, err := os.Stat(filepath.Join("dashboard", "templates", "layout.go.html")); err == nil {
		baseDir = "dashboard/templates"
	}

	layoutPath := filepath.Join(baseDir, "layout.go.html")
	indexPath := filepath.Join(baseDir, "index.go.html")
	itemPath := filepath.Join(baseDir, "item.go.html")

	indexTmpl = template.Must(template.ParseFiles(layoutPath, indexPath))
	itemTmpl = template.Must(template.ParseFiles(layoutPath, itemPath))
}

func main() {
	initTemplates()

	db, err := dbpkg.Connect(10, 2*time.Second)
	if err != nil {
		log.Fatalf("Database connection error: %v", err)
	}
	defer db.Close()

	log.Println("Successfully connected to Database!")

	amqpURL := os.Getenv("AMQP_URL")
	if amqpURL == "" {
		amqpURL = "amqp://guest:guest@rabbitmq:5672/"
	}
	ch, err := rbmq.Connect(amqpURL, 10, 2*time.Second)
	if err != nil {
		log.Fatalf("RabbitMQ connection error: %v", err)
	}
	defer ch.Close()

	mux := http.NewServeMux()

	mux.Handle("GET /metrics", promhttp.Handler())

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
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

		var targetPrice float64
		if tpStr := r.FormValue("target_price"); tpStr != "" {
			cleanStr := strings.Replace(strings.TrimSpace(tpStr), ",", ".", 1)
			if tp, err := strconv.ParseFloat(cleanStr, 64); err == nil {
				targetPrice = tp
			}
		}

		id, err := insertItem(db, title, url, targetPrice)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		item, err := getItem(db, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		scrapeJob := models.ScrapeJob{
			ID:  id,
			URL: url,
		}
		if err := rbmq.Publish(ch, rbmq.QueueName, scrapeJob); err != nil {
			log.Printf("Error publishing item %d to RabbitMQ: %v", id, err)
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

	mux.HandleFunc("GET /items/{id}/row", func(w http.ResponseWriter, r *http.Request) {
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

		indexTmpl.ExecuteTemplate(w, "item-row", item)
	})

	mux.HandleFunc("POST /items/{id}/target-price", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var targetPrice float64
		if tpStr := r.FormValue("target_price"); tpStr != "" {
			cleanStr := strings.Replace(strings.TrimSpace(tpStr), ",", ".", 1)
			if tp, err := strconv.ParseFloat(cleanStr, 64); err == nil && tp > 0 {
				targetPrice = tp
			}
		}

		if err := updateTargetPrice(db, id, targetPrice); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		item, err := getItem(db, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		itemTmpl.ExecuteTemplate(w, "target-price-card", map[string]any{"Item": item})
	})

	log.Println("Dashboard running on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
