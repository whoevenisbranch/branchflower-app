package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
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

	db, err := Connect("app.db")
	if err != nil {
		log.Fatal(err)
		return
	}
	defer db.Close()

	if err = CreateTables(db); err != nil {
		log.Fatal(err)
		return
	}

	repo := NewRepo(db)

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

	athlete, err := stravaClient.GetAthlete(context.Background())
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	user, err := repo.GetUserByStravaId(context.Background(), athlete.StravaId)
	if errors.Is(err, sql.ErrNoRows) {
		fmt.Printf("No existing user exists for strava id = %d, Creating new user...\n", athlete.StravaId)
		user, err = repo.CreateUser(context.Background(), athlete)
	}

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	user.Greet()

	if user.LastSyncAt == nil {
		fmt.Println("Performing activity sync. Please wait..")

		activities, err := stravaClient.GetAllAthleteActivities(context.Background())
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		pendingEntries := normaliseActivities(user.ID, activities)

		//Returns an error if not every activity is successfully sync'd
		err = repo.AddDailyActivities(context.Background(), pendingEntries)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		repo.SetUserLastSync(context.Background(), user.ID)
		fmt.Println("Activity sync completed successfully!")
	}

	fmt.Println(repo.CountTotalActiveDaysById(context.Background(), user.ID))

	now := time.Now().UTC()
	from := time.Date(now.Year(), now.Month(), now.Day()-7, 0, 0, 0, 0, time.UTC)
	to := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	weeklyActiveDays, err := repo.FilterUserActiveDays(context.Background(), user.ID, from, to)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println(len(weeklyActiveDays))
	fmt.Println(weeklyActiveDays[len(weeklyActiveDays)-1].MovingTimeSeconds)
}

func normaliseActivities(userID int, activities []Activity) map[time.Time]DailyActivity {

	records := make(map[time.Time]DailyActivity)

	for _, activity := range activities {

		year, month, day := activity.StartTimestamp.Date()
		date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

		record, ok := records[date]

		if !ok {
			record = DailyActivity{
				UserID: userID,
				Date:   date,
			}
		}

		record.MovingTimeSeconds += activity.MovingTimeSeconds
		record.ActivityCount++
		records[date] = record
	}

	return records
}
