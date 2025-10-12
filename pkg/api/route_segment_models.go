package api

import (
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
)

// RouteSegmentResponse represents a route segment in API v2 responses
type RouteSegmentResponse struct {
	ID            uint64    `json:"id"`
	Name          string    `json:"name"`
	Notes         string    `json:"notes,omitempty"`
	Filename      string    `json:"filename"`
	TotalDistance float64   `json:"total_distance"`
	MinElevation  float64   `json:"min_elevation"`
	MaxElevation  float64   `json:"max_elevation"`
	TotalUp       float64   `json:"total_up"`
	TotalDown     float64   `json:"total_down"`
	Bidirectional bool      `json:"bidirectional"`
	Circular      bool      `json:"circular"`
	MatchCount    int       `json:"match_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// NewRouteSegmentResponse converts a database route segment to API response
func NewRouteSegmentResponse(rs *database.RouteSegment) RouteSegmentResponse {
	matchCount := len(rs.RouteSegmentMatches)

	return RouteSegmentResponse{
		ID:            rs.ID,
		Name:          rs.Name,
		Notes:         rs.Notes,
		Filename:      rs.Filename,
		TotalDistance: rs.TotalDistance,
		MinElevation:  rs.MinElevation,
		MaxElevation:  rs.MaxElevation,
		TotalUp:       rs.TotalUp,
		TotalDown:     rs.TotalDown,
		Bidirectional: rs.Bidirectional,
		Circular:      rs.Circular,
		MatchCount:    matchCount,
		CreatedAt:     rs.CreatedAt,
		UpdatedAt:     rs.UpdatedAt,
	}
}

// NewRouteSegmentsResponse converts database route segments to API responses
func NewRouteSegmentsResponse(rss []*database.RouteSegment) []RouteSegmentResponse {
	results := make([]RouteSegmentResponse, len(rss))
	for i, rs := range rss {
		results[i] = NewRouteSegmentResponse(rs)
	}
	return results
}

// MapPoint represents a GPS point on a route segment
type MapPoint struct {
	Lat           float64 `json:"lat"`
	Lng           float64 `json:"lng"`
	Elevation     float64 `json:"elevation"`
	TotalDistance float64 `json:"total_distance"`
}

// RouteSegmentMatch represents a match of a route segment in a workout
type RouteSegmentMatch struct {
	WorkoutID    uint64  `json:"workout_id"`
	WorkoutName  string  `json:"workout_name"`
	UserID       uint64  `json:"user_id"`
	UserName     string  `json:"user_name"`
	Distance     float64 `json:"distance"`
	Duration     int     `json:"duration"`
	AverageSpeed float64 `json:"average_speed"`
}

// RouteSegmentDetailResponse represents a detailed route segment with map data and matches
type RouteSegmentDetailResponse struct {
	RouteSegmentResponse
	Points  []MapPoint          `json:"points"`
	Matches []RouteSegmentMatch `json:"matches"`
	Center  struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"center"`
	AddressString string `json:"address_string"`
}

// NewRouteSegmentDetailResponse converts a database route segment to detailed API response
func NewRouteSegmentDetailResponse(rs *database.RouteSegment) RouteSegmentDetailResponse {
	response := RouteSegmentDetailResponse{
		RouteSegmentResponse: NewRouteSegmentResponse(rs),
		AddressString:        rs.Address(),
	}

	// Convert points
	response.Points = make([]MapPoint, len(rs.Points))
	for i, p := range rs.Points {
		response.Points[i] = MapPoint{
			Lat:           p.Lat,
			Lng:           p.Lng,
			Elevation:     p.Elevation,
			TotalDistance: p.TotalDistance,
		}
	}

	// Set center
	response.Center.Lat = rs.Center.Lat
	response.Center.Lng = rs.Center.Lng

	// Convert matches
	response.Matches = make([]RouteSegmentMatch, len(rs.RouteSegmentMatches))
	for i, m := range rs.RouteSegmentMatches {
		response.Matches[i] = RouteSegmentMatch{
			WorkoutID:    m.WorkoutID,
			WorkoutName:  m.Workout.Name,
			UserID:       m.Workout.UserID,
			UserName:     m.Workout.User.Name,
			Distance:     m.Distance,
			Duration:     int(m.Duration.Seconds()),
			AverageSpeed: m.AverageSpeed(),
		}
	}

	return response
}
