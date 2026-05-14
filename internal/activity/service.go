package activity

import (
	"context"
	"log"
	"time"

	"github.com/whoevenisbranch/branchflower/internal/strava"
)

type ActivityService struct {
	store ActivityRepository
}

func NewService(repo ActivityRepository) ActivityService {
	return ActivityService{
		store: repo,
	}
}

func (svc *ActivityService) GetUserTreeData(ctx context.Context, userID int) (TreeData, error) {
	totalRunDays, err := svc.store.CountTotalActiveDaysById(ctx, userID)
	if err != nil {
		return TreeData{}, err
	}

	start, end := calculateStatWindow()

	windowActiveDaysMap, err := svc.store.FilterActiveDaysByUserID(ctx, userID, start, end)
	if err != nil {
		return TreeData{}, err
	}

	orderedWindow := densifyAndOrderDateDescMap(end, windowActiveDaysMap)

	baseScores := deriveBaseScores(totalRunDays, orderedWindow)
	uiScores := deriveUIScores(baseScores)

	return TreeData{
		OwnerID:     userID,
		BaseScores:  baseScores,
		UIScores:    uiScores,
		GeneratedAt: time.Now(),
	}, nil
}

func (svc *ActivityService) SyncActivities(ctx context.Context, userID int, token string) error {
	log.Println("Performing activity sync. Please wait..")

	client, err := strava.NewStravaClient(token)
	if err != nil {
		return err
	}

	stravaActivities, err := client.GetAllAthleteActivities(ctx)
	if err != nil {
		return err
	}

	activities := stravaActivities.ToActivites()

	runs := make([]strava.Activity, 0, len(activities))
	for _, v := range activities {
		if v.Type == "Run" || v.Type == "TrailRun" || v.Type == "VirtualRun" {
			runs = append(runs, v)
		}
	}

	pendingEntries := normaliseActivities(userID, runs)

	//Returns an error if not every activity is successfully sync'd
	err = svc.store.AddDailyActivities(ctx, pendingEntries)
	if err != nil {
		return err
	}

	log.Println("Activity sync completed successfully!")
	return nil

}

func normaliseActivities(userID int, activities []strava.Activity) map[time.Time]DailyActivity {

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

func calculateStatWindow() (from, to time.Time) {
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	length := CanopyWindowDays + BaselineWindowDays
	from = today.AddDate(0, 0, -(length + 1)) //D0 -> D131

	return from, today
}

func densifyAndOrderDateDescMap(end time.Time, sparseMap map[time.Time]DailyAggregate) []DailyAggregate {
	capacity := CanopyWindowDays + BaselineWindowDays
	slice := make([]DailyAggregate, 0, capacity)

	for i := range capacity {
		date := end.AddDate(0, 0, -i)
		val, ok := sparseMap[date]

		if !ok {
			val = DailyAggregate{
				ActivityCount:     0,
				MovingTimeSeconds: 0,
			}
		}

		slice = append(slice, val)
	}

	return slice
}
