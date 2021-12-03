package main

import (
	"database/sql"
	"errors"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func initDB(path string) *sql.DB {

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		log.Printf("Creating %s...\n", path)
		file, err := os.Create(path)
		if err != nil {
			log.Fatal(err)
		}
		file.Close()

		db, _ := sql.Open("sqlite3", path)
		initTables(db)
		return db
	} else {
		db, err := sql.Open("sqlite3", path)
		if err != nil {
			log.Fatal(err)
		}
		return db
	}
}

func initTables(db *sql.DB) {

	q := `CREATE TABLE IF NOT EXISTS urls (
		id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		url VARCHAR(500) NOT NULL,
		author_id VARCHAR(18),
		guild_id VARCHAR(18),
		channel_id VARCHAR(18),
		status_code INTEGER
	);
	CREATE UNIQUE INDEX idx_urls_url ON urls(url);`
	s, err := db.Prepare(q)
	if err != nil {
		log.Fatal(err)
	}
	s.Exec()
}

func addArchived(db *sql.DB, m *Message, status_code int) {

	q := `INSERT OR IGNORE INTO urls(url, author_id, guild_id, channel_id, status_code) VALUES (?, ?, ?, ?, ?)`
	s, err := db.Prepare(q)
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()
	_, err = s.Exec(m.URL, m.Author, m.Guild, m.Channel, status_code)
	if err != nil {
		log.Fatal(err)
	}
}

func isCached(db *sql.DB, url string) (bool, int) {

	var status_code int
	err := db.QueryRow("SELECT status_code FROM urls WHERE url = ?",
		url).Scan(&status_code)
	switch {
	case err == sql.ErrNoRows:
		return false, status_code
	case err != nil:
		log.Fatal(err)
	}
	return true, status_code
}
