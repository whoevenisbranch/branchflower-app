package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type StravaClient struct {
	httpClient  *http.Client
	baseURL     string
	accessToken string
}

func NewStravaClient(baseURL, accessToken string) (*StravaClient, error) {

	if baseURL == "" {
		return nil, ErrStravaClientMissingBaseURL
	}
	if accessToken == "" {
		return nil, ErrStravaClientMissingAccessToken
	}

	return &StravaClient{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		baseURL:     baseURL,
		accessToken: accessToken,
	}, nil
}

func (sc *StravaClient) GetAthlete(ctx context.Context) (Athlete, error) {
	dto, err := get[StravaAthleteDTO](sc, ctx, "/athlete")
	if err != nil {
		return Athlete{}, err
	}
	return dto.ToAthlete(), nil
}

func (sc *StravaClient) GetAthleteActivities(ctx context.Context) ([]Activity, error) {
	dto, err := get[StravaActivitiesDTO](sc, ctx, "/athlete/activities")
	if err != nil {
		return []Activity{}, err
	}
	return dto.ToActivies(), nil
}

func get[T any](sc *StravaClient, ctx context.Context, endpoint string) (T, error) {

	var zero T

	request, err := sc.buildHTTPRequest(endpoint, ctx)
	if err != nil {
		return zero, err
	}

	response, err := sc.httpClient.Do(request)
	if err != nil {
		return zero, err
	}
	defer response.Body.Close()

	dto, err := handleResponse[T](response)
	if err != nil {
		return zero, err
	}

	return dto, nil
}

func (sc *StravaClient) buildHTTPRequest(endpoint string, ctx context.Context) (*http.Request, error) {

	url := baseStravaURL + endpoint

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)

	if err != nil {
		return nil, err
	}

	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Branchflower-App")

	authorizationHeader := fmt.Sprintf("Bearer %s", sc.accessToken)
	request.Header.Set("Authorization", authorizationHeader)

	return request, nil
}

func handleResponse[T any](response *http.Response) (T, error) {
	statusCode := response.StatusCode

	switch {

	case statusCode >= 200 && statusCode < 300:
		var t T
		err := json.NewDecoder(response.Body).Decode(&t)
		if err != nil {
			return *new(T), err
		}
		return t, nil

	case statusCode == http.StatusUnauthorized:
		return *new(T), APIError{
			Code:    statusCode,
			Message: ErrStravaAuthError.Error(),
		}

	case statusCode >= 400 && statusCode < 500:
		return *new(T), APIError{
			Code:    statusCode,
			Message: ErrStravaAuthError.Error(),
		}

	case statusCode == http.StatusTooManyRequests || statusCode >= 500:
		return *new(T), ErrRecoverableServerError

	default:
		return *new(T), ErrUnrecognisedStatusCode
	}
}

//Athlete

type StravaAthleteDTO struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	//TODO: missing fields from the api
}

type Athlete struct {
	FullName string
	Username string
}

func (sa StravaAthleteDTO) ToAthlete() Athlete {
	return Athlete{
		FullName: fmt.Sprintf("%s %s", sa.FirstName, sa.LastName),
		Username: sa.Username,
	}
}

// Activities

type StravaSummaryActivityDTO struct {
	ID                 int64   `json:"id"`
	Name               string  `json:"name"`
	Distance           float64 `json:"distance"`
	MovingTime         float64 `json:"moving_time"`
	ElapsedTime        float64 `json:"elapsed_time"`
	TotalElevationGain float64 `json:"total_elevation_gain"`
	SportType          string  `json:"sport_type"`
}
type StravaActivitiesDTO []StravaSummaryActivityDTO

type Activity struct {
	Id   int64
	Name string
}

func (sa StravaActivitiesDTO) ToActivies() []Activity {

	var bucket []Activity

	for _, activity := range sa {
		bucket = append(bucket, Activity{
			Id:   activity.ID,
			Name: activity.Name,
		})
	}

	return bucket
}

// Error handling
var ErrStravaClientMissingBaseURL = errors.New("Cannot create Strava client without base URL set.")
var ErrStravaClientMissingAccessToken = errors.New("Cannot create Strava client without API Key set.")

var ErrStravaAuthError = errors.New("Strava Authentication Error.")
var ErrUnrecoverableClientError = errors.New("Unrecoverable Client Error.")
var ErrRecoverableServerError = errors.New("Recoverable Strava Server Error.")
var ErrUnrecognisedStatusCode = errors.New("Received Unexpected Status Code.")

type APIError struct {
	Code    int
	Message string
}

func (e APIError) Error() string {
	return fmt.Sprintf("API error: status=%d message=%s", e.Code, e.Message)
}
