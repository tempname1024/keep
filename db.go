package main

import (
	"database/sql"
	"errors"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

type Entry struct {
	ID      int
	Message Message
	Status  int
}

type Stats struct {
	Users    int
	Guilds   int
	Channels int
	URLs     int
}

func initDB(path string) *sql.DB {

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
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

	q := `
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
	CREATE UNIQUE INDEX idx_channels_channel_id ON channels(channel_id);

	CREATE TABLE IF NOT EXISTS urls (
		id integer NOT NULL PRIMARY KEY,
		url VARCHAR(500) NOT NULL,
		user_string_id INTEGER NOT NULL,
		guild_string_id INTEGER NOT NULL,
		channel_string_id INTEGER NOT NULL,
		status_code INTEGER NOT NULL,
		FOREIGN KEY(user_string_id) REFERENCES users(id),
		FOREIGN KEY(guild_string_id) REFERENCES guilds(id),
		FOREIGN KEY(channel_string_id) REFERENCES channels(id)
	);
	CREATE UNIQUE INDEX idx_urls_url ON urls(url);`

	_, err := db.Exec(q)
	if err != nil {
		log.Fatal(err)
	}
}

func (db *SqliteDB) AddArchived(m *Message, status_code int) {

	// Start a transaction using default isolation
	tx, err := db.db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback()

	// Insert new entries in users, guilds, channels tables for new values,
	// ignoring if already present
	_, err = tx.Exec(`
		INSERT OR IGNORE INTO users(user_id) VALUES(?);
		INSERT OR IGNORE INTO guilds(guild_id) VALUES(?);
		INSERT OR IGNORE INTO channels(channel_id) VALUES(?);`,
		m.Author, m.Guild, m.Channel)
	if err != nil {
		log.Fatal(err)
	}

	// Insert entry in URLs table using foreign key reference IDs
	_, err = tx.Exec(`
	INSERT OR IGNORE INTO
	urls(url, user_string_id, guild_string_id, channel_string_id, status_code)
	VALUES(
		?,
		(SELECT id FROM users WHERE user_id = ?),
		(SELECT id FROM guilds WHERE guild_id = ?),
		(SELECT id FROM channels WHERE channel_id = ?),
		?
	);`, m.URL, m.Author, m.Guild, m.Channel, status_code)
	if err != nil {
		log.Fatal(err)
	}

	// Finally commit the transaction
	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}
}

func (db *SqliteDB) IsCached(url string) (bool, int) {

	var status_code int
	err := db.db.QueryRow("SELECT status_code FROM urls WHERE url = ?",
		url).Scan(&status_code)
	switch {
	case err == sql.ErrNoRows:
		return false, status_code
	case err != nil:
		log.Fatal(err)
	}
	return true, status_code
}

