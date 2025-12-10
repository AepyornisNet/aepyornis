package api

import (
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
)

// MeasurementResponse represents a daily measurement in API v2 responses
type MeasurementResponse struct {
	ID               uint64    `json:"id"`
	Date             time.Time `json:"date"`
	Weight           *float64  `json:"weight,omitempty"`
	Height           *float64  `json:"height,omitempty"`
	Steps            *int      `json:"steps,omitempty"`
	FTP              *float64  `json:"ftp,omitempty"`
	RestingHeartRate *float64  `json:"resting_heart_rate,omitempty"`
	MaxHeartRate     *float64  `json:"max_heart_rate,omitempty"`
	UserID           uint64    `json:"user_id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// NewMeasurementResponse converts a database measurement to API response
func NewMeasurementResponse(m *database.Measurement) MeasurementResponse {
	mr := MeasurementResponse{
		ID:        m.ID,
		Date:      time.Time(m.Date),
		UserID:    m.UserID,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}

	if m.Weight != 0 {
		weight := m.Weight
		mr.Weight = &weight
	}
	if m.Height != 0 {
		height := m.Height
		mr.Height = &height
	}
	if m.Steps != 0 {
		steps := int(m.Steps)
		mr.Steps = &steps
	}
	if m.FTP != 0 {
		ftp := m.FTP
		mr.FTP = &ftp
	}
	if m.RestingHeartRate != 0 {
		rhr := m.RestingHeartRate
		mr.RestingHeartRate = &rhr
	}
	if m.MaxHeartRate != 0 {
		mhr := m.MaxHeartRate
		mr.MaxHeartRate = &mhr
	}

	return mr
}

// NewMeasurementsResponse converts database measurements to API responses
func NewMeasurementsResponse(ms []*database.Measurement) []MeasurementResponse {
	results := make([]MeasurementResponse, len(ms))
	for i, m := range ms {
		results[i] = NewMeasurementResponse(m)
	}
	return results
}
