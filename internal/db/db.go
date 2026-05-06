package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func New() *sql.DB {

	var db *sql.DB
	var err error

	db, err = sql.Open("sqlite3", "app.db")
	if err != nil {
		log.Fatal(fmt.Errorf("Failed to connect to database"))
	}

	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(fmt.Errorf("Failed to connect to database"))
	}

	fmt.Println("Connected!")

	return db
}
