package models

type Article struct {
	UserID int     `json:"user_id"`
	Title  string  `json:"title"`
	URL    string  `json:"url"`
	Price  float64 `json:"price"`
}
