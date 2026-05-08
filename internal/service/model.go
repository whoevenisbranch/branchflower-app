package service

import (
	"time"

	"github.com/whoevenisbranch/branchflower/internal/scoring"
)

type TreeData struct {
	BaseScores  scoring.BaseScores
	UIScores    scoring.UIScores
	GeneratedAt time.Time
}
