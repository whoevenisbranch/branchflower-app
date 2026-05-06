package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/whoevenisbranch/branchflower/internal/app"
	"github.com/whoevenisbranch/branchflower/internal/db"
	"github.com/whoevenisbranch/branchflower/internal/repo"
)

func main() {

	//Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	database := db.New()
	defer database.Close()

	if err = db.Migrate(database); err != nil {
		log.Fatal(err)
	}

	repository := repo.NewRepo(database)
	application := app.NewApp(repository)

	application.Run()

}
