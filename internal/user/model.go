package user

import "time"

type User struct {
	ID         int
	StravaID   int
	FirstName  string
	CreatedAt  time.Time
	LastSyncAt *time.Time
}