func (db *SqliteDB) Stats() (*Stats, error) {

	var stats Stats
	err := db.db.QueryRow(`
	SELECT
	(SELECT COUNT(*) FROM urls),
	(SELECT COUNT(*) FROM users),
	(SELECT COUNT(*) FROM guilds),
	(SELECT COUNT(*) FROM channels)
	;`).Scan(&stats.URLs, &stats.Users, &stats.Guilds, &stats.Channels)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func (db *SqliteDB) ListEntries(limit int, offset int, user string,
	guild string, channel string) (*[]Entry, error) {

	var rows *sql.Rows
	var err error
	if user == "" && guild == "" && channel == "" {
		rows, err = db.db.Query(`
		SELECT urls.id, urls.url, users.user_id, guilds.guild_id, channels.channel_id, status_code
		FROM urls
		INNER JOIN users ON users.id = urls.user_string_id
		INNER JOIN guilds ON guilds.id = urls.guild_string_id
		INNER JOIN channels ON channels.id = urls.channel_string_id
		ORDER BY urls.id DESC
		LIMIT ? OFFSET ?;`, limit, offset)
		if err != nil {
			return nil, err
		}
	} else if user != "" && guild == "" && channel == "" {
		rows, err = db.db.Query(`
		SELECT urls.id, urls.url, users.user_id, guilds.guild_id, channels.channel_id, status_code
		FROM urls
		INNER JOIN users ON users.id = urls.user_string_id
		INNER JOIN guilds ON guilds.id = urls.guild_string_id
		INNER JOIN channels ON channels.id = urls.channel_string_id
		WHERE user_string_id = (SELECT id FROM users WHERE user_id = ?)
		ORDER BY urls.id DESC
		LIMIT ? OFFSET ?;`, user, limit, offset)
		if err != nil {
			return nil, err
		}
	} else if user != "" && guild != "" && channel == "" {
		rows, err = db.db.Query(`
		SELECT urls.id, urls.url, users.user_id, guilds.guild_id, channels.channel_id, status_code
		FROM urls
		INNER JOIN users ON users.id = urls.user_string_id
		INNER JOIN guilds ON guilds.id = urls.guild_string_id
		INNER JOIN channels ON channels.id = urls.channel_string_id
		WHERE user_string_id = (SELECT id FROM users WHERE user_id = ?)
		AND guild_string_id = (SELECT id FROM guilds WHERE guild_id = ?)
		ORDER BY urls.id DESC
		LIMIT ? OFFSET ?;`, user, guild, limit, offset)
		if err != nil {
			return nil, err
		}
	} else if user != "" && guild == "" && channel != "" {
		rows, err = db.db.Query(`
		SELECT urls.id, urls.url, users.user_id, guilds.guild_id, channels.channel_id, status_code
		FROM urls
		INNER JOIN users ON users.id = urls.user_string_id
		INNER JOIN guilds ON guilds.id = urls.guild_string_id
		INNER JOIN channels ON channels.id = urls.channel_string_id
		WHERE user_string_id = (SELECT id FROM users WHERE user_id = ?)
		AND channel_string_id = (SELECT id FROM channels WHERE channel_id = ?)
		ORDER BY urls.id DESC
		LIMIT ? OFFSET ?;`, user, channel, limit, offset)
		if err != nil {
			return nil, err
		}
	} else if user != "" && guild != "" && channel != "" {
		rows, err = db.db.Query(`
		SELECT urls.id, urls.url, users.user_id, guilds.guild_id, channels.channel_id, status_code
		FROM urls
		INNER JOIN users ON users.id = urls.user_string_id
		INNER JOIN guilds ON guilds.id = urls.guild_string_id
		INNER JOIN channels ON channels.id = urls.channel_string_id
		WHERE user_string_id = (SELECT id FROM users WHERE user_id = ?)
		AND guild_string_id = (SELECT id FROM guilds WHERE guild_id = ?)
		AND channel_string_id = (SELECT id FROM channels WHERE channel_id = ?)
		ORDER BY urls.id DESC
		LIMIT ? OFFSET ?;`, user, guild, channel, limit, offset)
		if err != nil {
			return nil, err
		}
	} else if user == "" && guild != "" && channel != "" {
		rows, err = db.db.Query(`
		SELECT urls.id, urls.url, users.user_id, guilds.guild_id, channels.channel_id, status_code
		FROM urls
		INNER JOIN users ON users.id = urls.user_string_id
		INNER JOIN guilds ON guilds.id = urls.guild_string_id
		INNER JOIN channels ON channels.id = urls.channel_string_id
		WHERE guild_string_id = (SELECT id FROM guilds WHERE guild_id = ?)
		AND channel_string_id = (SELECT id FROM channels WHERE channel_id = ?)
		ORDER BY urls.id DESC
		LIMIT ? OFFSET ?;`, guild, channel, limit, offset)
		if err != nil {
			return nil, err
		}
	} else if user == "" && guild == "" && channel != "" {
		rows, err = db.db.Query(`
		SELECT urls.id, urls.url, users.user_id, guilds.guild_id, channels.channel_id, status_code
		FROM urls
		INNER JOIN users ON users.id = urls.user_string_id
		INNER JOIN guilds ON guilds.id = urls.guild_string_id
		INNER JOIN channels ON channels.id = urls.channel_string_id
		WHERE channel_string_id = (SELECT id FROM channels WHERE channel_id = ?)
		ORDER BY urls.id DESC
		LIMIT ? OFFSET ?;`, channel, limit, offset)
		if err != nil {
			return nil, err
		}
	} else if user == "" && guild != "" && channel == "" {
		rows, err = db.db.Query(`
		SELECT urls.id, urls.url, users.user_id, guilds.guild_id, channels.channel_id, status_code
		FROM urls
		INNER JOIN users ON users.id = urls.user_string_id
		INNER JOIN guilds ON guilds.id = urls.guild_string_id
		INNER JOIN channels ON channels.id = urls.channel_string_id
		WHERE guild_string_id = (SELECT id FROM guilds WHERE guild_id = ?)
		ORDER BY urls.id DESC
		LIMIT ? OFFSET ?;`, guild, limit, offset)
		if err != nil {
			return nil, err
		}
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.Message.URL, &e.Message.Author,
			&e.Message.Guild, &e.Message.Channel, &e.Status); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &entries, nil
}
