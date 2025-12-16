package api

import (
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jovandeginste/workout-tracker/v2/pkg/converters"
	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
	"github.com/jovandeginste/workout-tracker/v2/pkg/templatehelpers"
)

// WorkoutResponse represents a workout in API v2 responses
type WorkoutResponse struct {
	ID         uint64               `json:"id"`
	Date       time.Time            `json:"date"`
	Name       string               `json:"name"`
	Notes      string               `json:"notes"`
	Type       string               `json:"type"`
	SubType    string               `json:"sub_type"`
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
	AverageCadence      *float64 `json:"average_cadence,omitempty"`
	MaxCadence          *float64 `json:"max_cadence,omitempty"`
	AverageHeartRate    *float64 `json:"average_heart_rate,omitempty"`
	MaxHeartRate        *float64 `json:"max_heart_rate,omitempty"`
	AveragePower        *float64 `json:"average_power,omitempty"`
	MaxPower            *float64 `json:"max_power,omitempty"`
}

type WorkoutLapResponse struct {
	Start               time.Time `json:"start"`
	Stop                time.Time `json:"stop"`
	TotalDistance       float64   `json:"total_distance"`
	TotalDuration       int64     `json:"total_duration"`
	PauseDuration       int64     `json:"pause_duration"`
	MinElevation        float64   `json:"min_elevation"`
	MaxElevation        float64   `json:"max_elevation"`
	TotalUp             float64   `json:"total_up"`
	TotalDown           float64   `json:"total_down"`
	AverageSpeed        float64   `json:"average_speed"`
	AverageSpeedNoPause float64   `json:"average_speed_no_pause"`
	MaxSpeed            float64   `json:"max_speed"`
	AverageCadence      float64   `json:"average_cadence"`
	MaxCadence          float64   `json:"max_cadence"`
	AverageHeartRate    float64   `json:"average_heart_rate"`
	MaxHeartRate        float64   `json:"max_heart_rate"`
	AveragePower        float64   `json:"average_power"`
	MaxPower            float64   `json:"max_power"`
}

type WorkoutBreakdownResponse struct {
	Mode  string                         `json:"mode"` // "laps" or "unit"
	Items []WorkoutBreakdownItemResponse `json:"items,omitempty"`
}

type WorkoutBreakdownItemResponse struct {
	StartIndex int `json:"start_index"`
	EndIndex   int `json:"end_index"`

	Distance    float64 `json:"distance"`     // meters
	Duration    float64 `json:"duration"`     // moving duration in seconds
	AveragePace float64 `json:"average_pace"` // seconds per preferred unit

	MinElevation float64 `json:"min_elevation"`
	MaxElevation float64 `json:"max_elevation"`
	TotalUp      float64 `json:"total_up"`
	TotalDown    float64 `json:"total_down"`

	AverageSpeed        float64 `json:"average_speed"`
	AverageSpeedNoPause float64 `json:"average_speed_no_pause"`
	MaxSpeed            float64 `json:"max_speed"`

	AverageCadence float64 `json:"average_cadence"`
	MaxCadence     float64 `json:"max_cadence"`

	AverageHeartRate float64 `json:"average_heart_rate"`
	MaxHeartRate     float64 `json:"max_heart_rate"`

	AveragePower float64 `json:"average_power"`
	MaxPower     float64 `json:"max_power"`

	IsBest  bool `json:"is_best"`
	IsWorst bool `json:"is_worst"`
}

