package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
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

	if errors.Is(err, ErrStravaAuthError) {
		fmt.Println(err.Error())
		return
	}

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println()

	GreetAthlete(athlete)

	records := ProcessNewUserActivityBackfill(athlete.StravaId, activities)
	PrintRecords(records)

	// count := len(activities)
	// if count > 0 {
	// 	fmt.Printf("You have recorded %d activities on Strava!\n", count)
	// 	fmt.Printf("Your first activity recorded was \"%s\"\n", activities[count-1].Name)
	// 	fmt.Printf("Your most recent activity recorded was \"%s\"\n", activities[0].Name)
	// } else {
	// 	fmt.Printf("You have no recorded activities")
	// }

}

func GreetAthlete(athlete Athlete) {
	greeting := fmt.Sprintf("Welcome %s to Branchflower App!", athlete.FullName)

	if athlete.Username != "" {
		greeting += fmt.Sprintf(" Or should I say %s!", athlete.Username)
	}

	fmt.Println(greeting)
}

func ProcessNewUserActivityBackfill(id int, activities []Activity) map[time.Time]DailyActivityRecord {

	totalMovingTime := 0.0
	dailyRecords := make(map[time.Time]DailyActivityRecord)

	for _, activity := range activities {

		start := activity.StartTimestamp

		date := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())

		record, ok := dailyRecords[date]

		if !ok {

			record = DailyActivityRecord{
				StravaId: id,
				Date:     date,
			}
		}

		totalMovingTime += activity.MovingTime

		record.TotalMovingTime += activity.MovingTime
		record.LastUpdatedAt = time.Now().Unix()
		dailyRecords[date] = record
	}

	fmt.Printf("Total Profile moving time: %.2fH\n", totalMovingTime/3600)

	return dailyRecords
}

func PrintRecords(records map[time.Time]DailyActivityRecord) {

	fmt.Printf("Total active days: %d\n", len(records))

	keys := []time.Time{}
	for key := range records {
		keys = append(keys, key)
	}

	// Sort ascending by date
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Before(keys[j])
	})

	for _, key := range keys {
		record := records[key]
		fmt.Printf("Date: %v; TotalMovingTime: %.2f, LastUpdatedAt: %v\n", record.Date, record.TotalMovingTime, record.LastUpdatedAt)
	}
}

type DailyActivityRecord struct {
	StravaId        int
	Date            time.Time
	TotalMovingTime float64 //seconds
	LastUpdatedAt   int64   //the last time the record was updates
}
