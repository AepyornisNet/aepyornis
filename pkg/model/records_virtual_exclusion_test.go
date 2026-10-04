package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExcludeFromRecords(t *testing.T) {
	testCases := []struct {
		name        string
		workoutType WorkoutType
		subType     string
		expected    bool
	}{
		{"virtual_activity type", WorkoutTypeVirtualActivity, "", true},
		{"indoor_hand_cycling type", WorkoutTypeIndoorHandCycling, "", true},
		{"cycling road", WorkoutTypeCycling, "road", false},
		{"cycling empty", WorkoutTypeCycling, "", false},
		{"cycling spin", WorkoutTypeCycling, "spin", true},
		{"cycling indoor", WorkoutTypeCycling, "indoor_cycling", true},
		{"cycling virtual_ride", WorkoutTypeCycling, "virtual_ride", true},
		{"cycling case-insensitive virtual", WorkoutTypeCycling, "Zwift Virtual Cycling", true},
		{"running outdoor", WorkoutTypeRunning, "", false},
		{"running trail", WorkoutTypeRunning, "trail", false},
		{"running treadmill", WorkoutTypeRunning, "treadmill", true},
		{"running indoor", WorkoutTypeRunning, "indoor_running", true},
		{"running virtual_run", WorkoutTypeRunning, "virtual_run", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := &Workout{
				Type:    tc.workoutType,
				SubType: tc.subType,
			}
			assert.Equal(t, tc.expected, w.ExcludeFromRecords())
		})
	}
}

func TestVirtualWorkout_RecordsGeneration(t *testing.T) {
	db := TestDB(t)

	profile := &Profile{Username: "virtualuser", DisplayName: "Virtual User"}
	require.NoError(t, db.Create(profile).Error)

	now := time.Now()
	makeRecords := func(powerVal float64) []WorkoutRecord {
		records := make([]WorkoutRecord, 120)
		for i := 0; i < 120; i++ {
			records[i] = WorkoutRecord{
				Time:         now.Add(time.Duration(i) * time.Second),
				Duration:     time.Second,
				Distance:     10.0,
				ExtraMetrics: ExtraMetrics{"power": powerVal},
			}
		}
		return records
	}

	t.Run("virtual workout generates power records but no speed records", func(t *testing.T) {
		virtualWorkout := &Workout{
			ProfileID:     profile.ID,
			Name:          "Virtual Zwift Ride",
			Type:          WorkoutTypeCycling,
			SubType:       "virtual_ride",
			Date:          now.Add(1 * time.Hour),
			TotalDuration: 120,
			TotalDistance: 1200,
			Records:       makeRecords(350),
		}
		require.NoError(t, virtualWorkout.Create(db))
		require.NoError(t, virtualWorkout.UpdateRecords(db))

		var speedRecs []WorkoutIntervalBest
		require.NoError(t, db.Where("workout_id = ? AND type = ?", virtualWorkout.ID, WorkoutIntervalBestTypeSpeed).Find(&speedRecs).Error)
		assert.Empty(t, speedRecs, "Virtual workout must not have speed records")

		var powerRecs []WorkoutIntervalBest
		require.NoError(t, db.Where("workout_id = ? AND type = ?", virtualWorkout.ID, WorkoutIntervalBestTypePower).Find(&powerRecs).Error)
		assert.NotEmpty(t, powerRecs, "Virtual workout must have power records")
	})

	t.Run("outdoor workout generates both speed and power records", func(t *testing.T) {
		outdoorWorkout := &Workout{
			ProfileID:     profile.ID,
			Name:          "Outdoor Ride",
			Type:          WorkoutTypeCycling,
			SubType:       "road",
			Date:          now.Add(2 * time.Hour),
			TotalDuration: 1000,
			TotalDistance: 10000,
			Data:          &WorkoutGeoMeta{Center: MapCenter{Lat: 50.0, Lng: 8.0}},
			Records: func() []WorkoutRecord {
				recs := make([]WorkoutRecord, 1000)
				for i := 0; i < 1000; i++ {
					recs[i] = WorkoutRecord{
						Time:         now.Add(time.Duration(i) * time.Second),
						Duration:     time.Second,
						Distance:     10.0,
						ExtraMetrics: ExtraMetrics{"power": 250.0},
					}
				}
				return recs
			}(),
		}
		require.NoError(t, outdoorWorkout.Create(db))
		require.NoError(t, outdoorWorkout.UpdateRecords(db))

		var speedRecs []WorkoutIntervalBest
		require.NoError(t, db.Where("workout_id = ? AND type = ?", outdoorWorkout.ID, WorkoutIntervalBestTypeSpeed).Find(&speedRecs).Error)
		assert.NotEmpty(t, speedRecs, "Outdoor workout must have speed records")

		var powerRecs []WorkoutIntervalBest
		require.NoError(t, db.Where("workout_id = ? AND type = ?", outdoorWorkout.ID, WorkoutIntervalBestTypePower).Find(&powerRecs).Error)
		assert.NotEmpty(t, powerRecs, "Outdoor workout must have power records")
	})

	t.Run("indoor workout without location data calculates power records and breakdown", func(t *testing.T) {
		indoorWorkout := &Workout{
			ProfileID:     profile.ID,
			Name:          "Indoor Trainer (no GPS)",
			Type:          WorkoutTypeCycling,
			SubType:       "indoor_cycling",
			Date:          now.Add(3 * time.Hour),
			TotalDuration: 120,
			TotalDistance: 1200,
			Data:          nil, // No GPS data!
			Records:       makeRecords(300),
			Laps: []WorkoutLap{
				{SortOrder: 0, Start: now.Add(3 * time.Hour), Stop: now.Add(3*time.Hour + 60*time.Second), TotalDuration: 60 * time.Second, TotalDistance: 600},
				{SortOrder: 1, Start: now.Add(3*time.Hour + 60*time.Second), Stop: now.Add(3*time.Hour + 120*time.Second), TotalDuration: 60 * time.Second, TotalDistance: 600},
			},
		}
		require.NoError(t, indoorWorkout.Create(db))
		require.NoError(t, indoorWorkout.UpdateRecords(db))

		var powerRecs []WorkoutIntervalBest
		require.NoError(t, db.Where("workout_id = ? AND type = ?", indoorWorkout.ID, WorkoutIntervalBestTypePower).Find(&powerRecs).Error)
		assert.NotEmpty(t, powerRecs, "Indoor workout without GPS must calculate power records")

		// Verify statistics/breakdown calculation works without Data
		breakdown := indoorWorkout.statisticsWithUnit(500, "m")
		assert.NotEmpty(t, breakdown, "Breakdown items must not be empty even when Data is nil")
	})
}

