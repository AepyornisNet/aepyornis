package api

import (
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
)

// TotalsResponse represents workout totals in API v2 responses
type TotalsResponse struct {
	Workouts int64   `json:"workouts"`
	Distance float64 `json:"distance"`
	Duration int64   `json:"duration"` // Duration in seconds
	Up       float64 `json:"up"`
	Down     float64 `json:"down"`
}

// RecordResponse represents a single record value
type RecordResponse struct {
	Value     float64   `json:"value"`
	WorkoutID uint64    `json:"workout_id"`
	Date      time.Time `json:"date"`
}

// WorkoutRecordResponse represents workout records in API v2 responses
type WorkoutRecordResponse struct {
	WorkoutType         string          `json:"workout_type"`
	Active              bool            `json:"active"`
	Distance            *RecordResponse `json:"distance,omitempty"`
	AverageSpeed        *RecordResponse `json:"average_speed,omitempty"`
	AverageSpeedNoPause *RecordResponse `json:"average_speed_no_pause,omitempty"`
	MaxSpeed            *RecordResponse `json:"max_speed,omitempty"`
	Duration            *RecordResponse `json:"duration,omitempty"`
	TotalUp             *RecordResponse `json:"total_up,omitempty"`
}

// NewTotalsResponse converts a database Bucket to API response
func NewTotalsResponse(b *database.Bucket) TotalsResponse {
	return TotalsResponse{
		Workouts: int64(b.Workouts),
		Distance: b.Distance,
		Duration: int64(b.Duration.Seconds()),
		Up:       b.Up,
		Down:     0, // Down is not tracked in totals
	}
}

// NewWorkoutRecordResponse converts a database WorkoutRecord to API response
func NewWorkoutRecordResponse(wr *database.WorkoutRecord) WorkoutRecordResponse {
	response := WorkoutRecordResponse{
		WorkoutType: string(wr.WorkoutType),
		Active:      wr.Active,
	}

	if wr.Distance.Value != 0 {
		response.Distance = &RecordResponse{
			Value:     wr.Distance.Value,
			WorkoutID: wr.Distance.ID,
			Date:      wr.Distance.Date,
		}
	}
	if wr.AverageSpeed.Value != 0 {
		response.AverageSpeed = &RecordResponse{
			Value:     wr.AverageSpeed.Value,
			WorkoutID: wr.AverageSpeed.ID,
			Date:      wr.AverageSpeed.Date,
		}
	}
	if wr.AverageSpeedNoPause.Value != 0 {
		response.AverageSpeedNoPause = &RecordResponse{
			Value:     wr.AverageSpeedNoPause.Value,
			WorkoutID: wr.AverageSpeedNoPause.ID,
			Date:      wr.AverageSpeedNoPause.Date,
		}
	}
	if wr.MaxSpeed.Value != 0 {
		response.MaxSpeed = &RecordResponse{
			Value:     wr.MaxSpeed.Value,
			WorkoutID: wr.MaxSpeed.ID,
			Date:      wr.MaxSpeed.Date,
		}
	}
	if wr.Duration.Value != 0 {
		response.Duration = &RecordResponse{
			Value:     float64(wr.Duration.Value.Seconds()),
			WorkoutID: wr.Duration.ID,
			Date:      wr.Duration.Date,
		}
	}
	if wr.TotalUp.Value != 0 {
		response.TotalUp = &RecordResponse{
			Value:     wr.TotalUp.Value,
			WorkoutID: wr.TotalUp.ID,
			Date:      wr.TotalUp.Date,
		}
	}

	return response
}

// NewWorkoutRecordsResponse converts database workout records to API responses
func NewWorkoutRecordsResponse(wrs []*database.WorkoutRecord) []WorkoutRecordResponse {
	results := make([]WorkoutRecordResponse, len(wrs))
	for i, wr := range wrs {
		results[i] = NewWorkoutRecordResponse(wr)
	}
	return results
}
