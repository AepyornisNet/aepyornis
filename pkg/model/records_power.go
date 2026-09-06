package model

import (
	"math"
	"time"

	"gorm.io/gorm"
)

// PowerRecord represents a computed best power interval for a workout.
type PowerRecord struct {
	Label          string
	TargetDuration float64
	Distance       float64
	Duration       time.Duration
	AveragePower   float64
	AverageSpeed   float64
	WorkoutID      uint64
	Date           time.Time
	StartIndex     int
	EndIndex       int
	Active         bool
}

func betterPowerRecord(a, b PowerRecord) bool {
	if !b.Active {
		return true
	}

	if a.AveragePower == b.AveragePower {
		return a.Duration > b.Duration
	}

	return a.AveragePower > b.AveragePower
}

// GetWorkoutPowerIntervalRecordsWithRank returns all stored power interval records for the given workout
// with their rank computed on the fly for the owning user and workout type.
func GetWorkoutPowerIntervalRecordsWithRank(db *gorm.DB, profileID uint64, workoutType WorkoutType, workoutID uint64) ([]WorkoutIntervalRecordWithRank, error) {
	base := db.
		Table("workout_interval_records as wir").
		Select(`wir.*, RANK() OVER (
			PARTITION BY wir.type, wir.label
			ORDER BY wir.average DESC, wir.duration_seconds DESC, workouts.date ASC, wir.workout_id ASC
		) AS rank`).
		Joins("join workouts on workouts.id = wir.workout_id").
		Where("workouts.profile_id = ?", profileID).
		Where("workouts.type = ?", workoutType).
		Where("wir.type = ?", WorkoutIntervalBestTypePower)

	rows := []WorkoutIntervalRecordWithRank{}
	if err := db.Table("(?) as ranked", base).
		Where("workout_id = ?", workoutID).
		Order("target_distance asc, duration_seconds asc").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	return rows, nil
}

//nolint:gocyclo // sliding window search evaluates all targets in one pass
func bestPowerIntervalsForWorkout(w *Workout, targets []PowerRecordTarget) []PowerRecord {
	if w == nil || len(w.Records) < 2 {
		return nil
	}

	points := w.Records
	hasPower := false
	for _, p := range points {
		if val, ok := p.ExtraMetrics["power"]; ok && !math.IsNaN(val) {
			hasPower = true
			break
		}
	}
	if !hasPower {
		return nil
	}

	prefixDuration := make([]float64, len(points)+1)
	prefixEnergy := make([]float64, len(points)+1)
	prefixDistance := make([]float64, len(points)+1)

	for i, p := range points {
		dt := p.Duration
		if dt <= 0 {
			if !p.Time.IsZero() && i+1 < len(points) && !points[i+1].Time.IsZero() {
				diff := points[i+1].Time.Sub(p.Time)
				if diff > 0 {
					dt = diff
				}
			}
		}
		if dt <= 0 {
			dt = time.Second
		}

		dtSec := dt.Seconds()
		power := 0.0
		if val, ok := p.ExtraMetrics["power"]; ok && !math.IsNaN(val) && val >= 0 {
			power = val
		}

		prefixDuration[i+1] = prefixDuration[i] + dtSec
		prefixEnergy[i+1] = prefixEnergy[i] + (power * dtSec)
		prefixDistance[i+1] = prefixDistance[i] + p.Distance
	}

	results := make([]PowerRecord, 0, len(targets))

	for _, target := range targets {
		if target.TargetDuration <= 0 {
			continue
		}

		best := PowerRecord{
			Label:          target.Label,
			TargetDuration: target.TargetDuration,
		}

		end := 0
		for start := 0; start < len(points); start++ {
			for end < len(points) && prefixDuration[end+1]-prefixDuration[start] < target.TargetDuration {
				end++
			}
			if end >= len(points) {
				break
			}

			durSec := prefixDuration[end+1] - prefixDuration[start]
			if durSec <= 0 {
				continue
			}

			energy := prefixEnergy[end+1] - prefixEnergy[start]
			dist := prefixDistance[end+1] - prefixDistance[start]

			candidate := PowerRecord{
				Label:          target.Label,
				TargetDuration: target.TargetDuration,
				Distance:       dist,
				Duration:       time.Duration(durSec * float64(time.Second)),
				AveragePower:   energy / durSec,
				AverageSpeed:   dist / durSec,
				WorkoutID:      w.ID,
				Date:           w.Date,
				StartIndex:     start,
				EndIndex:       end,
				Active:         true,
			}

			if !best.Active || betterPowerRecord(candidate, best) {
				best = candidate
			}
		}

		if best.Active {
			results = append(results, best)
		}
	}

	return results
}
