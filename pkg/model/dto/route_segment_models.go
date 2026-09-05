package dto

import (
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/model"
)

type CourseRecordInfo struct {
	WorkoutID   uint64  `json:"workout_id"`
	WorkoutName string  `json:"workout_name"`
	ProfileID   uint64  `json:"profile_id"`
	ProfileName string  `json:"profile_name"`
	Duration    int     `json:"duration"`
	Speed       float64 `json:"speed"`
}

type RouteSegmentStatsResponse struct {
	TotalEfforts   int64             `json:"total_efforts"`
	UniqueAthletes int64             `json:"unique_athletes"`
	AvgDuration    float64           `json:"avg_duration"`
	AvgSpeed       float64           `json:"avg_speed"`
	CourseRecord   *CourseRecordInfo `json:"course_record,omitempty"`
}

// RouteSegmentResponse represents a route segment in API v2 responses
type RouteSegmentResponse struct {
	ID            uint64                       `json:"id"`
	ProfileID     uint64                       `json:"profile_id"`
	ProfileName   string                       `json:"profile_name,omitempty"`
	Name          string                       `json:"name"`
	Notes         string                       `json:"notes,omitempty"`
	Category      string                       `json:"category,omitempty"`
	Visibility    model.WorkoutVisibility      `json:"visibility"`
	Description   string                       `json:"description,omitempty"`
	Difficulty    model.RouteSegmentDifficulty `json:"difficulty,omitempty"`
	Filename      string                       `json:"filename"`
	TotalDistance float64                      `json:"total_distance"`
	MinElevation  float64                      `json:"min_elevation"`
	MaxElevation  float64                      `json:"max_elevation"`
	TotalUp       float64                      `json:"total_up"`
	TotalDown     float64                      `json:"total_down"`
	Bidirectional bool                         `json:"bidirectional"`
	Circular      bool                         `json:"circular"`
	MatchCount    int                          `json:"match_count"`
	LikeCount     int64                        `json:"like_count"`
	HasLiked      bool                         `json:"has_liked"`
	CanEdit       bool                         `json:"can_edit"`
	CanDelete     bool                         `json:"can_delete"`
	CreatedAt     time.Time                    `json:"created_at"`
	UpdatedAt     time.Time                    `json:"updated_at"`
}

// NewRouteSegmentResponse converts a database route segment to API response
func NewRouteSegmentResponse(rs *model.RouteSegment) RouteSegmentResponse {
	matchCount := len(rs.RouteSegmentMatches)

	profileName := ""
	if rs.Profile != nil {
		profileName = rs.Profile.DisplayName
	}

	visibility := rs.Visibility
	if visibility == "" {
		visibility = model.WorkoutVisibilityPublic
	}

	return RouteSegmentResponse{
		ID:            rs.ID,
		ProfileID:     rs.ProfileID,
		ProfileName:   profileName,
		Name:          rs.Name,
		Notes:         rs.Notes,
		Category:      rs.Category,
		Visibility:    visibility,
		Description:   rs.Description,
		Difficulty:    rs.Difficulty,
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
func NewRouteSegmentsResponse(rss []*model.RouteSegment) []RouteSegmentResponse {
	results := make([]RouteSegmentResponse, len(rss))
	for i, rs := range rss {
		results[i] = NewRouteSegmentResponse(rs)
	}
	return results
}

// MapPoint represents a GPS point on a route segment
type MapPoint struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// RouteSegmentMatch represents a match of a route segment in a workout
type RouteSegmentMatch struct {
	ID           uint64    `json:"id"`
	WorkoutID    uint64    `json:"workout_id"`
	WorkoutName  string    `json:"workout_name"`
	WorkoutDate  time.Time `json:"workout_date"`
	UserID       uint64    `json:"user_id,omitempty"`
	UserName     string    `json:"user_name,omitempty"`
	ProfileID    uint64    `json:"profile_id,omitempty"`
	ProfileName  string    `json:"profile_name,omitempty"`
	Distance     float64   `json:"distance"`
	Duration     int       `json:"duration"`
	AverageSpeed float64   `json:"average_speed"`
}

// RouteSegmentDetailResponse represents a detailed route segment with map data and matches
type RouteSegmentDetailResponse struct {
	RouteSegmentResponse
	Points        []MapPoint                 `json:"points"`
	Matches       []RouteSegmentMatch        `json:"matches"`
	Stats         *RouteSegmentStatsResponse `json:"stats,omitempty"`
	Center        struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"center"`
	AddressString *string `json:"address_string"`
}

type RouteSegmentsDetailResponse []*RouteSegmentResponse

// NewRouteSegmentDetailResponse converts a database route segment to detailed API response
func NewRouteSegmentDetailResponse(rs *model.RouteSegment) RouteSegmentDetailResponse {
	var addressString *string
	if addr := rs.Address(); addr != "" && addr != model.UnknownLocation {
		addressString = &addr
	}

	response := RouteSegmentDetailResponse{
		RouteSegmentResponse: NewRouteSegmentResponse(rs),
		AddressString:        addressString,
	}

	// Convert points
	response.Points = make([]MapPoint, len(rs.Points.Points))
	for i, p := range rs.Points.Points {
		response.Points[i] = MapPoint{
			Lat: p.Lat,
			Lng: p.Lng,
		}
	}

	// Set center
	response.Center.Lat = rs.Center.Lat
	response.Center.Lng = rs.Center.Lng

	// Convert matches
	response.Matches = make([]RouteSegmentMatch, len(rs.RouteSegmentMatches))
	for i, m := range rs.RouteSegmentMatches {
		response.Matches[i] = NewRouteSegmentMatchResponse(m)
	}

	return response
}

func NewRouteSegmentMatchResponse(m *model.RouteSegmentMatch) RouteSegmentMatch {
	if m == nil {
		return RouteSegmentMatch{}
	}

	var userID uint64
	var profileID uint64
	userName := ""
	profileName := ""
	workoutName := ""
	var workoutDate time.Time

	if m.Workout != nil {
		workoutName = m.Workout.Name
		workoutDate = m.Workout.GetDate()
		if m.Workout.Profile != nil {
			profileID = m.Workout.Profile.ID
			profileName = m.Workout.Profile.DisplayName
			if m.Workout.Profile.UserID != nil {
				userID = *m.Workout.Profile.UserID
			}
			userName = m.Workout.Profile.DisplayName
		}
	}

	return RouteSegmentMatch{
		ID:           m.ID,
		WorkoutID:    m.WorkoutID,
		WorkoutName:  workoutName,
		WorkoutDate:  workoutDate,
		UserID:       userID,
		UserName:     userName,
		ProfileID:    profileID,
		ProfileName:  profileName,
		Distance:     m.Distance,
		Duration:     int(m.Duration.Seconds()),
		AverageSpeed: m.AverageSpeed(),
	}
}
