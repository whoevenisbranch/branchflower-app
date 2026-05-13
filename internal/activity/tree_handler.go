package activity

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/whoevenisbranch/branchflower/internal/auth"
	"github.com/whoevenisbranch/branchflower/internal/user"
)

type Handler struct {
	userService     *user.UserService
	activityService *ActivityService
}

func NewHandler(userSvc *user.UserService, activitySvc *ActivityService) Handler {
	return Handler{
		userService:     userSvc,
		activityService: activitySvc,
	}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {

	handled, err := auth.AuthenticateWithStrava(w, r)
	if handled {
		return
	}
	if err != nil {
		log.Println(err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	session := getSessionFromCookie(r)

	u, err := h.userService.GetOrCreateUser(session.OAuth.AthleteId, session.OAuth.AccessToken)
	if err != nil {
		http.Error(w, "error", 500)
		return
	}
	log.Println("user:", u.FirstName)

	if u.LastSyncAt == nil || time.Since(*u.LastSyncAt) > 6*time.Hour {
		if err = h.activityService.SyncActivities(context.Background(), u.ID, session.OAuth.AccessToken); err != nil {
			http.Error(w, "sync error", 500)
			return
		}
	}

	err = h.userService.SetUserLastSync(context.Background(), u.ID)
	if err != nil {
		http.Error(w, "set sync error", 500)
		return
	}

	tree, err := h.activityService.GetUserTreeData(context.Background(), u.ID)
	if err != nil {
		http.Error(w, "tree error", 500)
		return
	}

	json.NewEncoder(w).Encode(tree)

}

func getSessionFromCookie(r *http.Request) auth.Session {
	cookie, _ := r.Cookie("session")

	return auth.Sessions[cookie.Value]
}
