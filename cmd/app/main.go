package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/whoevenisbranch/branchflower/internal/app"
	"github.com/whoevenisbranch/branchflower/internal/db"
	"github.com/whoevenisbranch/branchflower/internal/repo"
	"github.com/whoevenisbranch/branchflower/internal/service"
	"github.com/whoevenisbranch/branchflower/internal/strava"
)

const baseStravaURL string = "https://www.strava.com/api/v3"

func main() {

	//Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file. Exiting...")
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	dbConn := db.New()
	defer dbConn.Close()

	if err = db.Migrate(dbConn); err != nil {
		log.Printf("Error creating database tables. Exiting...")
		os.Exit(1)
	}

	stravaClient, err := strava.NewStravaClient(baseStravaURL)
	if err != nil {
		log.Fatal(err)
	}

	repository := repo.NewRepo(dbConn)
	service := service.NewService(repository, stravaClient)

	application := app.NewApp(service)

	if err = application.Run(ctx); err != nil {
		log.Printf("application error: %v", err)
		os.Exit(1)
	}
}
