package main

import (
	"database/sql"
	"fmt"

	models "github.com/Bommel48/go-scraper-notifier/pkg/models"
)

func listItems(db *sql.DB) ([]models.Item, error) {
	rows, err := db.Query(`
	SELECT DISTINCT ON (i.id)
        i.id,
        i.title,
        i.url,
        COALESCE(p.price, 0),
        COALESCE(p.recorded_at, i.created_at)
    FROM items i
    LEFT JOIN price_history p ON i.id = p.item_id
    ORDER BY i.id DESC, p.recorded_at DESC NULLS LAST;`)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	var items []models.Item
	for rows.Next() {
		var it models.Item
		if err := rows.Scan(&it.ID, &it.Title, &it.URL, &it.Price, &it.RecordedAt); err != nil {
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
