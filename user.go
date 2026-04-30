package main

import (
	"fmt"
	"math"
)

const thresholdWeeklyMinutes = 150
const deltaThreshold = 0.15

func calculate(snapshot map[int]SnapshotWeek) {

	var totalBaselineMinutes float64 = 0.0
	var totalRecentMinutes float64 = 0.0

	for i := 1; i <= 8; i++ {
		snapshot, ok := snapshot[i]
		if !ok {
			fmt.Printf("No activities recorded %d ago\n", i)
			continue
		}

		snapshotMins := snapshot.TotalMovingTimeSec / 60
		if i <= 4 {
			totalBaselineMinutes += snapshotMins
		} else {
			totalRecentMinutes += snapshotMins
		}
	}

	meanBaselineMinutes := totalBaselineMinutes / 4
	meanRecentMinutes := totalRecentMinutes / 4

	baselineScore := math.Min(meanBaselineMinutes/thresholdWeeklyMinutes, 1.0)
	recentScore := math.Min(meanRecentMinutes/thresholdWeeklyMinutes, 1.0)

	fmt.Printf("Mean baseline score: %.2f\n", baselineScore)
	fmt.Printf("Mean recent score: %.2f\n", recentScore)

	delta := recentScore - baselineScore

	fmt.Printf("Delta: %.2f\n", delta)

	baselineAdherence := meanBaselineMinutes >= thresholdWeeklyMinutes
	recentAdherence := meanRecentMinutes >= thresholdWeeklyMinutes

	var context string
	switch {

	case delta >= deltaThreshold: //Upward trend

		if !baselineAdherence && recentAdherence { //Recent improvement
			context = "Aligning to Threshold"
		} else {
			context = "Building to Threshold"
		}

	case delta <= -deltaThreshold: //Downward trend

		if baselineAdherence && !recentAdherence { //Recent regression
			context = "Recent Drop-Off"
		} else {
			context = "Continued Decline"
		}

	default: //Stable trend

		if baselineAdherence && recentAdherence { //Stable and above/at threshold
			context = "Maintained At/Above Threshold"
		} else {
			context = "Below Threshold"
		}

	}

	fmt.Printf("Context: %s\n", context)

}
