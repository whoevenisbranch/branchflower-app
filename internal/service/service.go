package service

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	"github.com/whoevenisbranch/branchflower/internal/repo"
	"github.com/whoevenisbranch/branchflower/internal/scoring"
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

func (s *Service) GetUser(ctx context.Context) (*repo.User, error) {
	athlete, err := s.stravaClient.GetAthlete(ctx)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.GetUserByStravaId(ctx, athlete.StravaId)
	if errors.Is(err, sql.ErrNoRows) {
		log.Printf("No existing user exists for strava id = %d, Creating new user...\n", athlete.StravaId)
		user, err = s.repo.CreateUser(ctx, athlete.StravaId, athlete.FirstName)
	}

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Service) SyncActivities(ctx context.Context, userID int) error {
	log.Println("Performing activity sync. Please wait..")

	stravaActivities, err := s.stravaClient.GetAllAthleteActivities(ctx)
	if err != nil {
		return err
	}

	activities := dtoToActivies(stravaActivities)

	runs := make([]repo.Activity, 0, len(activities))
	for _, v := range activities {
		if v.Type == "Run" || v.Type == "TrailRun" || v.Type == "VirtualRun" {
			runs = append(runs, v)
		}
	}

	pendingEntries := normaliseActivities(userID, runs)

	//Returns an error if not every activity is successfully sync'd
	err = s.repo.AddDailyActivities(ctx, pendingEntries)
	if err != nil {
		return err
	}

	err = s.repo.SetUserLastSync(ctx, userID)
	if err != nil {
		return err
	}

	log.Println("Activity sync completed successfully!")
	return nil

}

func (s *Service) GetReport(ctx context.Context, userID int) (Report, error) {
	totalRunDays, err := s.repo.CountTotalActiveDaysById(ctx, userID)
	if err != nil {
		return Report{}, err
	}

	start, end := calculateStatWindow()

	windowActiveDaysMap, err := s.repo.FilterUserActiveDays(ctx, userID, start, end)
	if err != nil {
		return Report{}, err
	}

	orderedWindow := densifyAndOrderDateDescMap(end, windowActiveDaysMap)

	baseScores := scoring.DeriveBaseScores(totalRunDays, orderedWindow)

	return Report{
		BaseScores: baseScores,
	}, nil
}

func calculateStatWindow() (from, to time.Time) {
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	length := scoring.CanopyWindowDays + scoring.BaselineWindowDays
	from = today.AddDate(0, 0, -(length + 1)) //D0 -> D131

	return from, today
}

func densifyAndOrderDateDescMap(end time.Time, sparseMap map[time.Time]repo.DailyAggregate) []repo.DailyAggregate {
	capacity := scoring.CanopyWindowDays + scoring.BaselineWindowDays
	slice := make([]repo.DailyAggregate, 0, capacity)

	for i := range capacity {
		date := end.AddDate(0, 0, -i)
		val, ok := sparseMap[date]

		if !ok {
			val = repo.DailyAggregate{
				ActivityCount:     0,
				MovingTimeSeconds: 0,
			}
		}

		slice = append(slice, val)
	}

	return slice
}

func normaliseActivities(userID int, activities []repo.Activity) map[time.Time]repo.DailyActivity {

	records := make(map[time.Time]repo.DailyActivity)

	for _, activity := range activities {

		year, month, day := activity.StartTimestamp.Date()
		date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

		record, ok := records[date]

		if !ok {
			record = repo.DailyActivity{
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

func dtoToActivies(sa strava.StravaActivitiesDTO) []repo.Activity {

	var bucket []repo.Activity

	for _, activity := range sa {
		bucket = append(bucket, repo.Activity{
			Id:                activity.ID,
			Name:              activity.Name,
			Type:              activity.SportType,
			StartTimestamp:    activity.StartDate,
			MovingTimeSeconds: activity.MovingTimeSeconds,
		})
	}

	return bucket
}
