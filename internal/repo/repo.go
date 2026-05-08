package repo

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/whoevenisbranch/branchflower/internal/models"
	"github.com/whoevenisbranch/branchflower/internal/strava"
	"github.com/whoevenisbranch/branchflower/internal/utility"
)

//
// Repo
//

type Repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) Repo {
	return Repo{db: db}
}

// User Queries
func (r *Repo) CreateUser(ctx context.Context, athlete strava.Athlete) (*models.User, error) {
	defer utility.TimeCheck("repo.CreateUser", time.Now())

	var err error

	now := time.Now().UTC()

	result, err := r.db.ExecContext(ctx,
		`INSERT INTO users (strava_id, first_name, created_at)
		VALUES (?, ?, ?)`, athlete.StravaId, athlete.FirstName, now,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	log.Printf("Created user: %d\n", id)

	return &models.User{
		ID:        int(id),
		StravaID:  athlete.StravaId,
		FirstName: athlete.FirstName,
		CreatedAt: now,
	}, nil

}

func (r *Repo) GetUserByStravaId(ctx context.Context, stravaID int) (*models.User, error) {
	defer utility.TimeCheck("repo.GetUserByStravaId", time.Now())

	var u models.User
	var err error

	err = r.db.QueryRowContext(ctx,
		`SELECT * FROM users WHERE strava_id = ?`, stravaID,
	).Scan(&u.ID, &u.StravaID, &u.FirstName, &u.CreatedAt, &u.LastSyncAt)

	if err != nil {
		return nil, err
	}

	log.Printf("Found user: %d\n", u.ID)
	return &u, nil
}

func (r *Repo) SetUserLastSync(ctx context.Context, userID int) error {
	defer utility.TimeCheck("repo.SetUserLastSync", time.Now())

	var err error

	now := time.Now().UTC()

	result, err := r.db.ExecContext(ctx,
		`UPDATE users
		SET last_sync_at = ?
		WHERE id = ?`, now, userID,
	)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("Failed to set last sync for user: %d", userID)
	}

	return nil
}

//
// Daily Activity Queries
//

func (r *Repo) AddDailyActivities(ctx context.Context, activites map[time.Time]models.DailyActivity) error {
	defer utility.TimeCheck("repo.AddDailyActivities", time.Now())

	expected := len(activites)
	var actual = 0

	stmt, err := r.db.PrepareContext(ctx,
		`INSERT INTO daily_activities_runs 
		(user_id, date, activity_count, moving_time_seconds, last_updated) 
		VALUES (?, ?, ?, ?, ?)
		`)
	if err != nil {
		return err
	}

	for _, record := range activites {

		userID := record.UserID
		date := record.Date
		activityCount := record.ActivityCount
		movingTime := record.MovingTimeSeconds
		now := time.Now().UTC()

		_, err := stmt.ExecContext(ctx, userID, date, activityCount, movingTime, now)
		if err != nil {
			log.Printf("Insert failed. Reason: %s\n", err)
			continue
		}
		actual++

	}
	stmt.Close()

	if expected != actual {
		return fmt.Errorf("Expected %d rows affected, got: %d\n", expected, actual)
	}

	return nil
}

func (r *Repo) CountTotalActiveDaysById(ctx context.Context, userId int) (int, error) {
	defer utility.TimeCheck("repo.CountTotalActiveDaysById", time.Now())

	var count int
	var err error

	err = r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM daily_activities_runs WHERE user_id = ?`, userId,
	).Scan(&count)

	if err != nil {
		return 0, err
	}

	return count, nil

}

func (r *Repo) FilterUserActiveDays(ctx context.Context, userID int, from, to time.Time) (map[time.Time]DailyAggregate, error) {
	defer utility.TimeCheck("repo.FilterUserActiveDays", time.Now())

	var records = make(map[time.Time]DailyAggregate)
	var err error

	rows, err := r.db.QueryContext(ctx, `
		SELECT date, activity_count, moving_time_seconds
		FROM daily_activities_runs
		WHERE user_id = ? AND date >= ? AND date <= ?
		ORDER BY date ASC`, userID, from, to)

	if err != nil {
		return records, err
	}
	defer rows.Close()

	for rows.Next() {
		var date time.Time
		var agg DailyAggregate

		err = rows.Scan(
			&date,
			&agg.ActivityCount,
			&agg.MovingTimeSeconds,
		)

		if err != nil {
			return records, err
		}

		records[date] = agg
	}

	return records, nil
}

type DailyAggregate struct {
	ActivityCount     int
	MovingTimeSeconds int
}
