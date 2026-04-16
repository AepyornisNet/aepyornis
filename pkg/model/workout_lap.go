package model

import "time"

type WorkoutLap struct {
	WorkoutID uint64        `gorm:"not null;primaryKey;index:idx_workout_laps_parent_order,unique" json:"-"`
	SortOrder int           `gorm:"not null;primaryKey;index:idx_workout_laps_parent_order,unique" json:"-"`
	StatsID   *uint64       `json:"-"`
	Stats     *WorkoutStats `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:StatsID;references:ID" json:"stats,omitempty"`

	Start         time.Time     `json:"start"`         // The start time of the lap
	Stop          time.Time     `json:"stop"`          // The stop time of the lap
	TotalDistance float64       `json:"totalDistance"` // The total distance of the lap
	TotalDuration time.Duration `json:"totalDuration"` // The total duration of the lap
	PauseDuration time.Duration `json:"pauseDuration"` // The total pause duration of the lap
}
