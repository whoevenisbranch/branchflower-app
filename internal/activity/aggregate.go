package activity

import "math"

const (
	CanopyWindowDays   int = 42
	BaselineWindowDays int = 90
)

const oneHourInSeconds float64 = 3600.0
const canopyActiveDaysCap float64 = 28.0

func deriveBaseScores(totalRunDays int, bucket []DailyAggregate) BaseScores {

	derived := calculateDerivedAggregates(bucket)
	derived.display()

	var base BaseScores
	base.History = clamp(float64(totalRunDays), 0, 1)
	base = calculateCanopyScores(&derived)

	trendSignal := calculateTrend(&derived)
	treeState := classifyTreeState(base.Fullness, base.Vitality, trendSignal)

	base.State = treeState

	return base

}

func calculateCanopyScores(derived *derivedAggregates) BaseScores {

	var score BaseScores

	volumeScore := clamp(derived.currCanopyHrs/max(derived.expectedCanopyHrs, 1.0), 0.0, 2.0)

	consistencyScore := min(float64(derived.currCanopyActiveDays), canopyActiveDaysCap) / canopyActiveDaysCap

	momentum := (derived.recentHalfCanopyHrs - derived.olderHalfCanopyHrs) / max(derived.olderHalfCanopyHrs, 1.0)
	momentumScore := clamp(momentum, -1.0, 1.0)
	normalisedMomentum := (momentumScore + 1.0) / 2

	canopyScore := (0.45 * volumeScore) + (0.35 * consistencyScore) + (0.2 * normalisedMomentum)

	Fullness := clamp(canopyScore, 0.0, 1.0)

	Stability := clamp(1.0-math.Abs(normalisedMomentum), 0.2, 1.0)

	Vitality := (0.6 * consistencyScore) + (0.4 * min(volumeScore, 1.0))

	score.Fullness = Fullness
	score.Stability = Stability
	score.Vitality = Vitality

	return score
}

func calculateDerivedAggregates(bucket []DailyAggregate) derivedAggregates {

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
	canopyHoursTrend := clamp(diffHours/max(derived.prevCanopyHrs, 1.0), -1.0, 1.0)

	diffDays := float64(derived.currCanopyActiveDays - derived.prevCanopyActiveDays)
	canopyActiveDayTrend := clamp(diffDays/28, -1.0, 1.0)

	trendSignal = (0.7 * canopyHoursTrend) + (0.3 * canopyActiveDayTrend)

	return trendSignal
}

func classifyTreeState(Fullness, Vitality, trendSignal float64) string {
	var state string

	if Fullness < 0.3 && Vitality < 0.35 {
		state = "resting"
	} else if Fullness >= 0.7 && Vitality >= 0.6 && trendSignal >= -0.15 {
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

func deriveUIScores(baseScores BaseScores) UIScores {

	var uiScores UIScores

	trunk := getUITrunkValues(baseScores.History)
	canopy := getUICanopyValues(baseScores)

	uiScores.Trunk = trunk
	uiScores.Canopy = canopy
	uiScores.Palette = baseScores.State

	return uiScores
}

func getUITrunkValues(historyScore float64) trunk {

	var t trunk

	t.Height = lerp(80, 180, historyScore)
	t.Width = lerp(16, 36, math.Sqrt(historyScore))

	return t
}

func getUICanopyValues(scores BaseScores) canopy {

	var c canopy

	c.RadiusX = lerp(45, 90, scores.Fullness)
	c.RadiusY = lerp(45, 75, scores.Fullness)

	c.Density = lerp(0.25, 1.0, scores.Fullness)

	c.Smoothness = lerp(0.2, 1.0, scores.Stability)

	c.Saturation = lerp(35, 75, scores.Vitality)
	c.Lightness = lerp(36, 48, scores.Vitality)

	return c

}

func clamp(val, lo, hi float64) float64 {
	if val < lo {
		return lo
	}

	if val > hi {
		return hi
	}

	return val
}

func lerp(start, end, factor float64) float64 {
	return start + (end-start)*factor
}