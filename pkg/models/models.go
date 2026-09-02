package models

import "time"

type PricePoint struct {
	Price      float64
	RecordedAt time.Time
}

type Item struct {
	ID           int
	Title        string
	URL          string
	Price        float64
	StartPrice   float64
	LowestPrice  float64
	RecordedAt   time.Time
	PriceHistory []PricePoint
}

type ScrapeJob struct {
	ID  int    `json:"id"`
	URL string `json:"url"`
}
