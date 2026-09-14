package main

import (
	"net/http"
	"testing"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestRoutesRegistration(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("POST /items", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("POST /items/{id}/delete", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("GET /items/{id}", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("GET /items/{id}/row", func(w http.ResponseWriter, r *http.Request) {})
}
