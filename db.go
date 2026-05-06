package main

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

func Connect(path string) (*sql.DB, error) {

	var db *sql.DB
	var err error

	db, err = sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	pingErr := db.Ping()
	if pingErr != nil {
		return nil, err
	}

	fmt.Println("Connected!")

	return db, nil
}

func CreateTables(db *sql.DB) error {
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

	if _, err := db.Exec(userTable); err != nil {
		return err
	}

	if _, err := db.Exec(dailyActivityTable); err != nil {
		return err
	}

	fmt.Println("Tables initialised successfully")
	return nil
}
