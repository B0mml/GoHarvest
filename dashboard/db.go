package main

import (
	"database/sql"
	"fmt"

	models "github.com/Bommel48/go-scraper-notifier/pkg/models"
)

func listItems(db *sql.DB) ([]models.Item, error) {
	query := `
	SELECT
		i.id,
		i.title,
		i.url,
		COALESCE(latest.price, 0) AS current_price,
		COALESCE(first_p.price, 0) AS start_price,
		COALESCE(min_p.price, 0) AS lowest_price,
		COALESCE(latest.recorded_at, i.created_at) AS recorded_at
	FROM items i
	LEFT JOIN LATERAL (
		SELECT price, recorded_at FROM price_history WHERE item_id = i.id ORDER BY recorded_at DESC, id DESC LIMIT 1
	) latest ON true
	LEFT JOIN LATERAL (
		SELECT price FROM price_history WHERE item_id = i.id ORDER BY recorded_at ASC, id ASC LIMIT 1
	) first_p ON true
	LEFT JOIN (
		SELECT item_id, MIN(price) AS price FROM price_history GROUP BY item_id
	) min_p ON min_p.item_id = i.id
	ORDER BY i.id DESC;`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	var items []models.Item
	for rows.Next() {
		var it models.Item
		if err := rows.Scan(&it.ID, &it.Title, &it.URL, &it.Price, &it.StartPrice, &it.LowestPrice, &it.RecordedAt); err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		items = append(items, it)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return items, nil
}

// // returns one item + all its price history rows
func getItem(db *sql.DB, id int) (models.Item, error) {
	var it models.Item

	query := `SELECT id, title, url FROM items WHERE id = $1;`
	err := db.QueryRow(query, id).Scan(&it.ID, &it.Title, &it.URL)
	if err != nil {
		return it, fmt.Errorf("failed to get item: %w", err)
	}

	historyQuery := `SELECT price, recorded_at FROM price_history WHERE item_id = $1 ORDER BY recorded_at ASC;`
	rows, err := db.Query(historyQuery, id)
	if err != nil {
		return it, fmt.Errorf("failed to get price history: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var p models.PricePoint
		if err := rows.Scan(&p.Price, &p.RecordedAt); err != nil {
			return it, err
		}
		it.PriceHistory = append(it.PriceHistory, p)
	}

	if len(it.PriceHistory) > 0 {
		it.StartPrice = it.PriceHistory[0].Price
		it.Price = it.PriceHistory[len(it.PriceHistory)-1].Price
		it.RecordedAt = it.PriceHistory[len(it.PriceHistory)-1].RecordedAt
		lowest := it.PriceHistory[0].Price
		for _, p := range it.PriceHistory {
			if p.Price < lowest {
				lowest = p.Price
			}
		}
		it.LowestPrice = lowest
	}

	if err := rows.Err(); err != nil {
		return it, fmt.Errorf("rows iteration error: %w", err)
	}

	return it, nil
}

// insertItem inserts an item and returns the item ID
func insertItem(db *sql.DB, title string, url string) (int, error) {
	query := `
            INSERT INTO items (user_id, title, url)
            VALUES ($1, $2, $3)
            ON CONFLICT (user_id, url) DO UPDATE SET title = EXCLUDED.title
            RETURNING id;`

	var itemID int
	err := db.QueryRow(query, 1, title, url).Scan(&itemID)
	if err != nil {
		return 0, fmt.Errorf("failed to upsert item: %w", err)
	}

	return itemID, nil
}

// // delete from items (cascades to price_history)
func deleteItem(db *sql.DB, id int) error {
	_, err := db.Exec(`DELETE FROM items WHERE id = $1;`, id)
	if err != nil {
		return fmt.Errorf("failed to delete item %d: %w", id, err)
	}

	return nil
}
