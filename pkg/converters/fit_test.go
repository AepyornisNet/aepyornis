package converters

import (
	"math"
	"testing"
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/muktihari/fit/kit/datetime"
	"github.com/muktihari/fit/profile/filedef"
	"github.com/muktihari/fit/profile/mesgdef"
	"github.com/stretchr/testify/assert"
)

// TestFitTimeIsValid ensures the FIT epoch (decoded from uint32(0)) is
// rejected as an invalid timestamp, fixing the regression where workout
// titles were set to "cycling - 1989-12-31 01:00:00".
func TestFitTimeIsValid(t *testing.T) {
	fitEpoch := datetime.Epoch()

	assert.False(t, fitTimeIsValid(time.Time{}), "Go zero time must be invalid")
	assert.False(t, fitTimeIsValid(fitEpoch), "FIT epoch must be invalid")
	assert.False(t, fitTimeIsValid(fitEpoch.In(time.FixedZone("UTC+1", 3600))), "FIT epoch in local TZ must be invalid")
	assert.True(t, fitTimeIsValid(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)), "real timestamp must be valid")
}

func TestFormatFitWorkoutName_EpochNotUsed(t *testing.T) {
	// When the only available timestamp is the FIT epoch, the name must
	// not include a date component.
	name := formatFitWorkoutName("cycling", datetime.Epoch())
	assert.Equal(t, "cycling", name, "FIT epoch must not appear in workout name")

	name = formatFitWorkoutName("cycling", time.Time{})
	assert.Equal(t, "cycling", name, "Go zero time must not appear in workout name")

	realTime := time.Date(2024, 6, 15, 9, 30, 0, 0, time.UTC)
	name = formatFitWorkoutName("cycling", realTime)
	assert.Equal(t, "cycling - 2024-06-15 09:30:00", name, "real timestamp must appear in workout name")
}

func TestFitActivityStartTime_SkipsEpochLocalTimestamp(t *testing.T) {
	realStart := time.Date(2024, 6, 15, 9, 30, 0, 0, time.UTC)

	// Simulate a FIT file where LocalTimestamp decoded to the FIT epoch
	// (uint32 value 0) but sessions have the correct start time.
	activity := mesgdef.NewActivity(nil)
	activity.LocalTimestamp = datetime.Epoch()

	session := mesgdef.NewSession(nil)
	session.StartTime = realStart

	act := &filedef.Activity{
		Activity: activity,
		Sessions: []*mesgdef.Session{session},
	}

	got := fitActivityStartTime(act)
	assert.Equal(t, realStart.Local(), got, "should fall through to session start time when LocalTimestamp is FIT epoch")
}

func TestDeriveFitSessionDurations_UsesSessionValuesWhenValid(t *testing.T) {
	elapsed, moving, pause := deriveFitSessionDurations(
		3600,
		3600,
		3000,
		3000,
		nil,
		nil,
	)

	assert.Equal(t, time.Hour, elapsed)
	assert.Equal(t, 50*time.Minute, moving)
	assert.Equal(t, 10*time.Minute, pause)
}

func TestDeriveFitSessionDurations_FallsBackToLapsWhenSessionMissing(t *testing.T) {
	laps := []model.WorkoutLap{
		{TotalDuration: 10 * time.Minute, PauseDuration: 2 * time.Minute},
		{TotalDuration: 20 * time.Minute, PauseDuration: 5 * time.Minute},
	}

	elapsed, moving, pause := deriveFitSessionDurations(
		math.MaxUint32,
		0,
		math.MaxUint32,
		0,
		laps,
		nil,
	)

	assert.Equal(t, 30*time.Minute, elapsed)
	assert.Equal(t, 23*time.Minute, moving)
	assert.Equal(t, 7*time.Minute, pause)
}

func TestDeriveFitSessionDurations_FallsBackToRecordsWhenNoSessionOrLaps(t *testing.T) {
	records := []model.WorkoutRecord{
		{Duration: 0, TotalDuration: 0, Distance: 0, TotalDistance: 0},
		{Duration: 60 * time.Second, TotalDuration: 60 * time.Second, Distance: 120, TotalDistance: 120},
		{Duration: 60 * time.Second, TotalDuration: 120 * time.Second, Distance: 0, TotalDistance: 120},
	}

	elapsed, moving, pause := deriveFitSessionDurations(
		math.MaxUint32,
		0,
		math.MaxUint32,
		0,
		nil,
		records,
	)

	assert.Equal(t, 120*time.Second, elapsed)
	assert.Equal(t, 60*time.Second, moving)
	assert.Equal(t, 60*time.Second, pause)
}
