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

	athlete, err := stravaClient.GetAthlete(context.Background())
	activities, err := stravaClient.GetAthleteActivities(context.Background())

	fmt.Println()

	if err == nil {
		GreetAthlete(athlete)
		fmt.Println(len(activities))

		if len(activities) > 0 {
			PrintLastUploadedActivity(activities[0])
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

func PrintLastUploadedActivity(activity Activity) {
	fmt.Printf("Last recorded activity was called %s (%d)\n", activity.Name, activity.Id)
}
