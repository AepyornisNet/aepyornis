package model

// WorkoutStats stores aggregated statistics for a workout or lap.
type WorkoutStats struct {
	Model

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
