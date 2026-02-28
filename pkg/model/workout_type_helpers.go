package model

import (
	"maps"
	"slices"
)

type (
	WorkoutType string
)

const (
	WorkoutTypeUnknown    WorkoutType = "unknown"
	WorkoutTypeAutoDetect WorkoutType = "auto"

	WorkoutTypeClassLocation   = "location"
	WorkoutTypeClassDistance   = "distance"
	WorkoutTypeClassRepetition = "repetition"
	WorkoutTypeClassWeight     = "weight"
	WorkoutTypeClassDuration   = "duration"
)

type WorkoutTypeConfiguration struct {
	Location   bool
	Distance   bool
	Repetition bool
	Weight     bool
}

var (
	workoutTypes        []WorkoutType
	workoutTypesByClass map[string][]WorkoutType
)

func WorkoutTypes() []WorkoutType {
	if len(workoutTypes) > 0 {
		return workoutTypes
	}

	workoutTypes = slices.Collect(maps.Keys(workoutTypeConfigs))

	slices.Sort(workoutTypes)

	return workoutTypes
}

func getOrSetByClass(class string, fn func(c WorkoutTypeConfiguration) bool) []WorkoutType {
	if workoutTypesByClass == nil {
		workoutTypesByClass = make(map[string][]WorkoutType)
	}

	if wt, ok := workoutTypesByClass[class]; ok {
		return wt
	}

	keys := []WorkoutType{}

	for k, c := range workoutTypeConfigs {
		if !fn(c) {
			continue
		}

		keys = append(keys, k)
	}

	slices.Sort(keys)
	workoutTypesByClass[class] = keys

	return keys
}

func DistanceWorkoutTypes() []WorkoutType {
	return getOrSetByClass(WorkoutTypeClassDistance, func(c WorkoutTypeConfiguration) bool {
		return c.Distance
	})
}

func WeightWorkoutTypes() []WorkoutType {
	return getOrSetByClass(WorkoutTypeClassWeight, func(c WorkoutTypeConfiguration) bool {
		return c.Weight
	})
}

func RepetitionWorkoutTypes() []WorkoutType {
	return getOrSetByClass(WorkoutTypeClassRepetition, func(c WorkoutTypeConfiguration) bool {
		return c.Repetition
	})
}

func LocationWorkoutTypes() []WorkoutType {
	return getOrSetByClass(WorkoutTypeClassLocation, func(c WorkoutTypeConfiguration) bool {
		return c.Location
	})
}

func DurationWorkoutTypes() []WorkoutType {
	return getOrSetByClass(WorkoutTypeClassDuration, func(c WorkoutTypeConfiguration) bool {
		return true // All workout types store duration
	})
}

func (wt WorkoutType) StringT() string {
	return "sports." + wt.String()
}

func (wt WorkoutType) String() string {
	if wt == "" {
		return string(WorkoutTypeUnknown)
	}

	return string(wt)
}

func (wt WorkoutType) IsDistance() bool {
	return workoutTypeConfigs[wt].Distance
}

func (wt WorkoutType) IsRepetition() bool {
	return workoutTypeConfigs[wt].Repetition
}

func (wt WorkoutType) IsDuration() bool {
	_, ok := workoutTypeConfigs[wt]
	return ok
}

func (wt WorkoutType) IsWeight() bool {
	return workoutTypeConfigs[wt].Weight
}

func (wt WorkoutType) IsLocation() bool {
	return workoutTypeConfigs[wt].Location
}

func AsWorkoutType(s string) WorkoutType {
	return WorkoutType(s)
}
