package main

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var JobsCounter = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "worker_jobs_total",
		Help: "Total number of worker jobs processed",
	},
	[]string{"status"},
)

var ScrapeDuration = promauto.NewHistogram(
	prometheus.HistogramOpts{
		Name: "worker_scrape_duration_seconds",
		Help: "Scrape duration in seconds",
	},
)

var ActiveWorkers = promauto.NewGauge(
	prometheus.GaugeOpts{
		Name: "worker_active_scrapes",
		Help: "Current Number of active scrapes",
	},
)

func StartMetricsServer(addr string) error {
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Printf("Listen and serve error: %v", err)
		return err
	}

	return nil
}
