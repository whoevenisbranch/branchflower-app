package repo

import (
	"log"
	"time"
)

type User struct {
	ID         int
	StravaID   int
	FirstName  string
	CreatedAt  time.Time
	LastSyncAt *time.Time
}

func (u *User) Greet() {
	log.Printf("Welcome %s to Branchflower App!", u.FirstName)
}

type DailyActivity struct {
	ID                int
	UserID            int
	Date              time.Time
	ActivityCount     int
	MovingTimeSeconds int
	LastUpdatedAt     time.Time
}

type Activity struct {
	Id                int64
	Name              string
	Type              string
	StartTimestamp    time.Time
	MovingTimeSeconds int
}

type DailyAggregate struct {
	ActivityCount     int
	MovingTimeSeconds int
}
