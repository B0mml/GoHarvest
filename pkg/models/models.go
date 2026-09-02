package models

import "time"

type Article struct {
	UserID int     `json:"user_id"`
	Title  string  `json:"title"`
	URL    string  `json:"url"`
	Price  float64 `json:"price"`
}

type PricePoint struct {
	Price      float64
	RecordedAt time.Time
}

type Item struct {
	ID           int
	Title        string
	URL          string
	Price        float64
	RecordedAt   time.Time
	PriceHistory []PricePoint
}
