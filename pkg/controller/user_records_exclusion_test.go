package controller

import (
	"fmt"
	"testing"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserController_VisibleRecordsExclusion(t *testing.T) {
	db := createTestDB(t)

	uc := &userController{db: db}

	user := &model.User{
		UserData: model.UserData{
			Active:                   true,
			DefaultWorkoutVisibility: model.WorkoutVisibilityPublic,
		},
		UserSecrets: model.UserSecrets{
			Email:    fmt.Sprintf("recordsuser_%d@example.com", time.Now().UnixNano()),
			Password: "pass",
		},
		Profile: model.Profile{
			Username:    fmt.Sprintf("recordsuser_%d", time.Now().UnixNano()),
			DisplayName: "Records User",
		},
	}
	user.SetDB(db)
	require.NoError(t, user.Create(db))

	now := time.Now()
	makeRecords := func(distPerSec, powerVal float64, count int) []model.WorkoutRecord {
		records := make([]model.WorkoutRecord, count)
		for i := 0; i < count; i++ {
			records[i] = model.WorkoutRecord{
				Time:         now.Add(time.Duration(i) * time.Second),
				Duration:     time.Second,
				Distance:     distPerSec,
				ExtraMetrics: model.ExtraMetrics{"power": powerVal},
			}
		}
		return records
	}

	outdoorStats := &model.WorkoutStats{
		MaxSpeed:            15.0,
		AverageSpeed:        10.0,
		AverageSpeedNoPause: 10.0,
		TotalUp:             200,
	}

	outdoor := &model.Workout{
		ProfileID:     user.Profile.ID,
		Name:          "Outdoor Ride",
		Type:          model.WorkoutTypeCycling,
		SubType:       "road",
		Visibility:    model.WorkoutVisibilityPublic,
		Date:          now.Add(1 * time.Hour),
		TotalDuration: 3600 * time.Second,
		TotalDistance: 36000,
		Stats:         outdoorStats,
		Data:          &model.WorkoutGeoMeta{Center: model.MapCenter{Lat: 50.0, Lng: 8.0}},
		Records:       makeRecords(10.0, 200, 3600),
	}
	require.NoError(t, outdoor.Create(db))
	require.NoError(t, outdoor.UpdateRecords(db))

	virtualStats := &model.WorkoutStats{
		MaxSpeed:            30.0,
		AverageSpeed:        25.0,
		AverageSpeedNoPause: 25.0,
		TotalUp:             1000,
	}

	virtual := &model.Workout{
		ProfileID:     user.Profile.ID,
		Name:          "Virtual Ride",
		Type:          model.WorkoutTypeCycling,
		SubType:       "virtual_ride",
		Visibility:    model.WorkoutVisibilityPublic,
		Date:          now.Add(2 * time.Hour),
		TotalDuration: 7200 * time.Second,
		TotalDistance: 180000,
		Stats:         virtualStats,
		Records:       makeRecords(25.0, 400, 7200),
	}
	require.NoError(t, virtual.Create(db))
	require.NoError(t, virtual.UpdateRecords(db))

	viewerProfileID := uint64(0) // anonymous viewer

	t.Run("getVisibleRecordForType excludes virtual workout from speed and distance but includes duration", func(t *testing.T) {
		record, err := uc.getVisibleRecordForType(user, viewerProfileID, model.WorkoutTypeCycling, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, record)

		assert.Equal(t, outdoor.ID, record.MaxSpeed.ID)
		assert.Equal(t, 15.0, record.MaxSpeed.Value)

		assert.Equal(t, outdoor.ID, record.Distance.ID)
		assert.Equal(t, 36000.0, record.Distance.Value)

		assert.Equal(t, outdoor.ID, record.TotalUp.ID)
		assert.Equal(t, 200.0, record.TotalUp.Value)

		assert.Equal(t, virtual.ID, record.Duration.ID)
		assert.Equal(t, 7200*time.Second, record.Duration.Value)
	})

	t.Run("getVisibleStoredDistanceRecords excludes virtual workouts", func(t *testing.T) {
		distRecs, err := uc.getVisibleStoredDistanceRecords(user, viewerProfileID, model.WorkoutTypeCycling, nil, nil)
		require.NoError(t, err)
		require.NotEmpty(t, distRecs)

		for _, rec := range distRecs {
			assert.Equal(t, outdoor.ID, rec.WorkoutID, "Speed interval record must belong to outdoor workout")
		}
	})

	t.Run("getVisibleStoredPowerRecords includes virtual workouts", func(t *testing.T) {
		powerRecs, err := uc.getVisibleStoredPowerRecords(user, viewerProfileID, model.WorkoutTypeCycling, nil, nil)
		require.NoError(t, err)
		require.NotEmpty(t, powerRecs)

		foundVirtual := false
		for _, rec := range powerRecs {
			if rec.WorkoutID == virtual.ID {
				foundVirtual = true
				break
			}
		}
		assert.True(t, foundVirtual, "Virtual workout power records must be visible")
	})

	t.Run("getVisibleDistanceRanking excludes virtual workouts", func(t *testing.T) {
		rankings, count, err := uc.getVisibleDistanceRanking(user, viewerProfileID, model.WorkoutTypeCycling, "10 km", nil, nil, 10, 0)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
		require.Len(t, rankings, 1)
		assert.Equal(t, outdoor.ID, rankings[0].WorkoutID)
	})
}
