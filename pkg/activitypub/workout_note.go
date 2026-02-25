package activitypub

import (
	"encoding/json"
	"strings"

	vocab "github.com/go-ap/activitypub"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
)

type WorkoutNote struct {
	vocab.Object
	WorkoutFitFile          vocab.IRI `jsonld:"workoutFitFile,omitempty"`
	WorkoutLocation         string    `jsonld:"workoutLocation,omitempty"`
	WorkoutSport            string    `jsonld:"workoutSport,omitempty"`
	WorkoutDuration         int64     `jsonld:"workoutDuration,omitempty"`
	WorkoutPauseDuration    int64     `jsonld:"workoutPauseDuration,omitempty"`
	WorkoutDistance         float64   `jsonld:"workoutDistance,omitempty"`
	WorkoutDistance2D       float64   `jsonld:"workoutDistance2D,omitempty"`
	WorkoutElevationGain    float64   `jsonld:"workoutElevationGain,omitempty"`
	WorkoutElevationLoss    float64   `jsonld:"workoutElevationLoss,omitempty"`
	WorkoutAverageSpeed     float64   `jsonld:"workoutAverageSpeed,omitempty"`
	WorkoutAverageSpeedMove float64   `jsonld:"workoutAverageSpeedMoving,omitempty"`
	WorkoutMaxSpeed         float64   `jsonld:"workoutMaxSpeed,omitempty"`
	WorkoutAverageCadence   float64   `jsonld:"workoutAverageCadence,omitempty"`
	WorkoutMaxCadence       float64   `jsonld:"workoutMaxCadence,omitempty"`
	WorkoutAverageHeartRate float64   `jsonld:"workoutAverageHeartRate,omitempty"`
	WorkoutMaxHeartRate     float64   `jsonld:"workoutMaxHeartRate,omitempty"`
	WorkoutAveragePower     float64   `jsonld:"workoutAveragePower,omitempty"`
	WorkoutMaxPower         float64   `jsonld:"workoutMaxPower,omitempty"`
	WorkoutRepetitions      int       `jsonld:"workoutRepetitions,omitempty"`
	WorkoutWeight           float64   `jsonld:"workoutWeight,omitempty"`
}

type workoutNoteExtensions struct {
	WorkoutFitFile          vocab.IRI `json:"workoutFitFile,omitempty"`
	WorkoutLocation         string    `json:"workoutLocation,omitempty"`
	WorkoutSport            string    `json:"workoutSport,omitempty"`
	WorkoutDuration         int64     `json:"workoutDuration,omitempty"`
	WorkoutPauseDuration    int64     `json:"workoutPauseDuration,omitempty"`
	WorkoutDistance         float64   `json:"workoutDistance,omitempty"`
	WorkoutDistance2D       float64   `json:"workoutDistance2D,omitempty"`
	WorkoutElevationGain    float64   `json:"workoutElevationGain,omitempty"`
	WorkoutElevationLoss    float64   `json:"workoutElevationLoss,omitempty"`
	WorkoutAverageSpeed     float64   `json:"workoutAverageSpeed,omitempty"`
	WorkoutAverageSpeedMove float64   `json:"workoutAverageSpeedMoving,omitempty"`
	WorkoutMaxSpeed         float64   `json:"workoutMaxSpeed,omitempty"`
	WorkoutAverageCadence   float64   `json:"workoutAverageCadence,omitempty"`
	WorkoutMaxCadence       float64   `json:"workoutMaxCadence,omitempty"`
	WorkoutAverageHeartRate float64   `json:"workoutAverageHeartRate,omitempty"`
	WorkoutMaxHeartRate     float64   `json:"workoutMaxHeartRate,omitempty"`
	WorkoutAveragePower     float64   `json:"workoutAveragePower,omitempty"`
	WorkoutMaxPower         float64   `json:"workoutMaxPower,omitempty"`
	WorkoutRepetitions      int       `json:"workoutRepetitions,omitempty"`
	WorkoutWeight           float64   `json:"workoutWeight,omitempty"`
}

func NewWorkoutNote() *WorkoutNote {
	note := &WorkoutNote{}
	note.Object = *vocab.ObjectNew(vocab.NoteType)
	return note
}

func (n *WorkoutNote) PopulateFromWorkout(workout *model.Workout, fitURL vocab.IRI) {
	if workout == nil {
		return
	}

	n.WorkoutFitFile = fitURL

	location := strings.TrimSpace(workout.FullAddress())
	if location == "" {
		location = strings.TrimSpace(workout.Address())
	}
	n.WorkoutLocation = location

	n.WorkoutSport = workout.Type.String()
	if workout.CustomType != "" {
		n.WorkoutSport = workout.CustomType
	}

	n.WorkoutDuration = int64(workout.TotalDuration().Seconds())
	n.WorkoutPauseDuration = int64(workout.PauseDuration().Seconds())
	n.WorkoutDistance = workout.TotalDistance()
	n.WorkoutDistance2D = workout.TotalDistance2D()
	n.WorkoutElevationGain = workout.TotalUp()
	n.WorkoutElevationLoss = workout.TotalDown()
	n.WorkoutAverageSpeed = workout.AverageSpeed()
	n.WorkoutAverageSpeedMove = workout.AverageSpeedNoPause()
	n.WorkoutMaxSpeed = workout.MaxSpeed()
	n.WorkoutAverageCadence = workout.AverageCadence()
	n.WorkoutMaxCadence = workout.MaxCadence()
	n.WorkoutRepetitions = workout.TotalRepetitions()
	n.WorkoutWeight = workout.Weight()

	if workout.Data != nil {
		n.WorkoutAverageHeartRate = workout.Data.AverageHeartRate
		n.WorkoutMaxHeartRate = workout.Data.MaxHeartRate
		n.WorkoutAveragePower = workout.Data.AveragePower
		n.WorkoutMaxPower = workout.Data.MaxPower
	}
}

