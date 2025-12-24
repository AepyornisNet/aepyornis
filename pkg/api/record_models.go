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
	WorkoutType         string                   `json:"workout_type"`
	Active              bool                     `json:"active"`
	Distance            *RecordResponse          `json:"distance,omitempty"`
	AverageSpeed        *RecordResponse          `json:"average_speed,omitempty"`
	AverageSpeedNoPause *RecordResponse          `json:"average_speed_no_pause,omitempty"`
	MaxSpeed            *RecordResponse          `json:"max_speed,omitempty"`
	Duration            *RecordResponse          `json:"duration,omitempty"`
	TotalUp             *RecordResponse          `json:"total_up,omitempty"`
	DistanceRecords     []DistanceRecordResponse `json:"distance_records,omitempty"`
	BiggestClimb        *ClimbRecordResponse     `json:"biggest_climb,omitempty"`
}

// DistanceRecordResponse represents a distance effort record
type DistanceRecordResponse struct {
	Label           string    `json:"label"`
	TargetDistance  float64   `json:"target_distance"`
	Distance        float64   `json:"distance"`
	DurationSeconds float64   `json:"duration_seconds"`
	AverageSpeed    float64   `json:"average_speed"`
	WorkoutID       uint64    `json:"workout_id"`
	Date            time.Time `json:"date"`
	StartIndex      int       `json:"start_index,omitempty"`
	EndIndex        int       `json:"end_index,omitempty"`
}

// ClimbRecordResponse represents the biggest detected climb
type ClimbRecordResponse struct {
	ElevationGain float64   `json:"elevation_gain"`
	Distance      float64   `json:"distance"`
	AverageSlope  float64   `json:"average_slope"`
	WorkoutID     uint64    `json:"workout_id"`
	Date          time.Time `json:"date"`
	StartIndex    int       `json:"start_index,omitempty"`
	EndIndex      int       `json:"end_index,omitempty"`
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

	if len(wr.DistanceRecords) > 0 {
		response.DistanceRecords = make([]DistanceRecordResponse, 0, len(wr.DistanceRecords))
		for _, dr := range wr.DistanceRecords {
			if !dr.Active {
				continue
			}

			response.DistanceRecords = append(response.DistanceRecords, DistanceRecordResponse{
				Label:           dr.Label,
				TargetDistance:  dr.TargetDistance,
				Distance:        dr.Distance,
				DurationSeconds: dr.Duration.Seconds(),
				AverageSpeed:    dr.AverageSpeed,
				WorkoutID:       dr.WorkoutID,
				Date:            dr.Date,
				StartIndex:      dr.StartIndex,
				EndIndex:        dr.EndIndex,
			})
		}
	}

	if wr.BiggestClimb != nil && wr.BiggestClimb.Active {
		response.BiggestClimb = &ClimbRecordResponse{
			ElevationGain: wr.BiggestClimb.ElevationGain,
			Distance:      wr.BiggestClimb.Distance,
			AverageSlope:  wr.BiggestClimb.AverageSlope,
			WorkoutID:     wr.BiggestClimb.WorkoutID,
			Date:          wr.BiggestClimb.Date,
			StartIndex:    wr.BiggestClimb.StartIndex,
			EndIndex:      wr.BiggestClimb.EndIndex,
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

// NewDistanceRecordResponses converts a slice of database distance records to API responses.
func NewDistanceRecordResponses(records []database.DistanceRecord) []DistanceRecordResponse {
	if len(records) == 0 {
		return []DistanceRecordResponse{}
	}

	results := make([]DistanceRecordResponse, 0, len(records))
	for _, dr := range records {
		if !dr.Active {
			continue
		}

		results = append(results, DistanceRecordResponse{
			Label:           dr.Label,
			TargetDistance:  dr.TargetDistance,
			Distance:        dr.Distance,
			DurationSeconds: dr.Duration.Seconds(),
			AverageSpeed:    dr.AverageSpeed,
			WorkoutID:       dr.WorkoutID,
			Date:            dr.Date,
			StartIndex:      dr.StartIndex,
			EndIndex:        dr.EndIndex,
		})
	}

	return results
}
