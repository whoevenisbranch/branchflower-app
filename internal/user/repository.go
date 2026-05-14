package user

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/whoevenisbranch/branchflower/internal/database"
)

type UserRepository struct {
	dbConn database.DB
}

func NewRepository(db *database.DB) UserRepository {
	return UserRepository{
		dbConn: *db,
	}
}

func (repo *UserRepository) CreateUser(ctx context.Context, stravaID int, name string) (User, error) {
	var err error

	now := time.Now().UTC()

	result, err := repo.dbConn.Conn.ExecContext(ctx,
		`INSERT INTO users (strava_id, first_name, created_at)
		VALUES (?, ?, ?)`, stravaID, name, now,
	)
	if err != nil {
		return User{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return User{}, err
	}

	log.Printf("Created user: %d\n", id)

	return User{
		ID:        int(id),
		StravaID:  stravaID,
		FirstName: name,
		CreatedAt: now,
	}, nil
}

func (repo *UserRepository) GetUserByStravaID(ctx context.Context, id int) (User, error) {

	var u User
	var err error

	err = repo.dbConn.Conn.QueryRowContext(ctx,
		`SELECT * FROM users WHERE strava_id = ?`, id,
	).Scan(&u.ID, &u.StravaID, &u.FirstName, &u.CreatedAt, &u.LastSyncAt)

	if err != nil {
		return User{}, err
	}

	return u, nil
}

func (repo *UserRepository) SetUserLastSync(ctx context.Context, userID int) error {

	var err error

	now := time.Now().UTC()

	result, err := repo.dbConn.Conn.ExecContext(ctx,
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