// WorkoutDetailResponse represents a detailed workout in API v2 responses
type WorkoutDetailResponse struct {
	WorkoutResponse
	Equipment           []EquipmentResponse         `json:"equipment,omitempty"`
	MapData             *MapDataResponse            `json:"map_data,omitempty"`
	Climbs              []ClimbSegmentResponse      `json:"climbs,omitempty"`
	RouteSegmentMatches []RouteSegmentMatchResponse `json:"route_segment_matches,omitempty"`
	Laps                []WorkoutLapResponse        `json:"laps,omitempty"`
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
	Position     [][]float64                    `json:"position"` // [[lat, lng], ...]
	Time         []time.Time                    `json:"time"`
	Distance     []float64                      `json:"distance"` // in km
	Duration     []float64                      `json:"duration"` // in seconds
	Speed        []float64                      `json:"speed"`    // in m/s
	Slope        []float64                      `json:"slope"`
	Elevation    []float64                      `json:"elevation"`
	ExtraMetrics map[string][]any               `json:"extra_metrics,omitempty"` // Additional metrics like heart-rate, cadence, temperature
	ZoneRanges   map[string][]ZoneRangeResponse `json:"zone_ranges,omitempty"`
}

// ZoneRangeResponse describes the absolute bounds of a training zone for display purposes.
type ZoneRangeResponse struct {
	Zone int      `json:"zone"`
	Min  float64  `json:"min"`
	Max  *float64 `json:"max,omitempty"`
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
		wr.SubType = w.Data.SubType
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
		wr.AverageCadence = &w.Data.AverageCadence
		wr.MaxCadence = &w.Data.MaxCadence
		wr.AverageHeartRate = &w.Data.AverageHeartRate
		wr.MaxHeartRate = &w.Data.MaxHeartRate
		wr.AveragePower = &w.Data.AveragePower
		wr.MaxPower = &w.Data.MaxPower

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

	if w.Data != nil && len(w.Data.Laps) > 1 {
		wr.Laps = NewWorkoutLapResponses(w.Data.Laps)
	}

	return wr
}

func NewWorkoutLapResponses(laps []converters.WorkoutLap) []WorkoutLapResponse {
	if len(laps) == 0 {
		return nil
	}

	resp := make([]WorkoutLapResponse, len(laps))
	for i, lap := range laps {
		resp[i] = WorkoutLapResponse{
			Start:               lap.Start,
			Stop:                lap.Stop,
			TotalDistance:       lap.TotalDistance,
			TotalDuration:       int64(lap.TotalDuration.Seconds()),
			PauseDuration:       int64(lap.PauseDuration.Seconds()),
			MinElevation:        lap.MinElevation,
			MaxElevation:        lap.MaxElevation,
			TotalUp:             lap.TotalUp,
			TotalDown:           lap.TotalDown,
			AverageSpeed:        lap.AverageSpeed,
			AverageSpeedNoPause: lap.AverageSpeedNoPause,
			MaxSpeed:            lap.MaxSpeed,
			AverageCadence:      lap.AverageCadence,
			MaxCadence:          lap.MaxCadence,
			AverageHeartRate:    lap.AverageHeartRate,
			MaxHeartRate:        lap.MaxHeartRate,
			AveragePower:        lap.AveragePower,
			MaxPower:            lap.MaxPower,
		}
	}

	return resp
}

func NewWorkoutBreakdownItemsFromLaps(laps []converters.WorkoutLap, points []database.MapPoint, units *database.UserPreferredUnits) []WorkoutBreakdownItemResponse {
	if len(laps) == 0 {
		return nil
	}

	items := make([]WorkoutBreakdownItemResponse, len(laps))

	for i, lap := range laps {
		startIdx := findClosestPointIndex(points, lap.Start)
		endIdx := findClosestPointIndex(points, lap.Stop)

		totalDuration := lap.TotalDuration.Seconds()
		pauseDuration := lap.PauseDuration.Seconds()
		movingDuration := totalDuration - pauseDuration
		if movingDuration < 0 {
			movingDuration = totalDuration
		}

		avgSpeed := 0.0
		avgSpeedNoPause := 0.0
		if totalDuration > 0 {
			avgSpeed = lap.TotalDistance / totalDuration
		}
		if movingDuration > 0 {
			avgSpeedNoPause = lap.TotalDistance / movingDuration
		}

		convertedDistance := convertDistanceToPreferred(lap.TotalDistance, units)
		pace := 0.0
		if convertedDistance > 0 {
			pace = movingDuration / convertedDistance
		}

		items[i] = WorkoutBreakdownItemResponse{
			StartIndex:          startIdx,
			EndIndex:            endIdx,
			Distance:            convertedDistance,
			Duration:            movingDuration,
			AveragePace:         pace,
			MinElevation:        convertElevationToPreferred(lap.MinElevation, units),
			MaxElevation:        convertElevationToPreferred(lap.MaxElevation, units),
			TotalUp:             convertElevationToPreferred(lap.TotalUp, units),
			TotalDown:           convertElevationToPreferred(lap.TotalDown, units),
			AverageSpeed:        convertSpeedToPreferred(avgSpeed, units),
			AverageSpeedNoPause: convertSpeedToPreferred(avgSpeedNoPause, units),
			MaxSpeed:            convertSpeedToPreferred(lap.MaxSpeed, units),
			AverageCadence:      lap.AverageCadence,
			MaxCadence:          lap.MaxCadence,
			AverageHeartRate:    lap.AverageHeartRate,
			MaxHeartRate:        lap.MaxHeartRate,
			AveragePower:        lap.AveragePower,
			MaxPower:            lap.MaxPower,
		}
	}

	return items
}

