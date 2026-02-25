package activitypub

import (
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
)

const (
	AEPYNamespaceURL = "http://joinaepyornis.orh/ns#"

	AEPYWorkoutFitFile          = "workoutFitFile"
	AEPYWorkoutLocation         = "workoutLocation"
	AEPYWorkoutSport            = "workoutSport"
	AEPYWorkoutDuration         = "workoutDuration"
	AEPYWorkoutPauseDuration    = "workoutPauseDuration"
	AEPYWorkoutDistance         = "workoutDistance"
	AEPYWorkoutDistance2D       = "workoutDistance2D"
	AEPYWorkoutElevationGain    = "workoutElevationGain"
	AEPYWorkoutElevationLoss    = "workoutElevationLoss"
	AEPYWorkoutAverageSpeed     = "workoutAverageSpeed"
	AEPYWorkoutAverageSpeedMove = "workoutAverageSpeedMoving"
	AEPYWorkoutMaxSpeed         = "workoutMaxSpeed"
	AEPYWorkoutAverageCadence   = "workoutAverageCadence"
	AEPYWorkoutMaxCadence       = "workoutMaxCadence"
	AEPYWorkoutAverageHeartRate = "workoutAverageHeartRate"
	AEPYWorkoutMaxHeartRate     = "workoutMaxHeartRate"
	AEPYWorkoutAveragePower     = "workoutAveragePower"
	AEPYWorkoutMaxPower         = "workoutMaxPower"
	AEPYWorkoutRepetitions      = "workoutRepetitions"
	AEPYWorkoutWeight           = "workoutWeight"

	aepyPrefix = "aepy"
)

var workoutExtensionTerms = []string{
	AEPYWorkoutFitFile,
	AEPYWorkoutLocation,
	AEPYWorkoutSport,
	AEPYWorkoutDuration,
	AEPYWorkoutPauseDuration,
	AEPYWorkoutDistance,
	AEPYWorkoutDistance2D,
	AEPYWorkoutElevationGain,
	AEPYWorkoutElevationLoss,
	AEPYWorkoutAverageSpeed,
	AEPYWorkoutAverageSpeedMove,
	AEPYWorkoutMaxSpeed,
	AEPYWorkoutAverageCadence,
	AEPYWorkoutMaxCadence,
	AEPYWorkoutAverageHeartRate,
	AEPYWorkoutMaxHeartRate,
	AEPYWorkoutAveragePower,
	AEPYWorkoutMaxPower,
	AEPYWorkoutRepetitions,
	AEPYWorkoutWeight,
}

func WorkoutJSONLDContext() jsonld.Context {
	ctx := jsonld.Context{
		{Term: jsonld.NilTerm, IRI: jsonld.IRI(vocab.ActivityBaseURI)},
		{Term: jsonld.Term(aepyPrefix), IRI: jsonld.IRI(AEPYNamespaceURL)},
	}

	for _, term := range workoutExtensionTerms {
		ctx = append(ctx, jsonld.ContextElement{Term: jsonld.Term(term), IRI: jsonld.IRI(aepyPrefix + ":" + term)})
	}

	return ctx
}
