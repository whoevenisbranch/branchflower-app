package database

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	Conn *sql.DB
}

func New() (*DB, error) {

	db, err := sql.Open("sqlite3", "app.db")
	if err != nil {
		return nil, err
	}

	pingErr := db.Ping()
	if pingErr != nil {
		return nil, err
	}

	return &DB{
		Conn: db,
	}, nil
}

func (db *DB) Disconnect() error {
	return db.Conn.Close()
}

func (db *DB) Migrate() error {
	userTable := `
    CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        strava_id INTEGER NOT NULL UNIQUE,
        first_name TEXT NOT NULL,
        created_at DATETIME NOT NULL,
        last_sync_at DATETIME
    );`

	dailyActivityTable := `
    CREATE TABLE IF NOT EXISTS daily_activities (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER NOT NULL,
        date DATETIME NOT NULL,
        activity_count INTEGER NOT NULL,
        moving_time_seconds INTEGER NOT NULL,
        last_updated DATETIME NOT NULL,

        FOREIGN KEY(user_id) REFERENCES users(id),

        UNIQUE(user_id, date) ON CONFLICT REPLACE
    );`

	if _, err := db.Conn.Exec(userTable); err != nil {
		return err
	}

	if _, err := db.Conn.Exec(dailyActivityTable); err != nil {
		return err
	}

	return nil
}
