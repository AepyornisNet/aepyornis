package api

import (
	"time"

	"github.com/google/uuid"
	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
)

// WorkoutResponse represents a workout in API v2 responses
type WorkoutResponse struct {
	ID         uint64               `json:"id"`
	Date       time.Time            `json:"date"`
	Name       string               `json:"name"`
	Notes      string               `json:"notes"`
	Type       string               `json:"type"`
	CustomType string               `json:"custom_type,omitempty"`
	UserID     uint64               `json:"user_id"`
	User       *UserProfileResponse `json:"user,omitempty"`
	PublicUUID *uuid.UUID           `json:"public_uuid,omitempty"`
	Locked     bool                 `json:"locked"`
	CreatedAt  time.Time            `json:"created_at"`
	UpdatedAt  time.Time            `json:"updated_at"`
	HasFile    bool                 `json:"has_file"`
	HasTracks  bool                 `json:"has_tracks"`

	// MapData fields (when available)
	AddressString       string   `json:"address_string,omitempty"`
	TotalDistance       *float64 `json:"total_distance,omitempty"`
	TotalDuration       *int64   `json:"total_duration,omitempty"` // Duration in seconds
	TotalWeight         *float64 `json:"total_weight,omitempty"`
	TotalRepetitions    *int     `json:"total_repetitions,omitempty"`
	TotalUp             *float64 `json:"total_up,omitempty"`
	TotalDown           *float64 `json:"total_down,omitempty"`
	AverageSpeed        *float64 `json:"average_speed,omitempty"`
	AverageSpeedNoPause *float64 `json:"average_speed_no_pause,omitempty"`
	MaxSpeed            *float64 `json:"max_speed,omitempty"`
	MinElevation        *float64 `json:"min_elevation,omitempty"`
	MaxElevation        *float64 `json:"max_elevation,omitempty"`
	PauseDuration       *int64   `json:"pause_duration,omitempty"` // Duration in seconds
}

// WorkoutDetailResponse represents a detailed workout in API v2 responses
type WorkoutDetailResponse struct {
	WorkoutResponse
	Equipment           []EquipmentResponse         `json:"equipment,omitempty"`
	MapData             *MapDataResponse            `json:"map_data,omitempty"`
	Climbs              []ClimbSegmentResponse      `json:"climbs,omitempty"`
	RouteSegmentMatches []RouteSegmentMatchResponse `json:"route_segment_matches,omitempty"`
}

// MapDataResponse represents workout map data in API v2 responses
type MapDataResponse struct {
	Creator      string                  `json:"creator,omitempty"`
	Center       MapCenterResponse       `json:"center"`
	ExtraMetrics []string                `json:"extra_metrics,omitempty"`
	Details      *MapDataDetailsResponse `json:"details,omitempty"`
}

