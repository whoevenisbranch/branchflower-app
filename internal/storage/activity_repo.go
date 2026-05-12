package storage

import (
	"context"
	"fmt"
	"log"
	"time"

	timeutils "github.com/whoevenisbranch/branchflower/internal/utility/time_utils"
)

type ActivityRepository struct {
	db *DB
}

func NewActivityRepository(db *DB) *ActivityRepository {
	return &ActivityRepository{
		db: db,
	}
}

func (r *ActivityRepository) AddDailyActivities(ctx context.Context, activites map[time.Time]DailyActivity) error {
	defer timeutils.TimeCheck("ActivityRepository.AddDailyActivities", time.Now())

	expected := len(activites)
	var actual = 0

	stmt, err := r.db.conn.PrepareContext(ctx,
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

func (r *ActivityRepository) CountTotalActiveDaysById(ctx context.Context, userId int) (int, error) {
	defer timeutils.TimeCheck("ActivityRepository.CountTotalActiveDaysById", time.Now())

	var count int
	var err error

	err = r.db.conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM daily_activities_runs WHERE user_id = ?`, userId,
	).Scan(&count)

	if err != nil {
		return 0, err
	}

	return count, nil

}

func (r *ActivityRepository) FilterActiveDaysByUserID(ctx context.Context, userID int, from, to time.Time) (map[time.Time]DailyAggregate, error) {
	defer timeutils.TimeCheck("ActivityRepository.FilterUserActiveDays", time.Now())

	var records = make(map[time.Time]DailyAggregate)
	var err error

	rows, err := r.db.conn.QueryContext(ctx, `
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
