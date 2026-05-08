package app

import (
	"context"
	"log"
	"time"

	"github.com/whoevenisbranch/branchflower/internal/service"
)

type app struct {
	service service.Service
}

func NewApp(service service.Service) *app {
	return &app{
		service: service,
	}
}

func (a *app) Run(ctx context.Context) error {

	var err error

	log.Println("### Welcome to Branchflower App ###")

	user, err := a.service.GetUser(ctx)
	if err != nil {
		return err
	}

	user.Greet()

	if user.LastSyncAt == nil || time.Since(*user.LastSyncAt) > 6*time.Hour {
		if err = a.service.SyncActivities(ctx, *user); err != nil {
			return err
		}
	}

	a.service.GetReport(ctx, *user)
	if err != nil {
		return err
	}

	return nil
}
