package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const numActivitiesPerPage = 200

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
			Timeout: 20 * time.Second,
		},
		baseURL:     baseURL,
		accessToken: accessToken,
	}, nil
}

func (sc *StravaClient) GetAthlete(ctx context.Context) (Athlete, error) {

	baseUrl := sc.baseURL + "/athlete"

	endpoint, err := url.Parse(baseUrl)
	if err != nil {
		return Athlete{}, err
	}

	dto, err := get[StravaAthleteDTO](sc, ctx, endpoint.String())
	if err != nil {
		return Athlete{}, err
	}
	return dto.ToAthlete(), nil
}

func (sc *StravaClient) GetAthleteActivities(ctx context.Context, before, after int64) ([]Activity, error) {

	baseUrl := sc.baseURL + "/athlete/activities"
	endpoint, err := url.Parse(baseUrl)
	if err != nil {
		return []Activity{}, err
	}

	queryParams := url.Values{}
	queryParams.Set("per_page", strconv.Itoa(numActivitiesPerPage))

	//protect against activity uploaded during collection
	queryParams.Set("before", strconv.FormatInt(before, 10))
	queryParams.Set("after", strconv.FormatInt(after, 10))

	bucket := []Activity{}
	pageCounter := 1

	for {
		queryParams.Set("page", strconv.Itoa(pageCounter))
		endpoint.RawQuery = queryParams.Encode()

		dto, err := get[StravaActivitiesDTO](sc, ctx, endpoint.String())
		if err != nil {
			return []Activity{}, err
		}

		returned := dto.ToActivies()

		bucket = append(bucket, returned...)

		//no next page to query
		if len(dto) < numActivitiesPerPage {
			break
		}

		pageCounter++
	}

	return bucket, nil
}

func get[T any](sc *StravaClient, ctx context.Context, endpoint string) (T, error) {
	defer timeCheck(time.Now())

	var zero T

	fmt.Printf("Requesting: %s", endpoint)

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

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)

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

	var zero T

	statusCode := response.StatusCode

	switch {

	case statusCode >= 200 && statusCode < 300:
		var t T
		err := json.NewDecoder(response.Body).Decode(&t)
		if err != nil {
			return zero, err
		}
		return t, nil

	case statusCode == http.StatusUnauthorized:

		return zero, APIError{
			Code:    statusCode,
			Message: ErrStravaAuthError.Error(),
		}

	case statusCode >= 400 && statusCode < 500:
		return zero, APIError{
			Code:    statusCode,
			Message: ErrStravaAuthError.Error(),
		}

	case statusCode == http.StatusTooManyRequests || statusCode >= 500:
		return zero, ErrRecoverableServerError

	default:
		return zero, ErrUnrecognisedStatusCode
	}
}

//Athlete

type StravaAthleteDTO struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	// other fields exist on API
}

type Athlete struct {
	StravaId    int
	FullName    string
	Username    string
	TotalRuns   int
	RunSnapshot map[int]SnapshotWeek
}

func (sa StravaAthleteDTO) ToAthlete() Athlete {
	return Athlete{
		StravaId: sa.ID,
		FullName: fmt.Sprintf("%s %s", sa.FirstName, sa.LastName),
		Username: sa.Username,
	}
}

// Activities

type StravaSummaryActivityDTO struct {
	ID                 int64     `json:"id"`
	Name               string    `json:"name"`
	StartDateLocal     time.Time `json:"start_date_local"`
	Distance           float64   `json:"distance"`
	MovingTime         float64   `json:"moving_time"`
	ElapsedTime        float64   `json:"elapsed_time"`
	TotalElevationGain float64   `json:"total_elevation_gain"`
	SportType          string    `json:"sport_type"`
	// other fields exist on API
}
type StravaActivitiesDTO []StravaSummaryActivityDTO

type Activity struct {
	Id             int64
	Name           string
	Type           string
	LocalStartTime time.Time
	Distance       float64
	Time           float64
}

func (sa StravaActivitiesDTO) ToActivies() []Activity {

	var bucket []Activity

	for _, activity := range sa {
		bucket = append(bucket, Activity{
			Id:             activity.ID,
			Name:           activity.Name,
			Type:           activity.SportType,
			LocalStartTime: time.Unix(activity.StartDateLocal.Unix(), 10),
			Distance:       activity.Distance,
			Time:           activity.MovingTime,
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
