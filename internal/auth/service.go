package auth

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/whoevenisbranch/branchflower/internal/strava"
)

const (
	requiredScopes = "read,activity:read_all"
	stravaAuthURL  = "http://www.strava.com/oauth/authorize"
)

type OAuthService struct {
	OAuthConfig StravaOAuthConfig
	Sessions    map[string]Session
}

func NewOAuthService(cfg StravaOAuthConfig) OAuthService {
	return OAuthService{
		OAuthConfig: cfg,
		Sessions:    make(map[string]Session),
	}
}

func (svc *OAuthService) AuthenticateWithStrava(w http.ResponseWriter, r *http.Request) (bool, error) {
	sessionToken, err := r.Cookie("session")
	if err != nil {
		log.Print("no cookie exists")
		svc.getAuthToken(w, r)
		return true, nil
	}

	session, ok := svc.Sessions[sessionToken.Value]
	if !ok {
		log.Print("no session exists")
		svc.getAuthToken(w, r)
		return true, nil
	}

	if time.Now().After(session.OAuth.Expiration) {
		log.Print(session.OAuth.Expiration)
		log.Print("token expired")
		delete(svc.Sessions, sessionToken.Value)
		svc.getAuthToken(w, r)
		return true, nil
	}

	err = svc.validateAuthToken(session.OAuth.AccessToken)
	if err != nil {
		svc.getAuthToken(w, r)
		return true, nil
	}

	return false, nil
}

func (svc *OAuthService) getAuthToken(w http.ResponseWriter, r *http.Request) {

	q := url.Values{}
	q.Set("client_id", svc.OAuthConfig.ClientId)
	q.Set("redirect_uri", svc.OAuthConfig.CallbackURL)
	q.Set("response_type", "code")
	q.Set("approval_prompt", "force")
	q.Set("scope", requiredScopes)

	authURL := stravaAuthURL + "?" + q.Encode()

	http.Redirect(w, r, authURL, http.StatusSeeOther)
}

func (svc *OAuthService) validateAuthToken(token string) error {

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

func (svc *OAuthService) GetSessionFromCookie(r *http.Request) Session {
	cookie, _ := r.Cookie("session")
	return svc.Sessions[cookie.Value]
}
