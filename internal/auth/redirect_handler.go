package auth

import (
	"crypto/rand"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"time"
)

const stravaTokenURL = "https://www.strava.com/oauth/token"

type RedirectHandler struct {
	service OAuthService
}

func NewAuthRedirectHandler(svc OAuthService) RedirectHandler {
	return RedirectHandler{
		service: svc,
	}
}

func (handler *RedirectHandler) HandleOAuthRedirect(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Print("could not parse form ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")
	log.Print(code)

	form := url.Values{}
	form.Set("client_id", handler.service.OAuthConfig.ClientId)
	form.Set("client_secret", handler.service.OAuthConfig.ClientSecret)
	form.Set("code", code)
	form.Set("grant_type", "authorization_code")

	resp, err := http.PostForm(stravaTokenURL, form)
	if err != nil {
		log.Print("error sending request for auth token", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Print("error response from strava ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var t OAuthAccessResponse
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		log.Print("could not parse json response", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	sessionID := rand.Text()
	handler.service.Sessions[sessionID] = Session{
		OAuth: OAuth{
			AccessToken: t.AccessToken,
			Expiration:  time.Now().UTC().Add(time.Duration(t.Expiration) * time.Second),
			AthleteId:   t.Athlete.ID,
		},
	}

	http.SetCookie(w, &http.Cookie{
		Value: sessionID,
		Name:  "session",
		Path:  "/",
	})

	log.Print("session created")

	http.Redirect(w, r, "/protected", http.StatusSeeOther)
}
