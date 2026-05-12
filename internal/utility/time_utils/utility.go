package timeutils

import (
	"log"
	"time"
)

func TimeCheck(message string, start time.Time) {
	elapsed := time.Since(start).Seconds()
	log.Printf("%s took %.2fs\n", message, elapsed)
}
