package storage

import (
	"context"
	"fmt"
	"log"
	"time"

	timeutils "github.com/whoevenisbranch/branchflower/internal/utility/time_utils"
)

type UserRepository struct {
	db *DB
}

func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

func (r *UserRepository) CreateUser(ctx context.Context, stravaID int, name string) (*User, error) {
	defer timeutils.TimeCheck("UserRepository.CreateUser", time.Now())

	var err error

	now := time.Now().UTC()

	result, err := r.db.conn.ExecContext(ctx,
		`INSERT INTO users (strava_id, first_name, created_at)
		VALUES (?, ?, ?)`, stravaID, name, now,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	log.Printf("Created user: %d\n", id)

	return &User{
		ID:        int(id),
		StravaID:  stravaID,
		FirstName: name,
		CreatedAt: now,
	}, nil

}

func (r *UserRepository) GetUserByStravaId(ctx context.Context, stravaID int) (*User, error) {
	defer timeutils.TimeCheck("UserRepository.GetUserByStravaId", time.Now())

	var u User
	var err error

	err = r.db.conn.QueryRowContext(ctx,
		`SELECT * FROM users WHERE strava_id = ?`, stravaID,
	).Scan(&u.ID, &u.StravaID, &u.FirstName, &u.CreatedAt, &u.LastSyncAt)

	if err != nil {
		return nil, err
	}

	log.Printf("Found user: %d\n", u.ID)
	return &u, nil
}

func (r *UserRepository) SetUserLastSync(ctx context.Context, userID int) error {
	defer timeutils.TimeCheck("UserRepository.SetUserLastSync", time.Now())

	var err error

	now := time.Now().UTC()

	result, err := r.db.conn.ExecContext(ctx,
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
