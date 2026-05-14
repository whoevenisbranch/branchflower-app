package activity

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/whoevenisbranch/branchflower/internal/database"
)

type ActivityRepository struct {
	dbConn database.DB
}

func NewRepository(db *database.DB) ActivityRepository {
	return ActivityRepository{
		dbConn: *db,
	}
}

func (repo *ActivityRepository) AddDailyActivities(ctx context.Context, activites map[time.Time]DailyActivity) error {

	expected := len(activites)
	var actual = 0

	stmt, err := repo.dbConn.Conn.PrepareContext(ctx,
		`INSERT INTO daily_activities 
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

func (repo *ActivityRepository) CountTotalActiveDaysById(ctx context.Context, userId int) (int, error) {

	var count int
	var err error

	err = repo.dbConn.Conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM daily_activities WHERE user_id = ?`, userId,
	).Scan(&count)

	if err != nil {
		return 0, err
	}

	return count, nil


}

func (repo *ActivityRepository) FilterActiveDaysByUserID(ctx context.Context, userID int, from, to time.Time) (map[time.Time]DailyAggregate, error) {

	var records = make(map[time.Time]DailyAggregate)
	var err error

	rows, err := repo.dbConn.Conn.QueryContext(ctx, `
		SELECT date, activity_count, moving_time_seconds
		FROM daily_activities
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
