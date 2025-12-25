package database

import "time"

type (
	WorkoutData struct {
		WorkoutStats
		Name             string        `json:"name"`                                // The name of the workout
		Type             string        `json:"type"`                                // The type of the workout
		SubType          string        `json:"subType"`                             // The subtype of the workout
		Start            time.Time     `json:"start"`                               // The start time of the workout
		Stop             time.Time     `json:"stop"`                                // The stop time of the workout
		TotalDistance    float64       `json:"totalDistance"`                       // The total distance of the workout
		TotalDuration    time.Duration `json:"totalDuration"`                       // The total duration of the workout
		PauseDuration    time.Duration `json:"pauseDuration"`                       // The total pause duration of the workout
		TotalRepetitions int           `json:"totalRepetitions"`                    // The number of repetitions of the workout
		TotalWeight      float64       `json:"totalWeight"`                         // The weight of the workout
		Laps             []WorkoutLap  `gorm:"serializer:json" json:"laps"`         // The laps of the workout
		ExtraMetrics     []string      `gorm:"serializer:json" json:"extraMetrics"` // Extra metrics available
	}

	WorkoutLap struct {
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

	if !from.Start.IsZero() {
		d.Start = from.Start
	}

	if !from.Stop.IsZero() {
		d.Stop = from.Stop
	}

	if from.SubType != "" {
		d.SubType = from.SubType
	}

	if from.TotalDistance != 0 {
		d.TotalDistance = from.TotalDistance
	}

	if from.TotalDuration != 0 {
		d.TotalDuration = from.TotalDuration
	}

	if from.PauseDuration != 0 {
		d.PauseDuration = from.PauseDuration
	}

	if from.TotalRepetitions != 0 {
		d.TotalRepetitions = from.TotalRepetitions
	}

	if from.TotalWeight != 0 {
		d.TotalWeight = from.TotalWeight
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

	if from.AveragePower != 0 {
		d.AveragePower = from.AveragePower
	}

	if from.MaxPower != 0 {
		d.MaxPower = from.MaxPower
	}

	if len(from.Laps) > 0 {
		d.Laps = from.Laps
	}
}