func NewWorkoutBreakdownItemsFromUnit(items []database.BreakdownItem, unit string, count float64, units *database.UserPreferredUnits) []WorkoutBreakdownItemResponse {
	if len(items) == 0 {
		return nil
	}

	resp := make([]WorkoutBreakdownItemResponse, len(items))
	for i, item := range items {
		movingSeconds := item.Duration.Seconds()
		pauseSeconds := item.PauseDuration.Seconds()
		totalSeconds := item.TotalDuration.Seconds()
		if totalSeconds == 0 {
			totalSeconds = movingSeconds + pauseSeconds
		}

		avgSpeed := 0.0
		avgSpeedNoPause := 0.0
		if totalSeconds > 0 {
			avgSpeed = item.Distance / totalSeconds
		}
		if movingSeconds > 0 {
			avgSpeedNoPause = item.Distance / movingSeconds
		}

		convertedDistance := convertDistanceToPreferred(item.Distance, units)
		pace := 0.0
		if convertedDistance > 0 {
			pace = movingSeconds / convertedDistance
		}

		resp[i] = WorkoutBreakdownItemResponse{
			StartIndex:          item.StartIndex,
			EndIndex:            item.EndIndex,
			Distance:            convertedDistance,
			Duration:            movingSeconds,
			AveragePace:         pace,
			MinElevation:        convertElevationToPreferred(item.MinElevation, units),
			MaxElevation:        convertElevationToPreferred(item.MaxElevation, units),
			TotalUp:             convertElevationToPreferred(item.TotalUp, units),
			TotalDown:           convertElevationToPreferred(item.TotalDown, units),
			AverageSpeed:        convertSpeedToPreferred(avgSpeed, units),
			AverageSpeedNoPause: convertSpeedToPreferred(avgSpeedNoPause, units),
			MaxSpeed:            convertSpeedToPreferred(item.MaxSpeed, units),
			AverageCadence:      item.AverageCadence,
			MaxCadence:          item.MaxCadence,
			AverageHeartRate:    item.AverageHeartRate,
			MaxHeartRate:        item.MaxHeartRate,
			AveragePower:        item.AveragePower,
			MaxPower:            item.MaxPower,
			IsBest:              item.IsBest,
			IsWorst:             item.IsWorst,
		}
	}

	return resp
}

func convertDistanceToPreferred(distanceMeters float64, units *database.UserPreferredUnits) float64 {
	if units == nil {
		return distanceMeters
	}

	switch units.Distance() {
	case "mi":
		return distanceMeters / templatehelpers.MeterPerMile
	case "km":
		return distanceMeters / templatehelpers.MeterPerKM
	case "m":
		return distanceMeters
	default:
		return distanceMeters / templatehelpers.MeterPerKM
	}
}

