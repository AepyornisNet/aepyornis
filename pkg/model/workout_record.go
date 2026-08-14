package model

import (
	"database/sql"
	"math"
	"time"

	"github.com/paulmach/orb"
	"github.com/restayway/gogis"
	"github.com/tkrajina/gpxgo/gpx"
)

type WorkoutRecord struct {
	WorkoutID uint64 `gorm:"not null;primaryKey;index:idx_workout_records_parent_order,unique" json:"-"`
	SortOrder int    `gorm:"not null;primaryKey;index:idx_workout_records_parent_order,unique" json:"-"`

	Time time.Time `json:"time"` // The time the point was recorded

	ExtraMetrics    ExtraMetrics  `json:"extraMetrics"`                           // Extra metrics at this point
	Point           gogis.Point   `gorm:"type:geometry(Point,4326)" json:"point"` // The location of the point
	Elevation       float64       `json:"elevation"`                              // The elevation of the point
	Distance        float64       `json:"distance"`                               // The distance from the previous point
	Distance2D      float64       `json:"distance2D"`                             // The 2D distance from the previous point
	TotalDistance   float64       `json:"totalDistance"`                          // The total distance of the workout up to this point
	TotalDistance2D float64       `json:"totalDistance2D"`                        // The total 2D distance of the workout up to this point
	Duration        time.Duration `json:"duration"`                               // The duration from the previous point
	TotalDuration   time.Duration `json:"totalDuration"`                          // The total duration of the workout up to this point
	SlopeGrade      float64       `json:"slopeGrade"`                             // The grade of the slope at this point
	Pause           sql.NullBool  `json:"is_pause"`                               // Indicates whether this entry is within a pause
}

func (WorkoutRecord) TableName() string {
	return "workout_records"
}

func (m *WorkoutRecord) Lat() float64 {
	return m.Point.Lat
}

func (m *WorkoutRecord) Lng() float64 {
	return m.Point.Lng
}

func (m *WorkoutRecord) ToOrbPoint() *orb.Point {
	return &orb.Point{m.Point.Lng, m.Point.Lat}
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

// PointDistance calculates 2D distance between two gogis points in meters.
func PointDistance(p1, p2 gogis.Point) float64 {
	pt1 := gpx.Point{Latitude: p1.Lat, Longitude: p1.Lng}
	pt2 := gpx.Point{Latitude: p2.Lat, Longitude: p2.Lng}

	return pt1.Distance2D(&pt2)
}

func (m *WorkoutRecord) DistanceTo(m2 *WorkoutRecord) float64 {
	if m == nil || m2 == nil {
		return math.Inf(1)
	}

	return PointDistance(m.Point, m2.Point)
}

func (m *WorkoutRecord) DistanceToPoint(p gogis.Point) float64 {
	if m == nil {
		return math.Inf(1)
	}

	return PointDistance(m.Point, p)
}

func (m *WorkoutRecord) AsGPXPoint() *gpx.Point {
	ele := gpx.NewNullableFloat64(m.Elevation)

	return &gpx.Point{Latitude: m.Point.Lat, Longitude: m.Point.Lng, Elevation: *ele}
}
