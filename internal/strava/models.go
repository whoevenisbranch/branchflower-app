package strava

import (
	"fmt"
	"time"
)

//Athlete

type stravaAthleteDTO struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	// other fields exist on API
}

type Athlete struct {
	StravaId  int
	FirstName string
	Username  string
}

func (sa stravaAthleteDTO) ToAthlete() Athlete {
	return Athlete{
		StravaId:  sa.ID,
		FirstName: sa.FirstName,
		Username:  sa.Username,
	}
}

// Activities

type stravaSummaryActivityDTO struct {
	ID                 int64     `json:"id"`
	Name               string    `json:"name"`
	StartDate          time.Time `json:"start_date_local"`
	Distance           float64   `json:"distance"`
	MovingTimeSeconds  int       `json:"moving_time"`
	ElapsedTime        float64   `json:"elapsed_time"`
	TotalElevationGain float64   `json:"total_elevation_gain"`
	SportType          string    `json:"sport_type"`
	// other fields exist on API
}
type StravaActivitiesDTO []stravaSummaryActivityDTO

type APIError struct {
	Code    int
	Message string
}

func (e APIError) Error() string {
	return fmt.Sprintf("API error: status=%d message=%s", e.Code, e.Message)
}
