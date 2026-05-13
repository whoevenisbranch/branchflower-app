package user

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"github.com/whoevenisbranch/branchflower/internal/strava"
)

type UserService struct {
	store *UserRepository
}

func NewService(repo *UserRepository) UserService {
	return UserService{
		store: repo,
	}
}

func (svc *UserService) GetOrCreateUser(stravaID int, token string) (User, error) {

	var user User
	var err error

	log.Print("UserService.GetOrCreateUser")
	ctx := context.Background()

	client, err := strava.NewStravaClient(token)
	if err != nil {
		return User{}, err
	}

	athlete, err := client.GetAthlete(ctx)

	user, err = svc.store.GetUserByStravaID(ctx, athlete.StravaId)
	if errors.Is(err, sql.ErrNoRows) {
		user, err = svc.store.CreateUser(ctx, athlete.StravaId, athlete.FirstName)
	}

	if err != nil {
		return User{}, err
	}

	return user, nil

}

func (svc *UserService) SetUserLastSync(ctx context.Context, userID int) error {
	return svc.store.SetUserLastSync(ctx, userID)
}
