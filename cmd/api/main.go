package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/whoevenisbranch/branchflower/internal/oauth"
	"github.com/whoevenisbranch/branchflower/internal/storage"
	"github.com/whoevenisbranch/branchflower/internal/tree"
)

func main() {

	//Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file. Exiting...")
		os.Exit(1)
	}

	//Connect to DB and create tables
	db, err := storage.New()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected!")
	defer db.Disconnect()

	err = storage.MigrateTables(db)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Migration Completed!")

	//Initialise Store
	userStore := storage.NewUserRepository(db)
	activityStore := storage.NewActivityRepository(db)

	//Initialise Service
	treeService := tree.NewService(userStore, activityStore)

	//Handlers

	authConfig := oauth.Config{
		ClientID:     os.Getenv("STRAVA_OAUTH_CLIENT_ID"),
		ClientSecret: os.Getenv("STRAVA_OAUTH_CLIENT_SECRET"),
		CallbackURL:  os.Getenv("CALLBACK_URL"),
	}
	authHandler := oauth.NewOAuthHandler(authConfig)

	treeHandler := tree.NewHandler(treeService)

	//Start HTTP server
	mux := http.NewServeMux()
	mux.Handle("/tree", authHandler.Protected(treeHandler.Home))

	mux.HandleFunc("/oauth/callback", authHandler.HandleOAuthRedirect)

	log.Println("Staring server on port :8080")
	log.Fatal(http.ListenAndServe(":8085", mux))

}
