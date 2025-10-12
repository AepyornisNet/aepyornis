package api

import (
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
)

// EquipmentResponse represents equipment in API v2 responses
type EquipmentResponse struct {
	ID          uint64    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Notes       string    `json:"notes,omitempty"`
	Active      bool      `json:"active"`
	DefaultFor  []string  `json:"default_for,omitempty"`
	UserID      uint64    `json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
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
		UserID:      e.UserID,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

// NewEquipmentListResponse converts database equipment list to API responses
func NewEquipmentListResponse(es []*database.Equipment) []EquipmentResponse {
	results := make([]EquipmentResponse, len(es))
	for i, e := range es {
		results[i] = NewEquipmentResponse(e)
	}
	return results
}
