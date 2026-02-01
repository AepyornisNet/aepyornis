package api

import (
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
)

// EquipmentResponse represents equipment in API v2 responses
type EquipmentResponse struct {
	ID          uint64               `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	Notes       string               `json:"notes,omitempty"`
	Active      bool                 `json:"active"`
	DefaultFor  []string             `json:"default_for,omitempty"`
	Usage       *EquipmentUsageStats `json:"usage,omitempty"`
	UserID      uint64               `json:"user_id"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}

// EquipmentUsageStats represents aggregated usage stats for equipment
type EquipmentUsageStats struct {
	Workouts        int        `json:"workouts"`
	Distance        float64    `json:"distance"`
	DurationSeconds float64    `json:"duration_seconds"`
	Repetitions     int        `json:"repetitions"`
	LastUsedAt      *time.Time `json:"last_used_at,omitempty"`
}

// NewEquipmentResponse converts a database equipment to API response
func NewEquipmentResponse(e *database.Equipment) EquipmentResponse {
	defaultFor := make([]string, len(e.DefaultFor))
	for i, wt := range e.DefaultFor {
		defaultFor[i] = string(wt)
	}

	return EquipmentResponse{
		ID:          e.ID,
		Name:        e.Name,
		Description: e.Description,
		Notes:       e.Notes,
		Active:      e.Active,
		DefaultFor:  defaultFor,
		Usage:       nil,
		UserID:      e.UserID,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

// NewEquipmentDetailResponse converts a database equipment to API response with usage stats.
func NewEquipmentDetailResponse(e *database.Equipment) EquipmentResponse {
	response := NewEquipmentResponse(e)
	response.Usage = buildEquipmentUsageStats(e)
	return response
}

func buildEquipmentUsageStats(e *database.Equipment) *EquipmentUsageStats {
	workoutsCount := len(e.Workouts)
	usage := &EquipmentUsageStats{
		Workouts: workoutsCount,
	}

	if workoutsCount == 0 {
		return usage
	}

	if totals, err := e.GetTotals(); err == nil {
		usage.Distance = totals.Distance
		usage.DurationSeconds = totals.Duration.Seconds()
		usage.Repetitions = totals.Repetitions
	}

	var lastUsed *time.Time
	for _, workout := range e.Workouts {
		workoutDate := workout.Date
		if lastUsed == nil || workoutDate.After(*lastUsed) {
			lastUsed = &workoutDate
		}
	}
	usage.LastUsedAt = lastUsed

	return usage
}

// NewEquipmentListResponse converts database equipment list to API responses
func NewEquipmentListResponse(es []*database.Equipment) []EquipmentResponse {
	results := make([]EquipmentResponse, len(es))
	for i, e := range es {
		results[i] = NewEquipmentResponse(e)
	}
	return results
}
