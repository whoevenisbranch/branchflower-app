package strava

func (sa stravaAthleteDTO) ToAthlete() Athlete {
	return Athlete{
		StravaId:  sa.ID,
		FirstName: sa.FirstName,
		Username:  sa.Username,
	}
}

func (sa StravaActivitiesDTO) ToActivites() []Activity {

	var bucket []Activity

	for _, activity := range sa {
		bucket = append(bucket, Activity{
			Id:                activity.ID,
			Name:              activity.Name,
			Type:              activity.SportType,
			StartTimestamp:    activity.StartDate,
			MovingTimeSeconds: activity.MovingTimeSeconds,
		})
	}

	return bucket
}
