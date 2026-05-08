package app

import (
	"context"
	"log"
	"time"

	"github.com/whoevenisbranch/branchflower/internal/service"
)

type application struct {
	svc service.Service
}

func NewApp(service service.Service) *application {
	return &application{
		svc: service,
	}
}

func (app *application) Run(ctx context.Context) error {

	//TODO: this is where the API will be served from

	var err error

	log.Println("### Welcome to Branchflower App ###")

	user, err := app.svc.GetUser(ctx)
	if err != nil {
		return err
	}

	user.Greet()

	if user.LastSyncAt == nil || time.Since(*user.LastSyncAt) > 6*time.Hour {
		if err = app.svc.SyncActivities(ctx, user.ID); err != nil {
			return err
		}
	}

	report, err := app.svc.GetUserTreeData(ctx, user.ID)
	if err != nil {
		return err
	}

	report.BaseScores.Display()
	report.UIScores.Display()

	return nil
}
