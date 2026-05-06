package service

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	"github.com/whoevenisbranch/branchflower/internal/models"
	"github.com/whoevenisbranch/branchflower/internal/repo"
	"github.com/whoevenisbranch/branchflower/internal/strava"
)

type Service struct {
	repo         repo.Repo
	stravaClient *strava.StravaClient
}

func NewService(repo repo.Repo, stravaClient *strava.StravaClient) Service {
	return Service{
		repo:         repo,
		stravaClient: stravaClient,
	}
}

func (s *Service) GetUser(ctx context.Context) (*models.User, error) {
	athlete, err := s.stravaClient.GetAthlete(ctx)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.GetUserByStravaId(ctx, athlete.StravaId)
	if errors.Is(err, sql.ErrNoRows) {
		log.Printf("No existing user exists for strava id = %d, Creating new user...\n", athlete.StravaId)
		user, err = s.repo.CreateUser(ctx, athlete)
	}

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Service) SyncActivities(ctx context.Context, user models.User) error {
	log.Println("Performing activity sync. Please wait..")

	activities, err := s.stravaClient.GetAllAthleteActivities(ctx)
	if err != nil {
		return err
	}

	pendingEntries := normaliseActivities(user.ID, activities)

	//Returns an error if not every activity is successfully sync'd
	err = s.repo.AddDailyActivities(ctx, pendingEntries)
	if err != nil {
		return err
	}

	s.repo.SetUserLastSync(ctx, user.ID)
	log.Println("Activity sync completed successfully!")
	return nil

}

func (s *Service) GetReport(ctx context.Context, user models.User) error {
	total, err := s.repo.CountTotalActiveDaysById(ctx, user.ID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	from := time.Date(now.Year(), now.Month(), now.Day()-7, 0, 0, 0, 0, time.UTC)
	to := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	weeklyActiveDays, err := s.repo.FilterUserActiveDays(ctx, user.ID, from, to)

	if err != nil {
		return err
	}

	log.Printf("Total Active Days = %d", total)

	weeklyTotal := len(weeklyActiveDays)
	log.Printf("Active days in the last week = %d", weeklyTotal)

	if weeklyTotal > 0 {
		log.Printf("Last active day = %s", weeklyActiveDays[len(weeklyActiveDays)-1].Date)
	}

	return nil
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