func convertElevationToPreferred(elevationMeters float64, units *database.UserPreferredUnits) float64 {
	if units == nil {
		return elevationMeters
	}

	switch units.Elevation() {
	case "ft":
		return elevationMeters * templatehelpers.FeetPerMeter
	default:
		return elevationMeters
	}
}

func convertSpeedToPreferred(speedMS float64, units *database.UserPreferredUnits) float64 {
	if units == nil {
		return speedMS * 3.6
	}

	switch units.Speed() {
	case "mph":
		return speedMS * 3.6 * templatehelpers.MilesPerKM
	default:
		return speedMS * 3.6
	}
}

func findClosestPointIndex(points []database.MapPoint, t time.Time) int {
	if len(points) == 0 || t.IsZero() {
		return -1
	}

	bestIdx := -1
	bestDiff := time.Duration(math.MaxInt64)

	for i := range points {
		diff := points[i].Time.Sub(t)
		if diff < 0 {
			diff = -diff
		}
		if diff < bestDiff {
			bestDiff = diff
			bestIdx = i
		}
	}

	return bestIdx
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

		mapData.Details.ZoneRanges = zoneMetrics.zoneRanges()
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

	z.populateUserMetrics()

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

func buildHeartRateZoneRanges(maxHR float64, restHR float64) []ZoneRangeResponse {
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

	upperBounds := []float64{
		restHR + 0.6*reserve,
		restHR + 0.7*reserve,
		restHR + 0.8*reserve,
		restHR + 0.9*reserve,
	}

	return []ZoneRangeResponse{
		{Zone: 1, Min: restHR, Max: float64Ptr(upperBounds[0])},
		{Zone: 2, Min: upperBounds[0], Max: float64Ptr(upperBounds[1])},
		{Zone: 3, Min: upperBounds[1], Max: float64Ptr(upperBounds[2])},
		{Zone: 4, Min: upperBounds[2], Max: float64Ptr(upperBounds[3])},
		{Zone: 5, Min: upperBounds[3]},
	}
}

func buildFTPZoneRanges(ftp float64) []ZoneRangeResponse {
	if ftp <= 0 {
		ftp = 200
	}

	thresholds := []float64{
		0.55 * ftp,
		0.75 * ftp,
		0.9 * ftp,
		1.05 * ftp,
		1.2 * ftp,
		1.5 * ftp,
	}

	return []ZoneRangeResponse{
		{Zone: 1, Min: 0, Max: float64Ptr(thresholds[0])},
		{Zone: 2, Min: thresholds[0], Max: float64Ptr(thresholds[1])},
		{Zone: 3, Min: thresholds[1], Max: float64Ptr(thresholds[2])},
		{Zone: 4, Min: thresholds[2], Max: float64Ptr(thresholds[3])},
		{Zone: 5, Min: thresholds[3], Max: float64Ptr(thresholds[4])},
		{Zone: 6, Min: thresholds[4], Max: float64Ptr(thresholds[5])},
		{Zone: 7, Min: thresholds[5]},
	}
}

func float64Ptr(val float64) *float64 {
	v := val
	return &v
}

func (z *zoneMetricsBuilder) populateUserMetrics() {
	if z.user == nil {
		return
	}

	if z.maxHR == 0 {
		z.maxHR = z.user.MaxHeartRateAt(z.date)
	}
	if z.restHR == 0 {
		z.restHR = z.user.RestingHeartRateAt(z.date)
	}
	if z.ftp == 0 {
		z.ftp = z.user.FTPAt(z.date)
	}
}

func (z *zoneMetricsBuilder) zoneRanges() map[string][]ZoneRangeResponse {
	if !z.shouldBuild() {
		return nil
	}

	z.populateUserMetrics()
	ranges := make(map[string][]ZoneRangeResponse)

	if z.hasHR {
		ranges["heart-rate"] = buildHeartRateZoneRanges(z.maxHR, z.restHR)
	}

	if z.hasPower {
		ranges["power"] = buildFTPZoneRanges(z.ftp)
	}

	if len(ranges) == 0 {
		return nil
	}

	return ranges
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
