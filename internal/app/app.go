package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/whoevenisbranch/branchflower/internal/models"
	"github.com/whoevenisbranch/branchflower/internal/oauth"
	"github.com/whoevenisbranch/branchflower/internal/repo"
	"github.com/whoevenisbranch/branchflower/internal/strava"
)

type app struct {
	repository repo.Repo
}

const baseStravaURL string = "https://www.strava.com/api/v3"

var stravaClient *strava.StravaClient

func NewApp(repository repo.Repo) app {
	return app{
		repository: repository,
	}
}

func (a *app) Run() {
	fmt.Println("### Welcome to Branchflower App ###")
	fmt.Println()

	accessToken, err := oauth.FetchAccessToken()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	stravaClient, err = strava.NewStravaClient(baseStravaURL, accessToken)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	athlete, err := stravaClient.GetAthlete(context.Background())
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	user, err := a.repository.GetUserByStravaId(context.Background(), athlete.StravaId)
	if errors.Is(err, sql.ErrNoRows) {
		fmt.Printf("No existing user exists for strava id = %d, Creating new user...\n", athlete.StravaId)
		user, err = a.repository.CreateUser(context.Background(), athlete)
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
		err = a.repository.AddDailyActivities(context.Background(), pendingEntries)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		a.repository.SetUserLastSync(context.Background(), user.ID)
		fmt.Println("Activity sync completed successfully!")
	}

	fmt.Println(a.repository.CountTotalActiveDaysById(context.Background(), user.ID))

	now := time.Now().UTC()
	from := time.Date(now.Year(), now.Month(), now.Day()-7, 0, 0, 0, 0, time.UTC)
	to := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	weeklyActiveDays, err := a.repository.FilterUserActiveDays(context.Background(), user.ID, from, to)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println(len(weeklyActiveDays))
	fmt.Println(weeklyActiveDays[len(weeklyActiveDays)-1].MovingTimeSeconds)
}

func normaliseActivities(userID int, activities []models.Activity) map[time.Time]models.DailyActivity {

	records := make(map[time.Time]models.DailyActivity)

	for _, activity := range activities {

		year, month, day := activity.StartTimestamp.Date()
		date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

		record, ok := records[date]

		if !ok {
			record = models.DailyActivity{
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
