package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Quote struct {
	ID        int
	Who       string
	Body      string
	CreatedAt time.Time
}

func dbPath() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".quotes")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "quotes.db")
}

func openDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath())
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS quotes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		who TEXT NOT NULL,
		body TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func countQuotes(db *sql.DB, search string) (int, error) {
	var count int
	if search == "" {
		err := db.QueryRow("SELECT COUNT(*) FROM quotes").Scan(&count)
		return count, err
	}
	pattern := "%" + search + "%"
	err := db.QueryRow(
		"SELECT COUNT(*) FROM quotes WHERE who LIKE ? OR body LIKE ?",
		pattern, pattern,
	).Scan(&count)
	return count, err
}

func fetchQuotes(db *sql.DB, search string, limit, offset int) ([]Quote, error) {
	var rows *sql.Rows
	var err error
	if search == "" {
		rows, err = db.Query(
			"SELECT id, who, body, created_at FROM quotes ORDER BY created_at DESC LIMIT ? OFFSET ?",
			limit, offset,
		)
	} else {
		pattern := "%" + search + "%"
		rows, err = db.Query(
			"SELECT id, who, body, created_at FROM quotes WHERE who LIKE ? OR body LIKE ? ORDER BY created_at DESC LIMIT ? OFFSET ?",
			pattern, pattern, limit, offset,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var quotes []Quote
	for rows.Next() {
		var q Quote
		if err := rows.Scan(&q.ID, &q.Who, &q.Body, &q.CreatedAt); err != nil {
			return nil, err
		}
		quotes = append(quotes, q)
	}
	return quotes, rows.Err()
}

func insertQuote(db *sql.DB, who, body string) error {
	_, err := db.Exec("INSERT INTO quotes (who, body, created_at) VALUES (?, ?, ?)", who, body, time.Now())
	return err
}

func deleteQuote(db *sql.DB, id int) error {
	res, err := db.Exec("DELETE FROM quotes WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("quote %d not found", id)
	}
	return nil
}
