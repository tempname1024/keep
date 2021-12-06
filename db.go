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
		id integer NOT NULL PRIMARY KEY,
		url VARCHAR(500) NOT NULL,
		user_string_id INTEGER NOT NULL,
		guild_string_id INTEGER NOT NULL,
		channel_string_id INTEGER NOT NULL,
		status_code INTEGER NOT NULL
	);
	CREATE UNIQUE INDEX idx_urls_url ON urls(url);

	CREATE TABLE IF NOT EXISTS users (
		id integer NOT NULL PRIMARY KEY,
		user_id VARCHAR(18)
	);
	CREATE UNIQUE INDEX idx_users_user_id ON users(user_id);

	CREATE TABLE IF NOT EXISTS guilds (
		id integer NOT NULL PRIMARY KEY,
		guild_id VARCHAR(18)
	);
	CREATE UNIQUE INDEX idx_guilds_guild_id ON guilds(guild_id);

	CREATE TABLE IF NOT EXISTS channels (
		id integer NOT NULL PRIMARY KEY,
		channel_id VARCHAR(18)
	);
	CREATE UNIQUE INDEX idx_channels_channel_id ON channels(channel_id);`
	_, err := db.Exec(q)
	if err != nil {
		log.Fatal(err)
	}
}

func addArchived(db *sql.DB, m *Message, status_code int) {

	// Start a transaction using default isolation
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	// Insert new entries in users, guilds, channels tables for new values,
	// ignoring those already present
	_, err = tx.Exec(`
		INSERT OR IGNORE INTO users(user_id) VALUES(?);
		INSERT OR IGNORE INTO guilds(guild_id) VALUES(?);
		INSERT OR IGNORE INTO channels(channel_id) VALUES(?);`,
		m.Author, m.Guild, m.Channel)
	if err != nil {
		_ = tx.Rollback()
		log.Fatal(err)
	}

	// Store IDs of previously-inserted (or already-existent) rows
	var user_string_id int
	var guild_string_id int
	var channel_string_id int

	// Query users/guilds/channels tables for aforementioned rows
	err = tx.QueryRow("SELECT id FROM users WHERE user_id = ?;",
		m.Author).Scan(&user_string_id)
	if err != nil {
		_ = tx.Rollback()
		log.Fatal(err)
	}
	err = tx.QueryRow("SELECT id FROM guilds WHERE guild_id = ?;",
		m.Guild).Scan(&guild_string_id)
	if err != nil {
		_ = tx.Rollback()
		log.Fatal(err)
	}
	err = tx.QueryRow("SELECT id FROM channels WHERE channel_id = ?;",
		m.Channel).Scan(&channel_string_id)
	if err != nil {
		log.Fatal(err)
	}

	// Insert entry in URLs table using IDs populated by previous selections
	_, err = tx.Exec(`INSERT OR IGNORE INTO
		urls(url, user_string_id, guild_string_id, channel_string_id, status_code)
		VALUES(?, ?, ?, ?, ?);`,
		m.URL, user_string_id, guild_string_id, channel_string_id, status_code)
	if err != nil {
		_ = tx.Rollback()
		log.Fatal(err)
	}

	// Finally commit the transaction
	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
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
