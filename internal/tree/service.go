package tree

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	"github.com/whoevenisbranch/branchflower/internal/storage"
	"github.com/whoevenisbranch/branchflower/internal/strava"
)

type Service struct {
	userRepo     *storage.UserRepository
	activityRepo *storage.ActivityRepository
}

func NewService(u *storage.UserRepository, a *storage.ActivityRepository) *Service {
	return &Service{
		userRepo:     u,
		activityRepo: a,
	}
}

func (s *Service) GetUser(ctx context.Context) (*storage.User, error) {

	stravaClient, err := getAuthenticatedStravaClient()
	if err != nil {
		return nil, err
	}

	athlete, err := stravaClient.GetAthlete(ctx)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetUserByStravaId(ctx, athlete.StravaId)
	if errors.Is(err, sql.ErrNoRows) {
		log.Printf("No existing user exists for strava id = %d, Creating new user...\n", athlete.StravaId)
		user, err = s.userRepo.CreateUser(ctx, athlete.StravaId, athlete.FirstName)
	}

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Service) SyncActivities(ctx context.Context, userID int) error {
	log.Println("Performing activity sync. Please wait..")

	stravaClient, err := getAuthenticatedStravaClient()
	if err != nil {
		return err
	}

	stravaActivities, err := stravaClient.GetAllAthleteActivities(ctx)
	if err != nil {
		return err
	}

	activities := dtoToActivies(stravaActivities)

	runs := make([]storage.Activity, 0, len(activities))
	for _, v := range activities {
		if v.Type == "Run" || v.Type == "TrailRun" || v.Type == "VirtualRun" {
			runs = append(runs, v)
		}
	}

	pendingEntries := normaliseActivities(userID, runs)

	//Returns an error if not every activity is successfully sync'd
	err = s.activityRepo.AddDailyActivities(ctx, pendingEntries)
	if err != nil {
		return err
	}

	err = s.userRepo.SetUserLastSync(ctx, userID)
	if err != nil {
		return err
	}

	log.Println("Activity sync completed successfully!")
	return nil

}

func (s *Service) GetUserTreeData(ctx context.Context, userID int) (TreeData, error) {
	totalRunDays, err := s.activityRepo.CountTotalActiveDaysById(ctx, userID)
	if err != nil {
		return TreeData{}, err
	}

	start, end := calculateStatWindow()

	windowActiveDaysMap, err := s.activityRepo.FilterActiveDaysByUserID(ctx, userID, start, end)
	if err != nil {
		return TreeData{}, err
	}

	orderedWindow := densifyAndOrderDateDescMap(end, windowActiveDaysMap)

	baseScores := deriveBaseScores(totalRunDays, orderedWindow)
	uiScores := deriveUIScores(baseScores)

	return TreeData{
		BaseScores: baseScores,
		UIScores:   uiScores,
	}, nil
}

func calculateStatWindow() (from, to time.Time) {
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	length := CanopyWindowDays + BaselineWindowDays
	from = today.AddDate(0, 0, -(length + 1)) //D0 -> D131

	return from, today
}

func densifyAndOrderDateDescMap(end time.Time, sparseMap map[time.Time]storage.DailyAggregate) []storage.DailyAggregate {
	capacity := CanopyWindowDays + BaselineWindowDays
	slice := make([]storage.DailyAggregate, 0, capacity)

	for i := range capacity {
		date := end.AddDate(0, 0, -i)
		val, ok := sparseMap[date]

		if !ok {
			val = storage.DailyAggregate{
				ActivityCount:     0,
				MovingTimeSeconds: 0,
			}
		}

		slice = append(slice, val)
	}

	return slice
}

func normaliseActivities(userID int, activities []storage.Activity) map[time.Time]storage.DailyActivity {

	records := make(map[time.Time]storage.DailyActivity)

	for _, activity := range activities {

		year, month, day := activity.StartTimestamp.Date()
		date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

		record, ok := records[date]

		if !ok {
			record = storage.DailyActivity{
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

func dtoToActivies(sa strava.StravaActivitiesDTO) []storage.Activity {

	var bucket []storage.Activity

	for _, activity := range sa {
		bucket = append(bucket, storage.Activity{
			Id:                activity.ID,
			Name:              activity.Name,
			Type:              activity.SportType,
			StartTimestamp:    activity.StartDate,
			MovingTimeSeconds: activity.MovingTimeSeconds,
		})
	}

	return bucket
}

func getAuthenticatedStravaClient() (strava.StravaClient, error) {
	// accessToken, err := oauth.FetchAccessToken()
	// if err != nil {
	// 	return strava.StravaClient{}, err
	// }
	// stravaClient, err := strava.NewStravaClient(accessToken)
	// if err != nil {
	// 	return strava.StravaClient{}, err
	// }

	return strava.StravaClient{}, nil
}
