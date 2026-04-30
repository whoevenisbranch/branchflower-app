package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/joho/godotenv"
)

const baseStravaURL string = "https://www.strava.com/api/v3"

func main() {

	//Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	fmt.Printf("Current Week Start: %s\n", getWeekStartISO(time.Now()))

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

	fmt.Println("Successfully created Stava Client ...")

	fmt.Println("Requesting Strava athlete profile ...")
	athlete, err := stravaClient.GetAthlete(context.Background())
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	GreetAthlete(athlete)

	err = backfillNewUser(stravaClient, &athlete)
	fmt.Println()

	if err != nil {
		if errors.Is(err, ErrStravaAuthError) {
			fmt.Println(err.Error())
			return
		}
		fmt.Println(err.Error())
	}

	fmt.Printf("Total recorded runs = %d\n", athlete.TotalRuns)

	fmt.Println()

	calculate(athlete.RunSnapshot)
}

func GreetAthlete(athlete Athlete) {
	greeting := fmt.Sprintf("Welcome %s to Branchflower App!", athlete.FullName)

	if athlete.Username != "" {
		greeting += fmt.Sprintf(" Or should I say %s!", athlete.Username)
	}

	fmt.Println(greeting)
}

//Utility

func timeCheck(start time.Time) {
	elapsed := time.Since(start).Seconds()
	fmt.Printf(" # Completed in %.2fs\n", elapsed)
}

func getWeekStartISO(t time.Time) time.Time {
	weekday := int(t.Weekday())

	//Make week start on Monday not Sunday
	if weekday == 0 {
		weekday = 7
	}

	start := t.AddDate(0, 0, -(weekday - 1))
	return time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
}

func filterRuns(activities []Activity) (int, []Activity) {

	currentWeekStart := getWeekStartISO(time.Now())
	windowWeekStart := currentWeekStart.AddDate(0, 0, -(8 * 7))

	runCount := 0

	bucket := make([]Activity, 0, len(activities))
	var runType = []string{"Run", "TrailRun", "VirtualRun"}

	for _, activity := range activities {
		//Filter run activities that are greater than or equal to 10 mins of moving time
		if slices.Contains(runType, activity.Type) && activity.Time/60 >= 10 {

			runCount++

			//only keep those that are in the prev.8 week window
			if !activity.LocalStartTime.Before(windowWeekStart) && !activity.LocalStartTime.After(currentWeekStart) {
				bucket = append(bucket, activity)
			}
		}
	}

	return runCount, bucket
}

func backfillNewUser(sc *StravaClient, athlete *Athlete) error {
	activites, err := sc.GetAthleteActivities(context.Background(), getWeekStartISO(time.Now()).Unix(), 0)
	if err != nil {
		return err
	}

	count, filtered := filterRuns(activites)

	athlete.TotalRuns = count

	athlete.RunSnapshot = extractSnapshotRuns(filtered)
	return nil

}

func extractSnapshotRuns(runs []Activity) map[int]SnapshotWeek {

	weeksAgoToRunSnapshot := make(map[int]SnapshotWeek)

	curr := getWeekStartISO(time.Now())

	for _, run := range runs {
		weekStartRun := getWeekStartISO(run.LocalStartTime)

		diff := curr.Sub(weekStartRun)
		weeksAgo := int(diff.Hours() / (24 * 7))

		snapshot, ok := weeksAgoToRunSnapshot[weeksAgo]

		if !ok {
			weeksAgoToRunSnapshot[weeksAgo] = SnapshotWeek{}
		}

		snapshot.TotalDistanceM += run.Distance
		snapshot.TotalMovingTimeSec += run.Time
		snapshot.Activities = append(snapshot.Activities, run)

		weeksAgoToRunSnapshot[weeksAgo] = snapshot

	}

	return weeksAgoToRunSnapshot
}

type SnapshotWeek struct {
	TotalDistanceM     float64
	TotalMovingTimeSec float64
	Activities         []Activity
}