// MapCenterResponse represents the center coordinates
type MapCenterResponse struct {
	TZ  string  `json:"tz"`
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// MapDataDetailsResponse represents detailed map points in compact format
type MapDataDetailsResponse struct {
	Position     [][]float64      `json:"position"` // [[lat, lng], ...]
	Time         []time.Time      `json:"time"`
	Distance     []float64        `json:"distance"` // in km
	Duration     []float64        `json:"duration"` // in seconds
	Speed        []float64        `json:"speed"`    // in m/s
	Slope        []float64        `json:"slope"`
	Elevation    []float64        `json:"elevation"`
	ExtraMetrics map[string][]any `json:"extra_metrics,omitempty"` // Additional metrics like heart-rate, cadence, temperature
}

// ClimbSegmentResponse represents a climb or descent segment
type ClimbSegmentResponse struct {
	Index         int     `json:"index"`
	Type          string  `json:"type"`
	StartDistance float64 `json:"start_distance"`
	Length        float64 `json:"length"`
	Elevation     float64 `json:"elevation"`
	AvgSlope      float64 `json:"avg_slope"`
	Category      string  `json:"category"`
}

// RouteSegmentMatchResponse represents a matched route segment
type RouteSegmentMatchResponse struct {
	RouteSegmentID uint64               `json:"route_segment_id"`
	WorkoutID      uint64               `json:"workout_id"`
	RouteSegment   RouteSegmentResponse `json:"route_segment"`
}

// WorkoutPopupData represents data for the heatmap popup
type WorkoutPopupData struct {
	ID         uint64 `json:"id"`
	Name       string `json:"name"`
	Date       string `json:"date"`
	Type       string `json:"type"`
	CustomType string `json:"custom_type,omitempty"`
	Locked     bool   `json:"locked"`

	// Type-specific fields
	TotalDistance             *float64 `json:"total_distance,omitempty"`
	TotalDuration             *int64   `json:"total_duration,omitempty"`
	TotalRepetitions          *int     `json:"total_repetitions,omitempty"`
	RepetitionFrequencyPerMin *float64 `json:"repetition_frequency_per_min,omitempty"`
	TotalWeight               *float64 `json:"total_weight,omitempty"`
	AverageSpeed              *float64 `json:"average_speed,omitempty"`
}

// CalendarEventResponse represents a calendar event for a workout
type CalendarEventResponse struct {
	Title string    `json:"title"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	URL   string    `json:"url"`
}

// NewWorkoutResponse converts a database workout to API response
func NewWorkoutResponse(w *database.Workout) WorkoutResponse {
	wr := WorkoutResponse{
		ID:         w.ID,
		Date:       w.Date,
		Name:       w.Name,
		Notes:      w.Notes,
		Type:       string(w.Type),
		CustomType: w.CustomType,
		UserID:     w.UserID,
		PublicUUID: w.PublicUUID,
		Locked:     w.Locked,
		CreatedAt:  w.CreatedAt,
		UpdatedAt:  w.UpdatedAt,
		HasFile:    w.HasFile(),
		HasTracks:  w.HasTracks(),
	}

	// Add user data if available (preloaded)
	if w.User != nil {
		userResp := NewUserProfileResponse(w.User)
		wr.User = &userResp
	}

	// Add map data if available
	if w.Data != nil {
		wr.AddressString = w.Data.AddressString
		wr.TotalDistance = &w.Data.TotalDistance

		// Convert durations to seconds (int64)
		totalDurationSecs := int64(w.Data.TotalDuration.Seconds())
		wr.TotalDuration = &totalDurationSecs

		wr.TotalWeight = &w.Data.TotalWeight
		wr.TotalRepetitions = &w.Data.TotalRepetitions
		wr.TotalUp = &w.Data.TotalUp
		wr.TotalDown = &w.Data.TotalDown
		wr.AverageSpeed = &w.Data.AverageSpeed
		wr.AverageSpeedNoPause = &w.Data.AverageSpeedNoPause
		wr.MaxSpeed = &w.Data.MaxSpeed
		wr.MinElevation = &w.Data.MinElevation
		wr.MaxElevation = &w.Data.MaxElevation

		// Convert pause duration to seconds (int64)
		pauseDurationSecs := int64(w.Data.PauseDuration.Seconds())
		wr.PauseDuration = &pauseDurationSecs
	}

	return wr
}

// NewWorkoutsResponse converts database workouts to API responses
func NewWorkoutsResponse(ws []*database.Workout) []WorkoutResponse {
	results := make([]WorkoutResponse, len(ws))
	for i, w := range ws {
		results[i] = NewWorkoutResponse(w)
	}
	return results
}

// NewWorkoutPopupData converts a database workout to popup data for heatmap
func NewWorkoutPopupData(w *database.Workout) WorkoutPopupData {
	popup := WorkoutPopupData{
		ID:         w.ID,
		Name:       w.Name,
		Date:       w.Date.Format("2006-01-02"),
		Type:       string(w.Type),
		CustomType: w.CustomType,
		Locked:     w.Locked,
	}

	// Add type-specific fields
	if w.Type.IsDistance() && w.Data != nil {
		popup.TotalDistance = &w.Data.TotalDistance
	}

	if w.Type.IsDuration() && w.Data != nil {
		duration := int64(w.Data.TotalDuration.Seconds())
		popup.TotalDuration = &duration
	}

	if w.Type.IsRepetition() && w.Data != nil {
		popup.TotalRepetitions = &w.Data.TotalRepetitions
		repFreq := w.RepetitionFrequencyPerMinute()
		popup.RepetitionFrequencyPerMin = &repFreq
	}

	if w.Type.IsWeight() && w.Data != nil {
		popup.TotalWeight = &w.Data.TotalWeight
	}

	if w.Type.IsDistance() && w.Type.IsDuration() && w.Data != nil {
		popup.AverageSpeed = &w.Data.AverageSpeed
	}

	return popup
}

// NewWorkoutDetailResponse converts a database workout to a detailed API response
func NewWorkoutDetailResponse(w *database.Workout) WorkoutDetailResponse {
	wr := WorkoutDetailResponse{
		WorkoutResponse: NewWorkoutResponse(w),
	}

	// Add equipment
	if len(w.Equipment) > 0 {
		wr.Equipment = make([]EquipmentResponse, len(w.Equipment))
		for i, e := range w.Equipment {
			wr.Equipment[i] = NewEquipmentResponse(&e)
		}
	}

	// Add map data with details
	if w.Data != nil {
		// Add climbs
		if len(w.Data.Climbs) > 0 {
			wr.Climbs = make([]ClimbSegmentResponse, len(w.Data.Climbs))
			for i, climb := range w.Data.Climbs {
				wr.Climbs[i] = ClimbSegmentResponse{
					Index:         climb.Index,
					Type:          climb.Type,
					StartDistance: climb.StartDistance,
					Length:        climb.Length,
					Elevation:     climb.Elevation,
					AvgSlope:      climb.AvgSlope,
					Category:      climb.Category,
				}
			}
		}

		wr.MapData = workoutResponseMapData(w)
	}

	// Add route segment matches
	if len(w.RouteSegmentMatches) > 0 {
		wr.RouteSegmentMatches = make([]RouteSegmentMatchResponse, len(w.RouteSegmentMatches))
		for i, match := range w.RouteSegmentMatches {
			wr.RouteSegmentMatches[i] = RouteSegmentMatchResponse{
				RouteSegmentID: match.RouteSegmentID,
				WorkoutID:      match.WorkoutID,
				RouteSegment:   NewRouteSegmentResponse(match.RouteSegment),
			}
		}
	}

	return wr
}

func workoutResponseMapData(w *database.Workout) *MapDataResponse {
	mapData := &MapDataResponse{
		Creator: w.Data.Creator,
		Center: MapCenterResponse{
			TZ:  w.Data.Center.TZ,
			Lat: w.Data.Center.Lat,
			Lng: w.Data.Center.Lng,
		},
		ExtraMetrics: w.Data.ExtraMetrics,
	}

	// Add detailed points in compact format
	if w.Data.Details != nil && len(w.Data.Details.Points) > 0 {
		points := w.Data.Details.Points
		mapData.Details = &MapDataDetailsResponse{
			Position:     make([][]float64, len(points)),
			Time:         make([]time.Time, len(points)),
			Distance:     make([]float64, len(points)),
			Duration:     make([]float64, len(points)),
			Speed:        make([]float64, len(points)),
			Slope:        make([]float64, len(points)),
			Elevation:    make([]float64, len(points)),
			ExtraMetrics: make(map[string][]any),
		}

		zoneMetrics := newZoneMetricsBuilder(w)

		// Initialize extra metrics arrays
		for _, metric := range mapData.ExtraMetrics {
			if metric == "speed" || metric == "elevation" {
				continue
			}

			mapData.Details.ExtraMetrics[metric] = make([]any, len(points))
		}

		zoneMetrics.ensureBuffers(mapData.Details.ExtraMetrics, len(points))

		for i, point := range points {
			mapData.Details.Position[i] = []float64{point.Lat, point.Lng}
			mapData.Details.Time[i] = point.Time
			mapData.Details.Distance[i] = point.TotalDistance / 1000 // Convert to km
			mapData.Details.Duration[i] = point.TotalDuration.Seconds()
			mapData.Details.Slope[i] = point.SlopeGrade
			mapData.Details.Elevation[i] = point.Elevation

			// Calculate speed from extra metrics or derive it
			speed := point.AverageSpeed()
			if ems, ok := point.ExtraMetrics["speed"]; ok && ems > 0 {
				speed = ems
			}
			mapData.Details.Speed[i] = speed

			// Add extra metrics
			for _, metric := range w.Data.ExtraMetrics {
				if metric == "speed" || metric == "elevation" {
					continue // Already handled
				}
				if val, ok := point.ExtraMetrics[metric]; ok {
					mapData.Details.ExtraMetrics[metric][i] = val
				} else {
					mapData.Details.ExtraMetrics[metric][i] = nil
				}
			}

			zoneMetrics.setForPoint(i, point.ExtraMetrics)
		}
	}

	return mapData
}

const (
	hrZoneMetricName  = "hr-zone"
	ftpZoneMetricName = "zone"
)

type zoneMetricsBuilder struct {
	user     *database.User
	date     time.Time
	maxHR    float64
	restHR   float64
	ftp      float64
	hasHR    bool
	hasPower bool
	hrZones  []any
	ftpZones []any
}

func newZoneMetricsBuilder(w *database.Workout) *zoneMetricsBuilder {
	if w == nil || w.Data == nil {
		return &zoneMetricsBuilder{}
	}

	return &zoneMetricsBuilder{
		user:     w.User,
		date:     w.Date,
		maxHR:    0,
		restHR:   0,
		ftp:      0,
		hasHR:    w.HasHeartRate(),
		hasPower: w.HasExtraMetric("power"),
	}
}

func (z *zoneMetricsBuilder) shouldBuild() bool {
	return z.user != nil && (z.hasHR || z.hasPower)
}

func (z *zoneMetricsBuilder) ensureBuffers(extra map[string][]any, length int) {
	if !z.shouldBuild() {
		return
	}

	// Cache user-dependent metrics once
	if z.maxHR == 0 {
		z.maxHR = z.user.MaxHeartRateAt(z.date)
	}
	if z.restHR == 0 {
		z.restHR = z.user.RestingHeartRateAt(z.date)
	}
	if z.ftp == 0 {
		z.ftp = z.user.FTPAt(z.date)
	}

	if z.hasHR {
		hrBuf := make([]any, length)
		extra[hrZoneMetricName] = hrBuf
		z.hrZones = hrBuf
	}

	if z.hasPower {
		ftpBuf := make([]any, length)
		extra[ftpZoneMetricName] = ftpBuf
		z.ftpZones = ftpBuf
	}
}

func (z *zoneMetricsBuilder) setForPoint(idx int, metrics database.ExtraMetrics) {
	if !z.shouldBuild() {
		return
	}

	if z.hasHR && z.hrZones != nil {
		if hr, ok := metrics["heart-rate"]; ok && hr > 0 {
			z.hrZones[idx] = calculateHeartRateZone(hr, z.maxHR, z.restHR)
		} else {
			z.hrZones[idx] = nil
		}
	}

	if z.hasPower && z.ftpZones != nil {
		if power, ok := metrics["power"]; ok && power > 0 {
			z.ftpZones[idx] = calculateFTPZone(power, z.ftp)
		} else {
			z.ftpZones[idx] = nil
		}
	}
}

func calculateHeartRateZone(hr float64, maxHR float64, restHR float64) int {
	if maxHR <= 0 {
		maxHR = 200
	}

	if restHR <= 0 {
		restHR = 60
	}

	reserve := maxHR - restHR
	if reserve <= 0 {
		reserve = maxHR
	}

	percent := (hr - restHR) / reserve

	switch {
	case percent < 0.6:
		return 1
	case percent < 0.7:
		return 2
	case percent < 0.8:
		return 3
	case percent < 0.9:
		return 4
	default:
		return 5
	}
}

func calculateFTPZone(power float64, ftp float64) int {
	if ftp <= 0 {
		ftp = 200
	}

	ratio := power / ftp

	switch {
	case ratio < 0.55:
		return 1
	case ratio < 0.75:
		return 2
	case ratio < 0.9:
		return 3
	case ratio < 1.05:
		return 4
	case ratio < 1.2:
		return 5
	case ratio < 1.5:
		return 6
	default:
		return 7
	}
}
