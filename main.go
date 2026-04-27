package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/joho/godotenv"
)

const baseStravaURL string = "https://www.strava.com/api/v3"

func main() {

	//Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	fmt.Println("### Welcome to Branchflower App ###")
	fmt.Println()

	accessToken, err := fetchAccessToken()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	stravaClient, err := NewStravaClient(baseStravaURL, accessToken)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println("Please wait whilst we gather your activities...")

	athlete, err := stravaClient.GetAthlete(context.Background())
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	activities, err := stravaClient.GetAllAthleteActivities(context.Background())

	fmt.Println()

	GreetAthlete(athlete)
	if err == nil {
		count := len(activities)
		if count > 0 {
			fmt.Printf("You have recorded %d activities on Strava!\n", count)
			fmt.Printf("Your first activity recorded was \"%s\"\n", activities[count-1].Name)
			fmt.Printf("Your most recent activity recorded was \"%s\"\n", activities[0].Name)
		} else {
			fmt.Printf("You have no recorded activities")
		}
		return
	}

	if errors.Is(err, ErrStravaAuthError) {
		fmt.Println(err.Error())
		return
	}

	fmt.Println(err.Error())

}

func GreetAthlete(athlete Athlete) {
	greeting := fmt.Sprintf("Welcome %s to Branchflower App!", athlete.FullName)

	if athlete.Username != "" {
		greeting += fmt.Sprintf(" Or should I say %s!", athlete.Username)
	}

	fmt.Println(greeting)
}
