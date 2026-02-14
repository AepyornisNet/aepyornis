package model

import (
	"math"
	"time"

	"gorm.io/gorm"
)

// WorkoutIntervalRecord stores best distance intervals per workout for ranking purposes.
type WorkoutIntervalRecord struct {
	Model
	WorkoutID       uint64  `gorm:"index:idx_workout_target"`
	Label           string  `gorm:"size:64"`
	TargetDistance  float64 `gorm:"index:idx_workout_target"`
	Distance        float64
	DurationSeconds float64
	AverageSpeed    float64
	StartIndex      int
	EndIndex        int
}

// WorkoutIntervalRecordWithRank represents a stored interval together with its computed rank.
type WorkoutIntervalRecordWithRank struct {
	WorkoutIntervalRecord
	Rank int64 `gorm:"column:rank"`
}

// GetWorkoutIntervalRecordsWithRank returns all stored interval records for the given workout with
// their rank computed on the fly for the owning user and workout type.
func GetWorkoutIntervalRecordsWithRank(db *gorm.DB, userID uint64, workoutType WorkoutType, workoutID uint64) ([]WorkoutIntervalRecordWithRank, error) {
	base := db.
		Table("workout_interval_records as wir").
		Select(`wir.*, RANK() OVER (
			PARTITION BY wir.label
			ORDER BY wir.duration_seconds ASC, wir.distance DESC, workouts.date ASC, wir.workout_id ASC
		) AS rank`).
		Joins("join workouts on workouts.id = wir.workout_id").
		Where("workouts.user_id = ?", userID).
		Where("workouts.type = ?", workoutType)

	rows := []WorkoutIntervalRecordWithRank{}
	if err := db.Table("(?) as ranked", base).
		Where("workout_id = ?", workoutID).
		Order("target_distance asc, duration_seconds asc, distance desc").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	return rows, nil
}

func betterDistanceRecord(a, b DistanceRecord) bool {
	if !b.Active {
		return true
	}

	if a.Duration == b.Duration {
		return a.Distance > b.Distance
	}

	return a.Duration < b.Duration
}

//nolint:gocyclo // sliding window search evaluates all targets in one pass
func fastestDistancesForWorkout(w *Workout, targets []DistanceRecordTarget) []DistanceRecord {
	if w == nil || w.Data == nil || w.Data.Details == nil || len(w.Data.Details.Points) < 2 {
		return nil
	}

	points := w.Data.Details.Points
	prefixDistance := make([]float64, len(points)+1)
	prefixMoving := make([]time.Duration, len(points)+1)

	for i, p := range points {
		prefixDistance[i+1] = prefixDistance[i] + p.Distance

		speed := p.AverageSpeed()
		if metricSpeed, ok := p.ExtraMetrics["speed"]; ok && !math.IsNaN(metricSpeed) && metricSpeed > 0 {
			speed = metricSpeed
		}

		if speed*3.6 >= 1.0 {
			prefixMoving[i+1] = prefixMoving[i] + p.Duration
		} else {
			prefixMoving[i+1] = prefixMoving[i]
		}
	}

	results := []DistanceRecord{}

	for _, target := range targets {
		best := DistanceRecord{Label: target.Label, TargetDistance: target.TargetDistance}
		start := 0
		for end := 0; end < len(points); end++ {
			for start <= end && prefixDistance[end+1]-prefixDistance[start] >= target.TargetDistance {
				dist := prefixDistance[end+1] - prefixDistance[start]
				dur := prefixMoving[end+1] - prefixMoving[start]

				if dur <= 0 {
					start++
					continue
				}

				candidate := DistanceRecord{
					Label:          target.Label,
					TargetDistance: target.TargetDistance,
					Distance:       dist,
					Duration:       dur,
					AverageSpeed:   dist / dur.Seconds(),
					WorkoutID:      w.ID,
					Date:           w.Date,
					StartIndex:     start,
					EndIndex:       end,
					Active:         true,
				}

				if !best.Active || betterDistanceRecord(candidate, best) {
					best = candidate
				}

				start++
			}
		}

		if best.Active {
			results = append(results, best)
		}
	}

	return results
}

func biggestClimbRecord(workouts []*Workout) *ClimbRecord {
	var best *ClimbRecord

	for _, w := range workouts {
		if w == nil || w.Data == nil {
			continue
		}

		for _, climb := range w.Data.Climbs {
			if climb.Type != "climb" {
				continue
			}

			candidate := ClimbRecord{
				ElevationGain: climb.Gain,
				Distance:      climb.Length,
				AverageSlope:  climb.AvgSlope,
				WorkoutID:     w.ID,
				Date:          w.Date,
				StartIndex:    climb.StartIdx,
				EndIndex:      climb.EndIdx,
				Active:        true,
			}

			if best == nil || candidate.ElevationGain > best.ElevationGain {
				best = &candidate
			}
		}
	}

	return best
}
