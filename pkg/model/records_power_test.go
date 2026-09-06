package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBestPowerIntervalsForWorkout(t *testing.T) {
	t.Run("nil or empty workout returns nil", func(t *testing.T) {
		assert.Nil(t, bestPowerIntervalsForWorkout(nil, nil))
		assert.Nil(t, bestPowerIntervalsForWorkout(&Workout{}, nil))
	})

	t.Run("workout without power metrics returns nil", func(t *testing.T) {
		w := &Workout{
			Records: []WorkoutRecord{
				{Distance: 10, Duration: time.Second},
				{Distance: 10, Duration: time.Second},
			},
		}
		targets := []PowerRecordTarget{
			{Label: "5 s", TargetDuration: 5},
		}
		assert.Nil(t, bestPowerIntervalsForWorkout(w, targets))
	})

	t.Run("calculates peak power intervals accurately", func(t *testing.T) {
		// 60 seconds total, 1-second samples
		// 0..19: 100W
		// 20..24: 500W (5 seconds peak of 500W)
		// 25..34: 400W (overall 20..34 is 15 seconds: 5*500 + 10*400 = 2500 + 4000 = 6500 / 15 = 433.33W)
		// 35..59: 150W
		now := time.Now()
		records := make([]WorkoutRecord, 60)
		for i := 0; i < 60; i++ {
			var power float64
			switch {
			case i >= 20 && i < 25:
				power = 500.0
			case i >= 25 && i < 35:
				power = 400.0
			case i >= 35:
				power = 150.0
			default:
				power = 100.0
			}

			records[i] = WorkoutRecord{
				Time:         now.Add(time.Duration(i) * time.Second),
				Duration:     time.Second,
				Distance:     10.0,
				ExtraMetrics: ExtraMetrics{"power": power},
			}
		}

		w := &Workout{
			Model:   Model{ID: 42},
			Type:    WorkoutTypeCycling,
			Date:    now,
			Records: records,
		}

		targets := []PowerRecordTarget{
			{Label: "5 s", TargetDuration: 5},
			{Label: "15 s", TargetDuration: 15},
			{Label: "30 s", TargetDuration: 30},
			{Label: "1 min", TargetDuration: 60},
			{Label: "5 min", TargetDuration: 300}, // Longer than workout duration (60s) -> should be skipped
		}

		results := bestPowerIntervalsForWorkout(w, targets)
		require.Len(t, results, 4)

		// 5s target
		assert.Equal(t, "5 s", results[0].Label)
		assert.Equal(t, 5.0, results[0].TargetDuration)
		assert.InDelta(t, 500.0, results[0].AveragePower, 0.01)
		assert.Equal(t, 20, results[0].StartIndex)
		assert.Equal(t, 24, results[0].EndIndex)

		// 15s target (indices 20..34)
		assert.Equal(t, "15 s", results[1].Label)
		assert.Equal(t, 15.0, results[1].TargetDuration)
		assert.InDelta(t, 433.33, results[1].AveragePower, 0.01)
		assert.Equal(t, 20, results[1].StartIndex)
		assert.Equal(t, 34, results[1].EndIndex)

		// 30s target
		assert.Equal(t, "30 s", results[2].Label)
		assert.Equal(t, 30.0, results[2].TargetDuration)
		assert.True(t, results[2].AveragePower > 200.0)

		// 1 min target
		assert.Equal(t, "1 min", results[3].Label)
		assert.Equal(t, 60.0, results[3].TargetDuration)
		assert.Equal(t, 0, results[3].StartIndex)
		assert.Equal(t, 59, results[3].EndIndex)
	})
}

