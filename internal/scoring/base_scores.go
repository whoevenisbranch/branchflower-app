package scoring

import (
	"log"
	"math"

	"github.com/whoevenisbranch/branchflower/internal/repo"
)

const (
	CanopyWindowDays   int = 42
	BaselineWindowDays int = 90
)

const oneHourInSeconds float64 = 3600.0
const canopyActiveDaysCap float64 = 28.0

func DeriveBaseScores(totalRunDays int, bucket []repo.DailyAggregate) BaseScores {

	derived := calculateDerivedAggregates(bucket)
	derived.display()

	var base BaseScores
	base.activeRunDays = totalRunDays
	base = calculateCanopyScores(&derived)

	trendSignal := calculateTrend(&derived)
	treeState := classifyTreeState(base.fullness, base.vitality, trendSignal)

	base.state = treeState

	return base

}

func calculateCanopyScores(derived *derivedAggregates) BaseScores {

	var score BaseScores

	volumeScore := Clamp(derived.currCanopyHrs/max(derived.expectedCanopyHrs, 1.0), 0.0, 2.0)

	consistencyScore := min(float64(derived.currCanopyActiveDays), canopyActiveDaysCap) / canopyActiveDaysCap

	momentum := (derived.recentHalfCanopyHrs - derived.olderHalfCanopyHrs) / max(derived.olderHalfCanopyHrs, 1.0)
	momentumScore := Clamp(momentum, -1.0, 1.0)
	normalisedMomentum := (momentumScore + 1.0) / 2

	canopyScore := (0.45 * volumeScore) + (0.35 * consistencyScore) + (0.2 * normalisedMomentum)

	fullness := Clamp(canopyScore, 0.0, 1.0)

	stability := Clamp(1.0-math.Abs(normalisedMomentum), 0.2, 1.0)

	vitality := (0.6 * consistencyScore) + (0.4 * min(volumeScore, 1.0))

	score.fullness = fullness
	score.stability = stability
	score.vitality = vitality

	return score
}

func calculateDerivedAggregates(bucket []repo.DailyAggregate) derivedAggregates {

	var derived derivedAggregates

	//current canopy
	currentCanopy := bucket[:CanopyWindowDays]

	half := CanopyWindowDays / 2
	//canopy halves
	recentCanopyHalf := currentCanopy[:half]
	olderCanopyHalf := currentCanopy[half:]

	//previous canopy
	previousCanopy := bucket[CanopyWindowDays:(CanopyWindowDays * 2)]

	//baseline window
	baselineWindow := bucket[CanopyWindowDays:]

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
	canopyHoursTrend := Clamp(diffHours/max(derived.prevCanopyHrs, 1.0), -1.0, 1.0)

	diffDays := float64(derived.currCanopyActiveDays - derived.prevCanopyActiveDays)
	canopyActiveDayTrend := Clamp(diffDays/28, -1.0, 1.0)

	trendSignal = (0.7 * canopyHoursTrend) + (0.3 * canopyActiveDayTrend)

	return trendSignal
}

func classifyTreeState(fullness, vitality, trendSignal float64) string {
	var state string

	if fullness < 0.3 && vitality < 0.35 {
		state = "resting"
	} else if fullness >= 0.7 && vitality >= 0.6 && trendSignal >= -0.15 {
		state = "flourishing"
	} else if trendSignal > 0.15 {
		state = "budding"
	} else if trendSignal < -0.15 {
		state = "thinning"
	} else {
		state = "steady"
	}

	return state
}

func Clamp(val, lo, hi float64) float64 {
	if val < lo {
		return lo
	}

	if val > hi {
		return hi
	}

	return val
}

type BaseScores struct {
	activeRunDays int
	fullness      float64
	stability     float64
	vitality      float64
	state         string
}

func (base *BaseScores) Display() {
	log.Printf("Fullness Score: %.2f", base.fullness)
	log.Printf("Stability Score: %.2f", base.stability)
	log.Printf("Vitality Score: %.2f", base.vitality)
	log.Printf("State: %s", base.state)
}

type derivedAggregates struct {
	currCanopyActiveDays int
	currCanopyHrs        float64
	prevCanopyActiveDays int
	prevCanopyHrs        float64
	recentHalfCanopyHrs  float64
	olderHalfCanopyHrs   float64
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
