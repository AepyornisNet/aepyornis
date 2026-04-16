package model

type WorkoutIntervalBest struct {
	Model
	WorkoutID       uint64  `gorm:"index:idx_workout_target"`
	Label           string  `gorm:"size:64"`
	TargetDistance  float64 `gorm:"index:idx_workout_target"`
	Distance        float64
	DurationSeconds float64
	Average         float64
	Type            WorkoutIntervalBestType `gorm:"size:16;not null;default:speed;index:idx_workout_target"`
	StartIndex      int
	EndIndex        int
}

func (WorkoutIntervalBest) TableName() string {
	return "workout_interval_records"
}
