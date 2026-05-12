package storage

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
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
		conn: db,
	}, nil
}

func (db *DB) Disconnect() error {
	return db.conn.Close()
}
