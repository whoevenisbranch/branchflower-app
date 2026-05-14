package auth

import "time"

type OAuth struct {
	AthleteId   int
	AccessToken string
	Expiration  time.Time
}

type Session struct {
	OAuth OAuth
}

type StravaOAuthConfig struct {
	ClientId     string
	ClientSecret string
	CallbackURL  string
}

type OAuthAccessResponse struct {
	AccessToken string `json:"access_token"`
	Expiration  int    `json:"expires_in"`
	Athlete     struct {
		ID int `json:"id"`
	} `json:"athlete"`
}
