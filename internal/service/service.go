package service

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"math"
	"time"

	"github.com/whoevenisbranch/branchflower/internal/models"
	"github.com/whoevenisbranch/branchflower/internal/repo"
	"github.com/whoevenisbranch/branchflower/internal/strava"
	"github.com/whoevenisbranch/branchflower/internal/utility"
)

const canopyWindowDays int = 42
const baselineWindowDays int = 90
const oneHourInSeconds float64 = 3600.0
const canopyActiveDaysCap float64 = 28.0

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

	runs := make([]models.Activity, 0, len(activities))
	for _, v := range activities {
		if v.Type == "Run" || v.Type == "TrailRun" || v.Type == "VirtualRun" {
			runs = append(runs, v)
		}
	}

	pendingEntries := normaliseActivities(user.ID, runs)

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
	totalActiveDays, err := s.repo.CountTotalActiveDaysById(ctx, user.ID)
	if err != nil {
		return err
	}
	log.Printf("Total Active Days (Run) = %d\n", totalActiveDays)

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	length := canopyWindowDays + baselineWindowDays
	from := today.AddDate(0, 0, -(length + 1)) //length + 1 to include today

	activeDaysMap, err := s.repo.FilterUserActiveDays(ctx, user.ID, from, today)
	if err != nil {
		return err
	}

	orderedDenseSlice := densifyAndOrderDateDescMap(today, activeDaysMap)

	totalSeconds := 0
	for _, entry := range orderedDenseSlice {
		totalSeconds += entry.MovingTimeSeconds
	}
	totalHours := float64(totalSeconds) / oneHourInSeconds
	log.Printf("Total Window Hours = %.2f", totalHours)

	deriveTreeStats(orderedDenseSlice)

	return nil
}

func deriveTreeStats(bucket []repo.DailyAggregate) {

	derived := calculateDerivedAggregates(bucket)
	derived.display()

	canopyScores := calculateCanopyScores(&derived)
	canopyScores.display()

	trendSignal := calculateTrend(&derived)
	log.Printf("Trend Signal: %.2f", trendSignal)

	season := classifySeason(canopyScores.fullness, canopyScores.vitality, trendSignal)
	log.Printf("Derived Season: %s", season)
}

func calculateCanopyScores(derived *derivedAggregates) canopyScore {

	var score canopyScore

	volumeScore := utility.Clamp(derived.currCanopyHrs/max(derived.expectedCanopyHrs, 1.0), 0.0, 2.0)

	consistencyScore := min(float64(derived.currCanopyActiveDays), canopyActiveDaysCap) / canopyActiveDaysCap

	momentum := (derived.recentHalfCanopyHrs - derived.olderHalfCanopyHrs) / max(derived.olderHalfCanopyHrs, 1.0)
	momentumScore := utility.Clamp(momentum, -1.0, 1.0)
	normalisedMomentum := (momentumScore + 1.0) / 2

	canopyScore := (0.45 * volumeScore) + (0.35 * consistencyScore) + (0.2 * normalisedMomentum)

	fullness := utility.Clamp(canopyScore, 0.0, 1.0)

	stability := utility.Clamp(1.0-math.Abs(normalisedMomentum), 0.2, 1.0)

	vitality := (0.6 * consistencyScore) + (0.4 * min(volumeScore, 1.0))

	score.overall = canopyScore
	score.volume = volumeScore
	score.consistency = consistencyScore
	score.momentum = normalisedMomentum
	score.fullness = fullness
	score.stability = stability
	score.vitality = vitality

	return score
}

