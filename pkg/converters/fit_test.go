package converters

import (
	"bytes"
	"math"
	"os"
	"testing"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/muktihari/fit/encoder"
	"github.com/muktihari/fit/kit/datetime"
	"github.com/muktihari/fit/profile/filedef"
	"github.com/muktihari/fit/profile/mesgdef"
	"github.com/muktihari/fit/profile/typedef"
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

func TestParseFit_Swimming_SynthesizeRecordsFromLengths(t *testing.T) {
	startTime := time.Date(2026, 8, 19, 8, 0, 0, 0, time.UTC)
	poolLength := 25.0

	lengths := []fitSwimLength{
		{
			start:    startTime,
			end:      startTime.Add(50 * time.Second),
			elapsed:  50,
			timer:    50,
			speed:    0.5,
			cadence:  30,
			strokes:  25,
			isActive: true,
			cumDist:  25,
		},
		{
			start:    startTime.Add(50 * time.Second),
			end:      startTime.Add(80 * time.Second),
			elapsed:  30,
			timer:    30,
			speed:    0,
			cadence:  0,
			isActive: false,
			cumDist:  25,
		},
		{
			start:    startTime.Add(80 * time.Second),
			end:      startTime.Add(130 * time.Second),
			elapsed:  50,
			timer:    50,
			speed:    0.5,
			cadence:  30,
			strokes:  25,
			isActive: true,
			cumDist:  50,
		},
	}

	meta, records := synthesizeRecordsFromSwimLengths(lengths, poolLength)
	assert.NotNil(t, meta)
	assert.NotEmpty(t, records)

	assert.Equal(t, 50.0, records[len(records)-1].TotalDistance)

	laps := parseFitSwimLaps(&filedef.Activity{}, lengths, poolLength)
	assert.Len(t, laps, 2)
	assert.Equal(t, 25.0, laps[0].TotalDistance)
	assert.Equal(t, 25.0, laps[1].TotalDistance)
}

func TestParseFit_Swimming_SelfContained(t *testing.T) {
	start := time.Date(2026, 8, 19, 8, 0, 0, 0, time.UTC)
	poolLen := 25.0

	act := filedef.NewActivity()
	act.FileId.
		SetType(typedef.FileActivity).
		SetTimeCreated(start).
		SetManufacturer(typedef.ManufacturerGarmin).
		SetProduct(1).
		SetProductName("Garmin Swim 2")

	// 5 Lengths: 2 active (50m) + 1 rest (30s) + 2 active (50m) = 100m total
	t0 := start
	l0 := mesgdef.NewLength(nil).
		SetStartTime(t0).
		SetTotalElapsedTimeScaled(40).
		SetTotalTimerTimeScaled(40).
		SetTotalStrokes(20).
		SetAvgSpeedScaled(poolLen / 40.0).
		SetSwimStroke(typedef.SwimStrokeBreaststroke).
		SetLengthType(typedef.LengthTypeActive).
		SetAvgSwimmingCadence(30)

	t1 := t0.Add(40 * time.Second)
	l1 := mesgdef.NewLength(nil).
		SetStartTime(t1).
		SetTotalElapsedTimeScaled(45).
		SetTotalTimerTimeScaled(45).
		SetTotalStrokes(22).
		SetAvgSpeedScaled(poolLen / 45.0).
		SetSwimStroke(typedef.SwimStrokeBackstroke).
		SetLengthType(typedef.LengthTypeActive).
		SetAvgSwimmingCadence(29)

	t2 := t1.Add(45 * time.Second)
	l2 := mesgdef.NewLength(nil).
		SetStartTime(t2).
		SetTotalElapsedTimeScaled(30).
		SetTotalTimerTimeScaled(30).
		SetLengthType(typedef.LengthTypeIdle)

	t3 := t2.Add(30 * time.Second)
	l3 := mesgdef.NewLength(nil).
		SetStartTime(t3).
		SetTotalElapsedTimeScaled(42).
		SetTotalTimerTimeScaled(42).
		SetTotalStrokes(18).
		SetAvgSpeedScaled(poolLen / 42.0).
		SetSwimStroke(typedef.SwimStrokeFreestyle).
		SetLengthType(typedef.LengthTypeActive).
		SetAvgSwimmingCadence(26)

	t4 := t3.Add(42 * time.Second)
	l4 := mesgdef.NewLength(nil).
		SetStartTime(t4).
		SetTotalElapsedTimeScaled(43).
		SetTotalTimerTimeScaled(43).
		SetTotalStrokes(19).
		SetAvgSpeedScaled(poolLen / 43.0).
		SetSwimStroke(typedef.SwimStrokeFreestyle).
		SetLengthType(typedef.LengthTypeActive).
		SetAvgSwimmingCadence(26)

	act.Lengths = append(act.Lengths, l0, l1, l2, l3, l4)

	totalElapsed := 200.0 // 40 + 45 + 30 + 42 + 43 = 200s
	totalTimer := 170.0   // 40 + 45 + 42 + 43 = 170s
	end := start.Add(time.Duration(totalElapsed * float64(time.Second)))

	session := mesgdef.NewSession(nil).
		SetStartTime(start).
		SetTimestamp(end).
		SetSport(typedef.SportSwimming).
		SetSubSport(typedef.SubSportLapSwimming).
		SetPoolLengthScaled(poolLen).
		SetPoolLengthUnit(typedef.DisplayMeasureMetric).
		SetTotalDistanceScaled(100).
		SetTotalElapsedTimeScaled(totalElapsed).
		SetTotalTimerTimeScaled(totalTimer).
		SetNumLengths(5).
		SetNumActiveLengths(4).
		SetTotalCycles(79).
		SetAvgCadence(28).
		SetAvgHeartRate(142).
		SetMaxHeartRate(165)

	act.Sessions = append(act.Sessions, session)

	// Generate 1Hz records with HeartRate and NO distance
	for s := 0; s <= int(totalElapsed); s++ {
		ts := start.Add(time.Duration(s) * time.Second)
		rec := mesgdef.NewRecord(nil).
			SetTimestamp(ts).
			SetHeartRate(uint8(130 + s%30))
		act.Records = append(act.Records, rec)
	}

	fitData := act.ToFIT(nil)
	buf := bytes.NewBuffer(nil)
	enc := encoder.New(buf)
	err := enc.Encode(&fitData)
	assert.NoError(t, err)

	workouts, err := ParseFit(buf.Bytes(), "swimming.fit")
	assert.NoError(t, err)
	assert.Len(t, workouts, 1)

	w := workouts[0]
	assert.Equal(t, model.WorkoutTypeSwimming, w.Type)
	assert.Equal(t, 100.0, w.TotalDistance)
	assert.Equal(t, 100.0, w.TotalDistance2D)
	assert.Len(t, w.Records, 201)

	// Distance progression check
	assert.Equal(t, 0.0, w.Records[0].TotalDistance)
	assert.Equal(t, 100.0, w.Records[len(w.Records)-1].TotalDistance)

	// Verify extra metrics
	assert.Contains(t, w.Records[10].ExtraMetrics, "speed")
	assert.Contains(t, w.Records[10].ExtraMetrics, "cadence")
	assert.Contains(t, w.Records[10].ExtraMetrics, "heart-rate")

	// Verify rest pause
	// Length 2 is from t=85s to t=115s
	assert.True(t, w.Records[90].Pause.Valid && w.Records[90].Pause.Bool, "rest period should be marked pause")
	assert.False(t, w.Records[10].Pause.Valid && w.Records[10].Pause.Bool, "active period should not be marked pause")

	// Verify 2 interval laps generated
	assert.Len(t, w.Laps, 2)
	assert.Equal(t, 50.0, w.Laps[0].TotalDistance)
	assert.Equal(t, 50.0, w.Laps[1].TotalDistance)

	// Verify stats
	assert.InDelta(t, 28.0, w.Stats.AverageCadence, 1.0)
	assert.InDelta(t, 142.0, w.Stats.AverageHeartRate, 1.0)
	assert.InDelta(t, 0.50, w.Stats.AverageSpeed, 0.05)
}

func TestParseFit_Swimming_SampleFile(t *testing.T) {
	data, err := os.ReadFile("/home/brihm/Downloads/24032027670_ACTIVITY.fit")
	if err != nil {
		t.Skip("sample swimming FIT file not found")
	}

	workouts, err := ParseFit(data, "24032027670_ACTIVITY.fit")
	assert.NoError(t, err)
	assert.Len(t, workouts, 1)

	w := workouts[0]
	assert.Equal(t, model.WorkoutTypeSwimming, w.Type)
	assert.Equal(t, 750.0, w.TotalDistance)
	assert.Equal(t, 750.0, w.TotalDistance2D)
	assert.NotEmpty(t, w.Records)

	// Verify distance increases from 0 to 750 across records
	assert.Equal(t, 0.0, w.Records[0].TotalDistance)
	assert.Equal(t, 750.0, w.Records[len(w.Records)-1].TotalDistance)

	// Verify records contain speed and cadence extra metrics
	var hasSpeed, hasCadence, hasHeartRate, hasPause bool
	for _, r := range w.Records {
		if _, ok := r.ExtraMetrics["speed"]; ok {
			hasSpeed = true
		}
		if _, ok := r.ExtraMetrics["cadence"]; ok {
			hasCadence = true
		}
		if _, ok := r.ExtraMetrics["heart-rate"]; ok {
			hasHeartRate = true
		}
		if r.Pause.Valid && r.Pause.Bool {
			hasPause = true
		}
	}
	assert.True(t, hasSpeed, "records should contain speed extra metric")
	assert.True(t, hasCadence, "records should contain cadence extra metric")
	assert.True(t, hasHeartRate, "records should contain heart-rate extra metric")
	assert.True(t, hasPause, "rest intervals should be flagged as pause")

	// Verify intervals / laps: 2 sets separated by rest
	assert.Len(t, w.Laps, 2)
	assert.Equal(t, 425.0, w.Laps[0].TotalDistance)
	assert.Equal(t, 325.0, w.Laps[1].TotalDistance)

	// Verify Stats
	assert.InDelta(t, 33.4, w.Stats.AverageCadence, 1.0)
	assert.InDelta(t, 0.40, w.Stats.AverageSpeed, 0.05)
	assert.InDelta(t, 144.0, w.Stats.AverageHeartRate, 1.0)
}
