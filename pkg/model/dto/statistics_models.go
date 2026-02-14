package dto

import (
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
)

// StatisticsResponse represents the API response for statistics
type StatisticsResponse struct {
	UserID       uint                        `json:"user_id"`
	BucketFormat string                      `json:"bucket_format"`
	Buckets      map[string]StatisticBuckets `json:"buckets"`
}

// StatisticBuckets represents statistics grouped by workout type
type StatisticBuckets struct {
	WorkoutType      string                   `json:"workout_type"`
	LocalWorkoutType string                   `json:"local_workout_type"`
	Buckets          map[string]StatisticData `json:"buckets"`
}

// StatisticData represents statistics for a specific bucket
type StatisticData struct {
	Bucket              string  `json:"bucket"`
	Workouts            int     `json:"workouts"`
	DurationSeconds     float64 `json:"duration_seconds"`
	Distance            float64 `json:"distance"`
	AverageSpeed        float64 `json:"average_speed"`
	AverageSpeedNoPause float64 `json:"average_speed_no_pause"`
	MaxSpeed            float64 `json:"max_speed"`
	Duration            float64 `json:"duration"`
}

// NewStatisticsResponse creates a new statistics response from database statistics
func NewStatisticsResponse(stats *model.Statistics) StatisticsResponse {
	buckets := make(map[string]StatisticBuckets)

	for workoutType, workoutBuckets := range stats.Buckets {
		bucketData := make(map[string]StatisticData)

		for bucketKey, bucket := range workoutBuckets.Buckets {
			bucketData[bucketKey] = StatisticData{
				Bucket:              bucket.Bucket,
				Workouts:            bucket.Workouts,
				DurationSeconds:     bucket.DurationSeconds,
				Distance:            bucket.Distance,
				AverageSpeed:        bucket.AverageSpeed,
				AverageSpeedNoPause: bucket.AverageSpeedNoPause,
				MaxSpeed:            bucket.MaxSpeed,
				Duration:            bucket.Duration.Seconds(),
			}
		}

		buckets[string(workoutType)] = StatisticBuckets{
			WorkoutType:      string(workoutType),
			LocalWorkoutType: workoutBuckets.LocalWorkoutType,
			Buckets:          bucketData,
		}
	}

	return StatisticsResponse{
		UserID:       uint(stats.UserID),
		BucketFormat: stats.BucketFormat,
		Buckets:      buckets,
	}
}
