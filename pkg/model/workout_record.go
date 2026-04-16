package model

import (
	"math"
	"time"

	"github.com/paulmach/orb"
	"github.com/tkrajina/gpxgo/gpx"
)

type WorkoutRecord struct {
	WorkoutID uint64 `gorm:"not null;primaryKey;index:idx_workout_records_parent_order,unique" json:"-"`
	SortOrder int    `gorm:"not null;primaryKey;index:idx_workout_records_parent_order,unique" json:"-"`

	Time time.Time `json:"time"` // The time the point was recorded

	ExtraMetrics    ExtraMetrics  `json:"extraMetrics"`    // Extra metrics at this point
	Lat             float64       `json:"lat"`             // The latitude of the point
	Lng             float64       `json:"lng"`             // The longitude of the point
	Elevation       float64       `json:"elevation"`       // The elevation of the point
	Distance        float64       `json:"distance"`        // The distance from the previous point
	Distance2D      float64       `json:"distance2D"`      // The 2D distance from the previous point
	TotalDistance   float64       `json:"totalDistance"`   // The total distance of the workout up to this point
	TotalDistance2D float64       `json:"totalDistance2D"` // The total 2D distance of the workout up to this point
	Duration        time.Duration `json:"duration"`        // The duration from the previous point
	TotalDuration   time.Duration `json:"totalDuration"`   // The total duration of the workout up to this point
	SlopeGrade      float64       `json:"slopeGrade"`      // The grade of the slope at this point
}

func (WorkoutRecord) TableName() string {
	return "workout_records"
}

func (m *WorkoutRecord) ToOrbPoint() *orb.Point {
	return &orb.Point{m.Lng, m.Lat}
}

func (m *WorkoutRecord) AverageSpeed() float64 {
	if m.Duration.Seconds() == 0 {
		return 0
	}

	return m.Distance / m.Duration.Seconds()
}

func (m *WorkoutRecord) EnhancedElevation() float64 {
	if v, ok := m.ExtraMetrics["elevation"]; ok && !math.IsNaN(v) {
		return v
	}

	return m.Elevation
}

func (m *WorkoutRecord) DistanceTo(m2 *WorkoutRecord) float64 {
	if m == nil || m2 == nil {
		return math.Inf(1)
	}

	return m.AsGPXPoint().Distance2D(m2.AsGPXPoint())
}

func (m *WorkoutRecord) AsGPXPoint() *gpx.Point {
	ele := gpx.NewNullableFloat64(m.Elevation)

	return &gpx.Point{Latitude: m.Lat, Longitude: m.Lng, Elevation: *ele}
}
