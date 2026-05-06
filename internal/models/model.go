package models

import (
	"fmt"
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
	greeting := fmt.Sprintf("Welcome %s to Branchflower App!", u.FirstName)
	fmt.Println(greeting)
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
	StartTimestamp    time.Time
	MovingTimeSeconds int
}
