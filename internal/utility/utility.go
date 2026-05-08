package utility

import (
	"log"
	"time"
)

func TimeCheck(message string, start time.Time) {
	elapsed := time.Since(start).Seconds()
	log.Printf("%s took %.2fs\n", message, elapsed)
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
