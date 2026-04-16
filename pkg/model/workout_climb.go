package model

import "time"

// WorkoutClimb represents a detected climb or descent.
type WorkoutClimb struct {
	WorkoutID uint64 `gorm:"not null;primaryKey;index:idx_workout_climbs_parent_order,unique" json:"-"`
	SortOrder int    `gorm:"not null;primaryKey;index:idx_workout_climbs_parent_order,unique" json:"-"`

	Index    int           `json:"index"`
	Type     SlopeKind     `json:"type"`
	StartIdx int           `json:"start_idx"`
	Start    WorkoutRecord `gorm:"serializer:json" json:"start"`
	EndIdx   int           `json:"end_idx"`
	End      WorkoutRecord `gorm:"serializer:json" json:"end"`
	Gain     float64       `json:"gain,omitempty"`
	Length   float64       `json:"length_m"`
	AvgSlope float64       `json:"avg_slope"`
	Duration time.Duration `json:"duration"`
	Category Category      `json:"category"`
}

func (s *WorkoutClimb) IsClimb() bool {
	return s.Type == SlopeKindClimb
}
