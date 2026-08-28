package db

import (
	"database/sql"
	"fmt"
)

// InitSchema ensures the required tables and initial seed data exist in PostgreSQL.
func InitSchema(db *sql.DB) error {
	createUsersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	createItemsTable := `
	CREATE TABLE IF NOT EXISTS items (
		id SERIAL PRIMARY KEY,
		user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		title TEXT NOT NULL,
		url TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		CONSTRAINT unique_user_url UNIQUE (user_id, url)
	);`

	createHistoryTable := `
	CREATE TABLE IF NOT EXISTS price_history (
		id SERIAL PRIMARY KEY,
		item_id INT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
		price NUMERIC(10, 2) NOT NULL,
		recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	seedUsers := `
	INSERT INTO users (id, email) VALUES 
		(1, 'user1@example.com'),
		(2, 'user2@example.com'),
		(3, 'user3@example.com')
	ON CONFLICT (id) DO NOTHING;`

	if _, err := db.Exec(createUsersTable); err != nil {
		return fmt.Errorf("error creating users table: %w", err)
	}
	if _, err := db.Exec(createItemsTable); err != nil {
		return fmt.Errorf("error creating items table: %w", err)
	}
	if _, err := db.Exec(createHistoryTable); err != nil {
		return fmt.Errorf("error creating price_history table: %w", err)
	}
	if _, err := db.Exec(seedUsers); err != nil {
		return fmt.Errorf("error seeding users: %w", err)
	}
	return nil
}
