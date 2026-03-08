package activitypub

import (
	"testing"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// NewWorkoutNote
// ---------------------------------------------------------------------------

func TestNewWorkoutNote_IsNote(t *testing.T) {
	note := NewWorkoutNote()
	require.NotNil(t, note)
	assert.Equal(t, vocab.NoteType, note.GetType())
}

// ---------------------------------------------------------------------------
// PopulateFromWorkout
// ---------------------------------------------------------------------------

func TestPopulateFromWorkout_NilWorkout(t *testing.T) {
	note := NewWorkoutNote()
	// Must not panic.
	note.PopulateFromWorkout(nil, "")
}

func TestPopulateFromWorkout_BasicFields(t *testing.T) {
	workout := &model.Workout{
		Name: "Morning Run",
		Type: model.WorkoutTypeRunning,
		Date: time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC),
		Data: &model.MapData{
			WorkoutData: model.WorkoutData{
				TotalDistance: 5000,
				TotalDuration: 30 * time.Minute,
				WorkoutStats:  model.WorkoutStats{AverageSpeed: 3.5},
			},
		},
	}

	note := NewWorkoutNote()
	note.PopulateFromWorkout(workout, "https://example.com/fit/1.fit")

	assert.Equal(t, vocab.IRI("https://example.com/fit/1.fit"), note.WorkoutFitFile)
	assert.Equal(t, "running", note.WorkoutSport)
	assert.EqualValues(t, int64((30 * time.Minute).Seconds()), note.WorkoutDuration)
	assert.InDelta(t, 5000.0, note.WorkoutDistance, 1)
}

func TestPopulateFromWorkout_CustomType(t *testing.T) {
	workout := &model.Workout{
		Name:       "Parkour Session",
		Type:       model.WorkoutTypeOther,
		CustomType: "parkour",
		Data:       &model.MapData{},
	}

	note := NewWorkoutNote()
	note.PopulateFromWorkout(workout, "")

	assert.Equal(t, "parkour", note.WorkoutSport)
}

func TestPopulateFromWorkout_EmptyFitURL(t *testing.T) {
	workout := &model.Workout{
		Name: "Simple Workout",
		Type: model.WorkoutTypeWalking,
		Data: &model.MapData{},
	}

	note := NewWorkoutNote()
	note.PopulateFromWorkout(workout, "")

	assert.True(t, vocab.IsNil(note.WorkoutFitFile) || note.WorkoutFitFile == "")
}

// ---------------------------------------------------------------------------
// WorkoutNoteContent
// ---------------------------------------------------------------------------

func TestWorkoutNoteContent_NilWorkout(t *testing.T) {
	content := WorkoutNoteContent(nil)
	assert.Equal(t, "Workout", content)
}

func TestWorkoutNoteContent_NameOnly(t *testing.T) {
	workout := &model.Workout{
		Name: "Rest Day",
		Data: &model.MapData{},
	}
	content := WorkoutNoteContent(workout)
	assert.Equal(t, "Rest Day", content)
}

func TestWorkoutNoteContent_WithDistance(t *testing.T) {
	workout := &model.Workout{
		Name: "Long Run",
		Data: &model.MapData{
			WorkoutData: model.WorkoutData{
				TotalDistance: 10000,
				TotalDuration: 60 * time.Minute,
				WorkoutStats:  model.WorkoutStats{AverageSpeed: 2.78},
			},
		},
	}
	content := WorkoutNoteContent(workout)
	assert.Contains(t, content, "Long Run")
	assert.Contains(t, content, "distance:")
	assert.Contains(t, content, "duration:")
}

func TestWorkoutNoteContent_WithElevation(t *testing.T) {
	workout := &model.Workout{
		Name: "Hill Climb",
		Data: &model.MapData{
			WorkoutData: model.WorkoutData{
				WorkoutStats: model.WorkoutStats{
					TotalUp: 300,
				},
				TotalDistance: 5000,
				TotalDuration: 45 * time.Minute,
			},
		},
	}
	content := WorkoutNoteContent(workout)
	assert.Contains(t, content, "elevation gain:")
}

func TestWorkoutNoteContent_WithRepetitions(t *testing.T) {
	workout := &model.Workout{
		Name: "Push-ups",
		Data: &model.MapData{
			WorkoutData: model.WorkoutData{
				TotalRepetitions: 50,
			},
		},
	}
	content := WorkoutNoteContent(workout)
	assert.Contains(t, content, "repetitions: 50")
}

func TestWorkoutNoteContent_WithWeight(t *testing.T) {
	workout := &model.Workout{
		Name: "Bench Press",
		Data: &model.MapData{
			WorkoutData: model.WorkoutData{
				TotalWeight: 80,
			},
		},
	}
	content := WorkoutNoteContent(workout)
	assert.Contains(t, content, "weight:")
}

// ---------------------------------------------------------------------------
// WorkoutFITFilename
// ---------------------------------------------------------------------------

func TestWorkoutFITFilename_NilWorkout(t *testing.T) {
	assert.Equal(t, "workout.fit", WorkoutFITFilename(nil))
}

func TestWorkoutFITFilename_WithWorkout(t *testing.T) {
	workout := &model.Workout{}
	workout.ID = 42
	filename := WorkoutFITFilename(workout)
	assert.Equal(t, "workout-42.fit", filename)
}
