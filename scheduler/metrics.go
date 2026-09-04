package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// TotalItems needing refresh
// Histogram Ticks

var TotalPublishedCounter = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "scheduler_published_total",
		Help: "Total number of jobs published",
	},
	[]string{"status"},
)

var TickDurationHistogram = promauto.NewHistogram(
	prometheus.HistogramOpts{
		Name: "scheduler_tick_duration_seconds",
		Help: "Duration of each DB check in seconds",
	},
)