func TestVirtualWorkout_ExclusionFromRecordsAndRankings(t *testing.T) {
	db := TestDB(t)

	user := &User{
		db: db,
		Profile: Profile{
			Username:    "rankinguser",
			DisplayName: "Ranking User",
		},
	}
	require.NoError(t, db.Create(&user.Profile).Error)
	user.ID = user.Profile.ID

	now := time.Now()
	makeRecords := func(distPerSec, powerVal float64, count int) []WorkoutRecord {
		records := make([]WorkoutRecord, count)
		for i := 0; i < count; i++ {
			records[i] = WorkoutRecord{
				Time:         now.Add(time.Duration(i) * time.Second),
				Duration:     time.Second,
				Distance:     distPerSec,
				ExtraMetrics: ExtraMetrics{"power": powerVal},
			}
		}
		return records
	}

	// Workout 1: Outdoor ride, moderate speed (10 m/s = 36 km/h), power 200W, duration 3600s
	outdoorStats := &WorkoutStats{
		MaxSpeed:            15.0,
		AverageSpeed:        10.0,
		AverageSpeedNoPause: 10.0,
		TotalUp:             200,
	}

	outdoor := &Workout{
		ProfileID:     user.Profile.ID,
		Name:          "Outdoor Ride",
		Type:          WorkoutTypeCycling,
		SubType:       "road",
		Date:          now.Add(1 * time.Hour),
		TotalDuration: 3600 * time.Second,
		TotalDistance: 36000,
		Stats:         outdoorStats,
		Data:          &WorkoutGeoMeta{Center: MapCenter{Lat: 50.0, Lng: 8.0}},
		Records:       makeRecords(10.0, 200, 3600),
	}
	require.NoError(t, outdoor.Create(db))
	require.NoError(t, outdoor.UpdateRecords(db))

	// Workout 2: Virtual ride, absurdly high speed (25 m/s = 90 km/h), high power 400W, long duration 7200s
	virtualStats := &WorkoutStats{
		MaxSpeed:            30.0,
		AverageSpeed:        25.0,
		AverageSpeedNoPause: 25.0,
		TotalUp:             1000,
	}

	virtual := &Workout{
		ProfileID:     user.Profile.ID,
		Name:          "Virtual Ride",
		Type:          WorkoutTypeCycling,
		SubType:       "virtual_ride",
		Date:          now.Add(2 * time.Hour),
		TotalDuration: 7200 * time.Second,
		TotalDistance: 180000,
		Stats:         virtualStats,
		Records:       makeRecords(25.0, 400, 7200),
	}
	require.NoError(t, virtual.Create(db))
	require.NoError(t, virtual.UpdateRecords(db))

	t.Run("GetRecords excludes virtual workout from speed and distance but includes duration", func(t *testing.T) {
		records, err := user.GetRecords(WorkoutTypeCycling, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, records)

		// MaxSpeed should come from outdoor ride (15.0), NOT virtual (30.0)
		assert.Equal(t, outdoor.ID, records.MaxSpeed.ID)
		assert.Equal(t, 15.0, records.MaxSpeed.Value)

		// Distance should come from outdoor ride (36000), NOT virtual (180000)
		assert.Equal(t, outdoor.ID, records.Distance.ID)
		assert.Equal(t, 36000.0, records.Distance.Value)

		// TotalUp should come from outdoor ride (200), NOT virtual (1000)
		assert.Equal(t, outdoor.ID, records.TotalUp.ID)
		assert.Equal(t, 200.0, records.TotalUp.Value)

		// Duration IS allowed for virtual workouts (7200s)
		assert.Equal(t, virtual.ID, records.Duration.ID)
		assert.Equal(t, 7200*time.Second, records.Duration.Value)
	})

	t.Run("getStoredPowerRecords includes virtual workout power records", func(t *testing.T) {
		powerRecs, err := user.getStoredPowerRecords(WorkoutTypeCycling, nil, nil)
		require.NoError(t, err)
		require.NotEmpty(t, powerRecs)

		// Peak power should be from virtual workout (400W vs 200W)
		for _, pr := range powerRecs {
			if pr.Label == "1 h" || pr.Label == "20 m" {
				assert.Equal(t, virtual.ID, pr.WorkoutID)
				assert.Equal(t, 400.0, pr.AveragePower)
			}
		}
	})

	t.Run("GetDistanceRecordRanking excludes virtual workouts", func(t *testing.T) {
		rankings, count, err := user.GetDistanceRecordRanking(WorkoutTypeCycling, "10 km", nil, nil, 10, 0)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
		require.Len(t, rankings, 1)
		assert.Equal(t, outdoor.ID, rankings[0].WorkoutID)
	})
}
