package activity

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/whoevenisbranch/branchflower/internal/auth"
	"github.com/whoevenisbranch/branchflower/internal/user"
)

var tmpl = template.Must(template.ParseFiles("templates/tree.html"))

type Handler struct {
	authService     auth.OAuthService
	userService     user.UserService
	activityService ActivityService
}

func NewHandler(userSvc user.UserService, activitySvc ActivityService, authSvc auth.OAuthService) Handler {
	return Handler{
		authService:     authSvc,
		userService:     userSvc,
		activityService: activitySvc,
	}
}

func (handler *Handler) Handle(w http.ResponseWriter, r *http.Request) {

	handled, err := handler.authService.AuthenticateWithStrava(w, r)
	if handled {
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	session := handler.authService.GetSessionFromCookie(r)

	u, err := handler.userService.GetOrCreateUser(session.OAuth.AthleteId, session.OAuth.AccessToken)
	if err != nil {
		http.Error(w, "error", 500)
		return
	}
	log.Println("user:", u.FirstName)

	if u.LastSyncAt == nil || time.Since(*u.LastSyncAt) > 6*time.Hour {
		if err = handler.activityService.SyncActivities(context.Background(), u.ID, session.OAuth.AccessToken); err != nil {
			http.Error(w, "sync error", 500)
			return
		}
		if err = handler.userService.SetUserLastSync(context.Background(), u.ID); err != nil {
			http.Error(w, "set sync error", 500)
			return
		}
	}

	tree, err := handler.activityService.GetUserTreeData(context.Background(), u.ID)
	if err != nil {
		http.Error(w, "tree error", 500)
		return
	}

	tData := templateData{
		UserInfo: u,
		Tree:     tree,
	}

	err = tmpl.Execute(w, tData)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}

}

type templateData struct {
	UserInfo user.User
	Tree     TreeData
}