func calculateDerivedAggregates(bucket []repo.DailyAggregate) derivedAggregates {

	var derived derivedAggregates

	//current canopy
	currentCanopy := bucket[:canopyWindowDays]

	half := canopyWindowDays / 2
	//canopy halves
	recentCanopyHalf := currentCanopy[:half]
	olderCanopyHalf := currentCanopy[half:]

	//previous canopy
	previousCanopy := bucket[canopyWindowDays:(canopyWindowDays * 2)]

	//baseline window
	baselineWindow := bucket[canopyWindowDays:]

	currentCanopyActiveDays := 0
	currentCanopyWindowHours := 0.0
	for _, v := range currentCanopy {
		if v.MovingTimeSeconds > 0 {
			currentCanopyActiveDays++
			currentCanopyWindowHours += float64(v.MovingTimeSeconds) / oneHourInSeconds
		}
	}

	recentHalfCanopyHours := 0.0
	for _, v := range recentCanopyHalf {
		recentHalfCanopyHours += float64(v.MovingTimeSeconds) / oneHourInSeconds
	}

	olderHalfCanopyHours := 0.0
	for _, v := range olderCanopyHalf {
		olderHalfCanopyHours += float64(v.MovingTimeSeconds) / oneHourInSeconds
	}

	baselineTotalHours := 0.0
	for _, v := range baselineWindow {
		baselineTotalHours += float64(v.MovingTimeSeconds) / 3600
	}

	baselineAvgDailyHours := baselineTotalHours / 90
	expectedCanopyHours := baselineAvgDailyHours * 42

	prevCanopyActiveDays := 0
	prevCanopyWindowHours := 0.0
	for _, v := range previousCanopy {
		if v.MovingTimeSeconds > 0 {
			prevCanopyActiveDays++
			prevCanopyWindowHours += float64(v.MovingTimeSeconds) / oneHourInSeconds
		}
	}

	derived.currCanopyActiveDays = currentCanopyActiveDays
	derived.currCanopyHrs = currentCanopyWindowHours
	derived.prevCanopyActiveDays = prevCanopyActiveDays
	derived.prevCanopyHrs = prevCanopyWindowHours
	derived.recentHalfCanopyHrs = recentHalfCanopyHours
	derived.olderHalfCanopyHrs = olderHalfCanopyHours
	derived.baselineAvgDailyHrs = baselineAvgDailyHours
	derived.expectedCanopyHrs = expectedCanopyHours

	return derived

}

func calculateTrend(derived *derivedAggregates) float64 {

	var trendSignal float64

	diffHours := derived.currCanopyHrs - derived.prevCanopyHrs
	canopyHoursTrend := utility.Clamp(diffHours/max(derived.prevCanopyHrs, 1.0), -1.0, 1.0)

	diffDays := float64(derived.currCanopyActiveDays - derived.prevCanopyActiveDays)
	canopyActiveDayTrend := utility.Clamp(diffDays/28, -1.0, 1.0)

	trendSignal = (0.7 * canopyHoursTrend) + (0.3 * canopyActiveDayTrend)

	return trendSignal
}

func classifySeason(fullness, vitality, trendSignal float64) string {
	var season string
	if fullness > 0.3 && vitality < 0.35 {
		season = "winter"
	} else if trendSignal > 0.15 {
		season = "spring"
	} else if fullness >= 0.7 && vitality >= 0.6 && trendSignal >= -0.15 {
		season = "summer"
	} else {
		season = "autumn"
	}

	return season
}

func densifyAndOrderDateDescMap(end time.Time, sparseMap map[time.Time]repo.DailyAggregate) []repo.DailyAggregate {
	capacity := canopyWindowDays + baselineWindowDays
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

type canopyScore struct {
	overall     float64
	volume      float64
	consistency float64
	momentum    float64
	fullness    float64
	stability   float64
	vitality    float64
}

func (cs *canopyScore) display() {
	log.Printf("Canopy Volume Score: %.2f", cs.volume)
	log.Printf("Canopy Consistency Score: %.2f", cs.consistency)
	log.Printf("Canopy Momentum Score: %.2f", cs.momentum)
	log.Printf("Canopy Score = %.2f", cs.overall)
	log.Printf("Fullness Score: %.2f", cs.fullness)
	log.Printf("Stability Score: %.2f", cs.stability)
	log.Printf("Vitality Score: %.2f", cs.vitality)
}

type derivedAggregates struct {
	currCanopyActiveDays int
	currCanopyHrs        float64
	prevCanopyActiveDays int
	prevCanopyHrs        float64
	recentHalfCanopyHrs   float64
	olderHalfCanopyHrs  float64
	baselineAvgDailyHrs  float64
	expectedCanopyHrs    float64
}

func (da *derivedAggregates) display() {
	log.Printf("Active Days in Current Canopy Window = %d", da.currCanopyActiveDays)
	log.Printf("Active Days in Previous Canopy Window = %d", da.prevCanopyActiveDays)

	log.Printf("Baseline Average Daily Hours = %.2f", da.baselineAvgDailyHrs)
	log.Printf("Expected Canopy Hours = %.2f", da.expectedCanopyHrs)
	log.Printf("Current Canopy Hours = %.2f [1st: %.2f, 2nd:%.2f]", da.currCanopyHrs, da.recentHalfCanopyHrs, da.olderHalfCanopyHrs)
	log.Printf("Previous Canopy Hours = %.2f", da.prevCanopyHrs)
}
