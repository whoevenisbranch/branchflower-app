package oauth

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"
)

const (
	requiredScopes = "read,activity:read_all"

	stravaAuthURL  = "http://www.strava.com/oauth/authorize"
	stravaTokenURL = "https://www.strava.com/oauth/token"
)

type OAuthHandler struct {
	Config *Config
}

type Config struct {
	ClientID     string
	ClientSecret string
	CallbackURL  string
}

type OAuth struct {
	AccessToken string
	Expiration  time.Time
}

type Session struct {
	OAuth OAuth
	State string
}

type OAuthAccessResponse struct {
	AccessToken string `json:"access_token"`
	Expiration  int    `json:"expires_in"`
}

var sessions = make(map[string]Session)

func NewOAuthHandler(cfg Config) OAuthHandler {
	return OAuthHandler{
		Config: &cfg,
	}
}

func (h *OAuthHandler) Protected(handler http.HandlerFunc) http.Handler {
	return h.RequireAuth(handler)
}

func (h *OAuthHandler) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		sessionToken, err := r.Cookie("session")
		if err != nil {
			h.getAuthToken(w, r)
			return
		}

		session, ok := sessions[sessionToken.Value]
		if !ok {
			h.getAuthToken(w, r)
			return
		}

		if time.Now().After(session.OAuth.Expiration) {
			delete(sessions, sessionToken.Value)
			h.getAuthToken(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (h *OAuthHandler) getAuthToken(w http.ResponseWriter, r *http.Request) {

	//TODO: make stronger
	sessionID := rand.Text()
	state := fmt.Sprint(rand.Text())

	sessions[sessionID] = Session{
		State: state,
	}

	http.SetCookie(w, &http.Cookie{
		Value: sessionID,
		Name:  "session",
		Path:  "/",
	})

	q := url.Values{}
	q.Set("client_id", h.Config.ClientID)
	q.Set("redirect_uri", h.Config.CallbackURL)
	q.Set("response_type", "code")
	q.Set("approval_prompt", "force")
	q.Set("scope", requiredScopes)
	q.Set("state", state)

	authURL := stravaAuthURL + "?" + q.Encode()

	http.Redirect(w, r, authURL, http.StatusSeeOther)

}

func (h *OAuthHandler) HandleOAuthRedirect(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		log.Print("could not parse query", err)
		w.WriteHeader(http.StatusBadRequest)
	}

	returnedState := r.FormValue("state")
	if returnedState == "" {
		http.Error(w, "missing state", http.StatusBadRequest)
		return
	}

	cookie, err := r.Cookie("session")
	if err != nil {
		http.Error(w, "missing session", http.StatusBadRequest)
		return
	}

	session, ok := sessions[cookie.Value]
	if !ok {
		http.Error(w, "invalid session", http.StatusBadRequest)
		return
	}

	if session.State != returnedState {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")

	form := url.Values{}
	form.Set("client_id", h.Config.ClientID)
	form.Set("client_secret", h.Config.ClientSecret)
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

	session.OAuth = OAuth{
		AccessToken: t.AccessToken,
		Expiration:  time.Now().Add(time.Duration(t.Expiration) * time.Second),
	}
	session.State = ""

	sessions[cookie.Value] = session

	w.Header().Set("Location", "/tree")
	w.WriteHeader(http.StatusSeeOther)
}
