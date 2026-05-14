package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/whoevenisbranch/branchflower/internal/activity"
	"github.com/whoevenisbranch/branchflower/internal/auth"
	"github.com/whoevenisbranch/branchflower/internal/database"
	"github.com/whoevenisbranch/branchflower/internal/user"

	_ "github.com/mattn/go-sqlite3"
)

func main() {

	//Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file. Exiting...")
	}

	db, err := configureDatabase()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Disconnect()

	//Configure repositories
	userRepository := user.NewRepository(db)
	activityRepository := activity.NewRepository(db)

	//Configure services
	cfg := auth.StravaOAuthConfig{
		ClientId:     os.Getenv("STRAVA_OAUTH_CLIENT_ID"),
		ClientSecret: os.Getenv("STRAVA_OAUTH_CLIENT_SECRET"),
		CallbackURL:  os.Getenv("CALLBACK_URL"),
	}
	authService := auth.NewOAuthService(cfg)

	userService := user.NewService(userRepository)
	activityService := activity.NewService(activityRepository)

	//Handlers

	authHandler := auth.NewAuthRedirectHandler(authService)

	treeHandler := activity.NewHandler(userService, activityService, authService)

	//Start HTTP server
	http.HandleFunc("/protected", treeHandler.Handle)
	http.HandleFunc("/oauth/callback", authHandler.HandleOAuthRedirect)

	log.Println("Staring server on port :8085")
	log.Fatal(http.ListenAndServe(":8085", nil))

}

func configureDatabase() (*database.DB, error) {

	var err error
	var db *database.DB

	db, err = database.New()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected!")

	err = db.Migrate()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Migration Completed!")

	return db, err
}