func (n WorkoutNote) extensionValues() workoutNoteExtensions {
	return workoutNoteExtensions{
		WorkoutFitFile:          n.WorkoutFitFile,
		WorkoutLocation:         n.WorkoutLocation,
		WorkoutSport:            n.WorkoutSport,
		WorkoutDuration:         n.WorkoutDuration,
		WorkoutPauseDuration:    n.WorkoutPauseDuration,
		WorkoutDistance:         n.WorkoutDistance,
		WorkoutDistance2D:       n.WorkoutDistance2D,
		WorkoutElevationGain:    n.WorkoutElevationGain,
		WorkoutElevationLoss:    n.WorkoutElevationLoss,
		WorkoutAverageSpeed:     n.WorkoutAverageSpeed,
		WorkoutAverageSpeedMove: n.WorkoutAverageSpeedMove,
		WorkoutMaxSpeed:         n.WorkoutMaxSpeed,
		WorkoutAverageCadence:   n.WorkoutAverageCadence,
		WorkoutMaxCadence:       n.WorkoutMaxCadence,
		WorkoutAverageHeartRate: n.WorkoutAverageHeartRate,
		WorkoutMaxHeartRate:     n.WorkoutMaxHeartRate,
		WorkoutAveragePower:     n.WorkoutAveragePower,
		WorkoutMaxPower:         n.WorkoutMaxPower,
		WorkoutRepetitions:      n.WorkoutRepetitions,
		WorkoutWeight:           n.WorkoutWeight,
	}
}

func (n *WorkoutNote) applyExtensionValues(ext workoutNoteExtensions) {
	n.WorkoutFitFile = ext.WorkoutFitFile
	n.WorkoutLocation = ext.WorkoutLocation
	n.WorkoutSport = ext.WorkoutSport
	n.WorkoutDuration = ext.WorkoutDuration
	n.WorkoutPauseDuration = ext.WorkoutPauseDuration
	n.WorkoutDistance = ext.WorkoutDistance
	n.WorkoutDistance2D = ext.WorkoutDistance2D
	n.WorkoutElevationGain = ext.WorkoutElevationGain
	n.WorkoutElevationLoss = ext.WorkoutElevationLoss
	n.WorkoutAverageSpeed = ext.WorkoutAverageSpeed
	n.WorkoutAverageSpeedMove = ext.WorkoutAverageSpeedMove
	n.WorkoutMaxSpeed = ext.WorkoutMaxSpeed
	n.WorkoutAverageCadence = ext.WorkoutAverageCadence
	n.WorkoutMaxCadence = ext.WorkoutMaxCadence
	n.WorkoutAverageHeartRate = ext.WorkoutAverageHeartRate
	n.WorkoutMaxHeartRate = ext.WorkoutMaxHeartRate
	n.WorkoutAveragePower = ext.WorkoutAveragePower
	n.WorkoutMaxPower = ext.WorkoutMaxPower
	n.WorkoutRepetitions = ext.WorkoutRepetitions
	n.WorkoutWeight = ext.WorkoutWeight
}

func (n WorkoutNote) MarshalJSON() ([]byte, error) {
	objJSON, err := n.Object.MarshalJSON()
	if err != nil {
		return nil, err
	}

	payload := map[string]any{}
	if len(objJSON) > 0 {
		if err := json.Unmarshal(objJSON, &payload); err != nil {
			return nil, err
		}
	}

	extJSON, err := json.Marshal(n.extensionValues())
	if err != nil {
		return nil, err
	}

	extPayload := map[string]any{}
	if len(extJSON) > 0 {
		if err := json.Unmarshal(extJSON, &extPayload); err != nil {
			return nil, err
		}
	}

	for key, value := range extPayload {
		payload[key] = value
	}

	return json.Marshal(payload)
}

func (n *WorkoutNote) UnmarshalJSON(data []byte) error {
	if err := n.Object.UnmarshalJSON(data); err != nil {
		return err
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	extPayload := map[string]json.RawMessage{}
	for _, term := range workoutExtensionTerms {
		for _, key := range []string{term, aepyPrefix + ":" + term, AEPYNamespaceURL + term} {
			raw, ok := payload[key]
			if !ok {
				continue
			}

			extPayload[term] = raw
			break
		}
	}

	extJSON, err := json.Marshal(extPayload)
	if err != nil {
		return err
	}

	ext := workoutNoteExtensions{}
	if err := json.Unmarshal(extJSON, &ext); err != nil {
		return err
	}

	n.applyExtensionValues(ext)

	return nil
}
