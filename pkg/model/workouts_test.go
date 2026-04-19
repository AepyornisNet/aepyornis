package model

import (
	"testing"

	"github.com/AepyornisNet/aepyornis/pkg/geocoder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() { //nolint:gochecknoinits
	goOffline()
}

func goOffline() {
	geocoder.ForceOffline()
}

func defaultWorkout(t *testing.T) *Workout {
	t.Helper()

	u := defaultUser()
	f1, err := gpxFS.ReadFile("sample1.gpx")
	require.NoError(t, err)

	w, err := NewWorkout(
		&u.Profile,
		WorkoutTypeAutoDetect,
		"some notes",
		"file.gpx",
		f1,
	)

	require.NoError(t, err)
	assert.Len(t, w, 1)

	return w[0]
}

func TestWorkout_ParseWithType(t *testing.T) {
	u := defaultUser()
	f1, err := gpxFS.ReadFile("sample1.gpx")
	require.NoError(t, err)

	w, err := NewWorkout(
		&u.Profile,
		WorkoutTypeWalking,
		"some notes",
		"file.gpx",
		f1,
	)

	require.NoError(t, err)

	assert.NotNil(t, w)
	assert.Len(t, w, 1)
	assert.Equal(t, WorkoutTypeWalking, w[0].Type)
}

func TestWorkout_Parse(t *testing.T) {
	w := defaultWorkout(t)

	assert.NotNil(t, w)
	assert.Equal(t, WorkoutTypeRunning, w.Type)
	assert.Equal(t, "Garmin Connect", w.Creator)
	assert.Equal(t, "some notes", w.Notes)
	assert.InDelta(t, 39, w.Data.Center.Lat, 1)
	assert.InDelta(t, -77, w.Data.Center.Lng, 1)

	assert.Len(t, w.Records, 206)
	assert.InDelta(t, 3125, w.TotalDistance, 1)
	assert.InDelta(t, 3096, w.TotalDistance2D, 1)
	if assert.NotNil(t, w.Stats) {
		assert.InDelta(t, 3.297, w.Stats.AverageSpeed, 0.01)
		assert.InDelta(t, 3.297, w.Stats.AverageSpeedNoPause, 0.01)
	}
	assert.Equal(t, "Some name", w.Name)
	assert.Nil(t, w.Data.Address)
}

func TestWorkout_UpdateData(t *testing.T) {
	db := createMemoryDB(t)
	w := defaultWorkout(t)

	require.NoError(t, w.Save(db))

	ud := w.UpdatedAt
	d := w.Data
	drs := append([]WorkoutRecord(nil), w.Records...)
	assert.NotZero(t, d.ID)
	assert.NotZero(t, w.Data.ID)

	w.CalculateSlopes()

	w.setData(&Workout{Data: dummyMapData()})
	require.NoError(t, w.Save(db))

	assert.NotEqual(t, d, w.Data)
	assert.NotEqual(t, ud, w.UpdatedAt)
	ud = w.UpdatedAt

	require.NoError(t, w.UpdateData(db))
	assert.Len(t, w.Records, len(drs))
	if assert.NotEmpty(t, w.Records) {
		assert.Equal(t, drs[0].Time, w.Records[0].Time)
		assert.Equal(t, drs[len(drs)-1].Time, w.Records[len(w.Records)-1].Time)
		assert.InDelta(t, drs[len(drs)-1].TotalDistance, w.Records[len(w.Records)-1].TotalDistance, 0.01)
	}
	assert.NotEqual(t, ud, w.UpdatedAt)
}

func TestWorkout_SaveAndGet(t *testing.T) {
	db := createMemoryDB(t)
	w := defaultWorkout(t)

	assert.Zero(t, w.UpdatedAt)
	require.NoError(t, w.Save(db))
	assert.NotZero(t, w.UpdatedAt)

	newW, err := GetWorkoutDetails(db, w.ID)
	require.NoError(t, err)
	assert.Equal(t, w.ID, newW.ID)
	assert.Equal(t, w.Records, newW.Records)
}

func TestWorkout_Recreate(t *testing.T) {
	db := createMemoryDB(t)
	w := defaultWorkout(t)

	assert.Zero(t, w.UpdatedAt)
	require.NoError(t, w.Save(db))
	assert.NotZero(t, w.UpdatedAt)

	require.NoError(t, w.Delete(db))

	ws, err := GetWorkouts(db)
	require.NoError(t, err)
	assert.Empty(t, ws)

	require.NoError(t, w.Save(db))

	ws, err = GetWorkouts(db)
	require.NoError(t, err)
	assert.Len(t, ws, 1)
}