func TestGetWorkoutPowerIntervalRecordsWithRank(t *testing.T) {
	db := createMemoryDB(t)

	u := defaultUser()
	require.NoError(t, u.Save(db))

	w1 := &Workout{
		ProfileID: u.Profile.ID,
		Type:      WorkoutTypeCycling,
		Date:      time.Now().Add(-2 * time.Hour),
		Name:      "Ride 1",
	}
	require.NoError(t, w1.Save(db))

	w2 := &Workout{
		ProfileID: u.Profile.ID,
		Type:      WorkoutTypeCycling,
		Date:      time.Now().Add(-1 * time.Hour),
		Name:      "Ride 2",
	}
	require.NoError(t, w2.Save(db))

	// Insert power records for w1 (300W for 5s, 250W for 1 min)
	r1 := []*WorkoutIntervalBest{
		{
			WorkoutID:       w1.ID,
			Label:           "5 s",
			TargetDistance:  5,
			Distance:        50,
			DurationSeconds: 5,
			Average:         300,
			Type:            WorkoutIntervalBestTypePower,
		},
		{
			WorkoutID:       w1.ID,
			Label:           "1 min",
			TargetDistance:  60,
			Distance:        600,
			DurationSeconds: 60,
			Average:         250,
			Type:            WorkoutIntervalBestTypePower,
		},
	}
	require.NoError(t, db.Create(&r1).Error)

	// Insert power records for w2 (350W for 5s -> rank 1; 200W for 1 min -> rank 2)
	r2 := []*WorkoutIntervalBest{
		{
			WorkoutID:       w2.ID,
			Label:           "5 s",
			TargetDistance:  5,
			Distance:        60,
			DurationSeconds: 5,
			Average:         350,
			Type:            WorkoutIntervalBestTypePower,
		},
		{
			WorkoutID:       w2.ID,
			Label:           "1 min",
			TargetDistance:  60,
			Distance:        550,
			DurationSeconds: 60,
			Average:         200,
			Type:            WorkoutIntervalBestTypePower,
		},
	}
	require.NoError(t, db.Create(&r2).Error)

	// Query for w1:
	// "5 s" has 300W (vs 350W on w2) -> rank 2
	// "1 min" has 250W (vs 200W on w2) -> rank 1
	rankedW1, err := GetWorkoutPowerIntervalRecordsWithRank(db, u.Profile.ID, WorkoutTypeCycling, w1.ID)
	require.NoError(t, err)
	require.Len(t, rankedW1, 2)

	assert.Equal(t, "5 s", rankedW1[0].Label)
	assert.Equal(t, int64(2), rankedW1[0].Rank)
	assert.Equal(t, 300.0, rankedW1[0].Average)

	assert.Equal(t, "1 min", rankedW1[1].Label)
	assert.Equal(t, int64(1), rankedW1[1].Rank)
	assert.Equal(t, 250.0, rankedW1[1].Average)

	// Query for w2:
	// "5 s" has 350W -> rank 1
	// "1 min" has 200W -> rank 2
	rankedW2, err := GetWorkoutPowerIntervalRecordsWithRank(db, u.Profile.ID, WorkoutTypeCycling, w2.ID)
	require.NoError(t, err)
	require.Len(t, rankedW2, 2)

	assert.Equal(t, "5 s", rankedW2[0].Label)
	assert.Equal(t, int64(1), rankedW2[0].Rank)
	assert.Equal(t, 350.0, rankedW2[0].Average)

	assert.Equal(t, "1 min", rankedW2[1].Label)
	assert.Equal(t, int64(2), rankedW2[1].Rank)
	assert.Equal(t, 200.0, rankedW2[1].Average)
}

func TestWorkout_UpdateRecords_Power(t *testing.T) {
	db := createMemoryDB(t)

	u := defaultUser()
	require.NoError(t, u.Save(db))

	now := time.Now()
	records := make([]WorkoutRecord, 30)
	for i := 0; i < 30; i++ {
		records[i] = WorkoutRecord{
			Time:         now.Add(time.Duration(i) * time.Second),
			Duration:     time.Second,
			Distance:     10.0,
			ExtraMetrics: ExtraMetrics{"power": 300.0},
		}
	}

	w := &Workout{
		ProfileID: u.Profile.ID,
		Type:      WorkoutTypeCycling,
		Date:      now,
		Data:      &WorkoutGeoMeta{Center: MapCenter{}},
		Records:   records,
	}
	require.NoError(t, w.Save(db))

	err := w.UpdateRecords(db)
	require.NoError(t, err)

	var stored []WorkoutIntervalBest
	require.NoError(t, db.Where("workout_id = ?", w.ID).Find(&stored).Error)

	var powerCount, speedCount int
	for _, s := range stored {
		if s.Type == WorkoutIntervalBestTypePower {
			powerCount++
		} else if s.Type == WorkoutIntervalBestTypeSpeed {
			speedCount++
		}
	}

	// 5s, 15s, 30s targets are <= 30 seconds
	assert.Equal(t, 3, powerCount)
	assert.Equal(t, 0, speedCount) // 300m total distance is less than 5mi cycling distance target
}

func TestUser_GetRecords_Power(t *testing.T) {
	db := createMemoryDB(t)

	u := defaultUser()
	u.db = db
	require.NoError(t, u.Save(db))

	w := &Workout{
		ProfileID: u.Profile.ID,
		Type:      WorkoutTypeCycling,
		Date:      time.Now(),
		Name:      "Ride Power",
	}
	require.NoError(t, w.Save(db))

	r := []*WorkoutIntervalBest{
		{
			WorkoutID:       w.ID,
			Label:           "5 s",
			TargetDistance:  5,
			Distance:        50,
			DurationSeconds: 5,
			Average:         420,
			Type:            WorkoutIntervalBestTypePower,
		},
		{
			WorkoutID:       w.ID,
			Label:           "1 min",
			TargetDistance:  60,
			Distance:        600,
			DurationSeconds: 60,
			Average:         310,
			Type:            WorkoutIntervalBestTypePower,
		},
	}
	require.NoError(t, db.Create(&r).Error)

	records, err := u.GetRecords(WorkoutTypeCycling, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, records)
	assert.True(t, records.Active)
	require.Len(t, records.PowerRecords, 2)

	assert.Equal(t, "5 s", records.PowerRecords[0].Label)
	assert.Equal(t, 420.0, records.PowerRecords[0].AveragePower)
	assert.Equal(t, "1 min", records.PowerRecords[1].Label)
	assert.Equal(t, 310.0, records.PowerRecords[1].AveragePower)
}
