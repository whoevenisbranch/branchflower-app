package auth

import (
	"crypto/rand"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

const stravaTokenURL = "https://www.strava.com/oauth/token"

func HandleOAuthRedirect(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Print("could not parse form ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")

	form := url.Values{}
	form.Set("client_id", os.Getenv("STRAVA_OAUTH_CLIENT_ID"))
	form.Set("client_secret", os.Getenv("STRAVA_OAUTH_CLIENT_SECRET"))
	form.Set("code", code)
	form.Set("grant_type", "authorization_code")

	resp, err := http.PostForm(stravaTokenURL, form)
	if err != nil {
		log.Print("error sending request for auth token", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var t OAuthAccessResponse
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		log.Print("could not parse json response", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	sessionID := rand.Text()
	Sessions[sessionID] = Session{
		OAuth: OAuth{
			AccessToken: t.AccessToken,
			Expiration:  time.Now().Add(time.Duration(t.Expiration) * time.Second),
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
