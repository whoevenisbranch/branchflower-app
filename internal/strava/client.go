package strava

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const baseURL string = "https://www.strava.com/api/v3"
const numActivitiesPerPage = 200

var sharedHTTPClient *http.Client = &http.Client{
	Timeout: 10 * time.Second,
}

type StravaClient struct {
	httpClient  *http.Client
	accessToken string
}

func NewStravaClient(token string) (StravaClient, error) {

	if token == "" {
		return StravaClient{}, ErrStravaClientMissingAccessToken
	}

	return StravaClient{
		httpClient:  sharedHTTPClient,
		accessToken: token,
	}, nil
}

func (sc *StravaClient) GetAthlete(ctx context.Context) (Athlete, error) {

	baseUrl := baseURL + "/athlete"

	endpoint, err := url.Parse(baseUrl)
	if err != nil {
		return Athlete{}, err
	}

	dto, err := get[stravaAthleteDTO](sc, ctx, endpoint.String())
	if err != nil {
		return Athlete{}, err
	}

	return dto.ToAthlete(), nil
}

func (sc *StravaClient) GetAllAthleteActivities(ctx context.Context) (StravaActivitiesDTO, error) {

	baseUrl := baseURL + "/athlete/activities"
	endpoint, err := url.Parse(baseUrl)
	if err != nil {
		return StravaActivitiesDTO{}, err
	}

	queryParams := url.Values{}
	queryParams.Set("per_page", strconv.Itoa(numActivitiesPerPage))

	//protect against activity uploaded during collection
	queryParams.Set("before", strconv.FormatInt(time.Now().Unix(), 10))

	bucket := StravaActivitiesDTO{}
	pageCounter := 1

	for {
		queryParams.Set("page", strconv.Itoa(pageCounter))
		endpoint.RawQuery = queryParams.Encode()

		dto, err := get[StravaActivitiesDTO](sc, ctx, endpoint.String())
		if err != nil {
			return StravaActivitiesDTO{}, err
		}

		bucket = append(bucket, dto...)

		//no next page to query
		if len(dto) < numActivitiesPerPage {
			break
		}

		pageCounter++
	}

	return bucket, nil
}

func get[T any](sc *StravaClient, ctx context.Context, endpoint string) (T, error) {

	var zero T

	log.Printf("Requesting: %s", endpoint)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)

	if err != nil {
		return zero, err
	}

	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Branchflower-App")

	authorizationHeader := fmt.Sprintf("Bearer %s", sc.accessToken)
	request.Header.Set("Authorization", authorizationHeader)

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

// Error handling
var ErrStravaClientMissingBaseURL = errors.New("Cannot create Strava client without base URL set.")
var ErrStravaClientMissingAccessToken = errors.New("Cannot create Strava client without API Key set.")

var ErrStravaAuthError = errors.New("Strava Authentication Error.")
var ErrUnrecoverableClientError = errors.New("Unrecoverable Client Error.")
var ErrRecoverableServerError = errors.New("Recoverable Strava Server Error.")
var ErrUnrecognisedStatusCode = errors.New("Received Unexpected Status Code.")
