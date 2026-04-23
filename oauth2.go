package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/pkg/browser"
)

const (
	requiredScopes = "read,activity:read"

	stravaAuthURL  = "http://www.strava.com/oauth/authorize"
	stravaTokenURL = "https://www.strava.com/oauth/token"
)

func fetchAccessToken() (string, error) {

	code := ""

	clientId := os.Getenv("STRAVA_OAUTH_CLIENT_ID")
	clientSecret := os.Getenv("STRAVA_OAUTH_CLIENT_SECRET")
	callbackURL := os.Getenv("CALLBACK_URL")

	if clientId == "" || clientSecret == "" || callbackURL == "" {
		return "", fmt.Errorf("Missing: client id / secret / callback URL")
	}

	//TODO: make stronger
	state := fmt.Sprint(rand.Text())

	q := url.Values{}
	q.Set("client_id", clientId)
	q.Set("redirect_uri", callbackURL)
	q.Set("response_type", "code")
	q.Set("approval_prompt", "force")
	q.Set("scope", requiredScopes)
	q.Set("state", state)

	authURL := stravaAuthURL + "?" + q.Encode()

	closeChan := make(chan bool)

	// callback handler
	http.HandleFunc("/oauth/callback", func(w http.ResponseWriter, r *http.Request) {

		_ = r.ParseForm()

		stateValue := r.FormValue("state")
		acceptedScopes := r.FormValue("scope")
		codeValue := r.FormValue("code")
		if stateValue == state && acceptedScopes == requiredScopes && codeValue != "" {
			code = codeValue
		}

		w.WriteHeader(http.StatusSeeOther)
		w.Write([]byte("You may return to the terminal."))
		closeChan <- true
	})

	printRedirectHelp(authURL)

	_ = browser.OpenURL(authURL)

	server := &http.Server{Addr: ":8085"}
	// go routine for shutting down the server
	go func() {
		okToClose := <-closeChan
		if okToClose {
			if err := server.Shutdown(context.Background()); err != nil {
				log.Println("Failed to shutdown server", err)
			}
		}
	}()

	//Blocks until server is closed
	server.ListenAndServe()

	token, err := exchangeCodeForToken(clientId, clientSecret, code, "authorization_code")
	if err != nil {
		return "", fmt.Errorf("Unable to acquire Strava user token")
	}

	return token.AccessToken, nil
}

func printRedirectHelp(url string) {
	fmt.Println()
	fmt.Println("Attempting to open browser for authentication.")
	fmt.Println("If you are not redirected to the browser, use this link:")
	fmt.Println(url)
	fmt.Println()

}

func exchangeCodeForToken(clientId, secret, code, grantType string) (AuthResponse, error) {

	form := url.Values{}
	form.Set("client_id", clientId)
	form.Set("client_secret", secret)
	form.Set("code", code)
	form.Set("grant_type", grantType)

	resp, err := http.PostForm(stravaTokenURL, form)
	if err != nil {
		return AuthResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return AuthResponse{}, errors.New("token exchange failed")
	}

	var t AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return AuthResponse{}, err
	}

	token := AuthResponse{
		AccessToken: t.AccessToken,
	}

	return token, nil
}

type AuthResponse struct {
	AccessToken string `json:"access_token"`
}
