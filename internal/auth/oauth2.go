package auth

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/whoevenisbranch/branchflower/internal/strava"
)

const (
	requiredScopes = "read,activity:read_all"
	stravaAuthURL  = "http://www.strava.com/oauth/authorize"
)

var Sessions = make(map[string]Session)

func AuthenticateWithStrava(w http.ResponseWriter, r *http.Request) (bool, error) {
	sessionToken, err := r.Cookie("session")
	if err != nil {
		log.Print("no cookie exists")
		getAuthToken(w, r)
		return true, nil
	}

	session, ok := Sessions[sessionToken.Value]
	if !ok {
		log.Print("no session exists")
		getAuthToken(w, r)
		return true, nil
	}

	if time.Now().After(session.OAuth.Expiration) {
		log.Print("token expired")
		delete(Sessions, sessionToken.Value)
		getAuthToken(w, r)
		return true, nil
	}

	err = validateAuthToken(session.OAuth.AccessToken)
	if err != nil {
		getAuthToken(w, r)
		return true, nil
	}

	return false, nil
}

func getAuthToken(w http.ResponseWriter, r *http.Request) {

	q := url.Values{}
	q.Set("client_id", os.Getenv("STRAVA_OAUTH_CLIENT_ID"))
	q.Set("redirect_uri", os.Getenv("CALLBACK_URL"))
	q.Set("response_type", "code")
	q.Set("approval_prompt", "force")
	q.Set("scope", requiredScopes)

	authURL := stravaAuthURL + "?" + q.Encode()

	http.Redirect(w, r, authURL, http.StatusSeeOther)
}

func validateAuthToken(token string) error {

	client, err := strava.NewStravaClient(token)
	if err != nil {
		log.Print(err)
		return err
	}

	_, err = client.GetAthlete(context.Background())
	if err != nil {
		log.Print(err)
		return err
	}

	return nil

}
