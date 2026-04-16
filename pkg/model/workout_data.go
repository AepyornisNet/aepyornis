package model

import "time"

type (
	WorkoutData struct {
		WorkoutStats
	}

	WorkoutLap struct {
		WorkoutID uint64 `gorm:"not null;primaryKey;index:idx_workout_laps_parent_order,unique" json:"-"`
		SortOrder int    `gorm:"not null;primaryKey;index:idx_workout_laps_parent_order,unique" json:"-"`

		WorkoutStats
		Start         time.Time     `json:"start"`         // The start time of the lap
		Stop          time.Time     `json:"stop"`          // The stop time of the lap
		TotalDistance float64       `json:"totalDistance"` // The total distance of the lap
		TotalDuration time.Duration `json:"totalDuration"` // The total duration of the lap
		PauseDuration time.Duration `json:"pauseDuration"` // The total pause duration of the lap
	}

	WorkoutStats struct {
		// Elevation stats
		MinElevation float64 `json:"minElevation"` // The minimum elevation of the workout
		MaxElevation float64 `json:"maxElevation"` // The maximum elevation of the workout
		TotalUp      float64 `json:"totalUp"`      // The total distance up of the workout
		TotalDown    float64 `json:"totalDown"`    // The total distance down of the workout
		AverageSlope float64 `json:"averageSlope"` // The average slope of the workout
		MinSlope     float64 `json:"minSlope"`     // The minimum slope of the workout
		MaxSlope     float64 `json:"maxSlope"`     // The maximum slope of the workout

		// Speed stats
		AverageSpeed        float64 `json:"averageSpeed"`        // The average speed of the workout
		AverageSpeedNoPause float64 `json:"averageSpeedNoPause"` // The average speed of the workout without pausing
		MinSpeed            float64 `json:"minSpeed"`            // The minimum speed of the workout
		MaxSpeed            float64 `json:"maxSpeed"`            // The maximum speed of the workout

		// Cadence stats
		AverageCadence float64 `json:"averageCadence"` // The average cadence of the workout
		MinCadence     float64 `json:"minCadence"`     // The minimum cadence of the workout
		MaxCadence     float64 `json:"maxCadence"`     // The maximum cadence of the workout

		// Heart rate stats
		AverageHeartRate float64 `json:"averageHeartRate"` // The average heart rate of the workout
		MinHeartRate     float64 `json:"minHeartRate"`     // The minimum heart rate of the workout
		MaxHeartRate     float64 `json:"maxHeartRate"`     // The maximum heart rate of the workout

		// Respiration rate stats
		AverageRespirationRate float64 `json:"averageRespirationRate"` // The average respiration rate of the workout
		MinRespirationRate     float64 `json:"minRespirationRate"`     // The minimum respiration rate of the workout
		MaxRespirationRate     float64 `json:"maxRespirationRate"`     // The maximum respiration rate of the workout

		// Power stats
		AveragePower float64 `json:"averagePower"` // The average power of the workout
		MinPower     float64 `json:"minPower"`     // The minimum power of the workout
		MaxPower     float64 `json:"maxPower"`     // The maximum power of the workout

		// Temperature stats
		AverageTemperature float64 `json:"averageTemperature"` // The average temperature of the workout
		MinTemperature     float64 `json:"minTemperature"`     // The minimum temperature of the workout
		MaxTemperature     float64 `json:"maxTemperature"`     // The maximum temperature of the workout
	}
)

// MergeNonZero copies non-zero values from the provided data into the receiver.
// It intentionally skips zero-valued fields so partial updates do not wipe data.
//
//gocyclo:ignore
func (d *WorkoutData) MergeNonZero(from WorkoutData) {
	if d == nil {
		return
	}

	if from.MinElevation != 0 {
		d.MinElevation = from.MinElevation
	}

	if from.MaxElevation != 0 {
		d.MaxElevation = from.MaxElevation
	}

	if from.TotalUp != 0 {
		d.TotalUp = from.TotalUp
	}

	if from.TotalDown != 0 {
		d.TotalDown = from.TotalDown
	}

	if from.AverageSpeed != 0 {
		d.AverageSpeed = from.AverageSpeed
	}

	if from.AverageSpeedNoPause != 0 {
		d.AverageSpeedNoPause = from.AverageSpeedNoPause
	}

	if from.MaxSpeed != 0 {
		d.MaxSpeed = from.MaxSpeed
	}

	if from.AverageCadence != 0 {
		d.AverageCadence = from.AverageCadence
	}

	if from.MaxCadence != 0 {
		d.MaxCadence = from.MaxCadence
	}

	if from.AverageHeartRate != 0 {
		d.AverageHeartRate = from.AverageHeartRate
	}

	if from.MaxHeartRate != 0 {
		d.MaxHeartRate = from.MaxHeartRate
	}

	if from.AverageRespirationRate != 0 {
		d.AverageRespirationRate = from.AverageRespirationRate
	}

	if from.MaxRespirationRate != 0 {
		d.MaxRespirationRate = from.MaxRespirationRate
	}

	if from.AveragePower != 0 {
		d.AveragePower = from.AveragePower
	}

	if from.MaxPower != 0 {
		d.MaxPower = from.MaxPower
	}

	if from.AverageTemperature != 0 {
		d.AverageTemperature = from.AverageTemperature
	}

	if from.MaxTemperature != 0 {
		d.MaxTemperature = from.MaxTemperature
	}
}
