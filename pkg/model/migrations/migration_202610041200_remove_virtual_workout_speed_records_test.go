package migrations

import (
	"testing"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration_RemoveVirtualWorkoutSpeedRecords(t *testing.T) {
	db := model.TestDB(t)

	profile := &model.Profile{Username: "testrecords", DisplayName: "Test Records"}
	require.NoError(t, db.Create(profile).Error)

	now := time.Now()

	outdoor := &model.Workout{
		ProfileID: profile.ID,
		Name:      "Outdoor Ride",
		Type:      model.WorkoutTypeCycling,
		SubType:   "road",
		Date:      now,
	}
	require.NoError(t, db.Create(outdoor).Error)

	virtual := &model.Workout{
		ProfileID: profile.ID,
		Name:      "Virtual Ride",
		Type:      model.WorkoutTypeCycling,
		SubType:   "virtual_ride",
		Date:      now.Add(1 * time.Hour),
	}
	require.NoError(t, db.Create(virtual).Error)

	indoorSpin := &model.Workout{
		ProfileID: profile.ID,
		Name:      "Spin Class",
		Type:      model.WorkoutTypeCycling,
		SubType:   "spin",
		Date:      now.Add(2 * time.Hour),
	}
	require.NoError(t, db.Create(indoorSpin).Error)

	// Create speed and power records for all
	records := []*model.WorkoutIntervalBest{
		{
			WorkoutID:       outdoor.ID,
			Type:            model.WorkoutIntervalBestTypeSpeed,
			Label:           "10km",
			Distance:        10000,
			TargetDistance:  10000,
			DurationSeconds: 1200,
			Average:         8.33,
		},
		{
			WorkoutID:       outdoor.ID,
			Type:            model.WorkoutIntervalBestTypePower,
			Label:           "20m",
			Distance:        10000,
			TargetDistance:  1200,
			DurationSeconds: 1200,
			Average:         250,
		},
		{
			WorkoutID:       virtual.ID,
			Type:            model.WorkoutIntervalBestTypeSpeed,
			Label:           "10km",
			Distance:        10000,
			TargetDistance:  10000,
			DurationSeconds: 900,
			Average:         11.11,
		},
		{
			WorkoutID:       virtual.ID,
			Type:            model.WorkoutIntervalBestTypePower,
			Label:           "20m",
			Distance:        10000,
			TargetDistance:  1200,
			DurationSeconds: 1200,
			Average:         300,
		},
		{
			WorkoutID:       indoorSpin.ID,
			Type:            model.WorkoutIntervalBestTypeSpeed,
			Label:           "10km",
			Distance:        10000,
			TargetDistance:  10000,
			DurationSeconds: 1000,
			Average:         10.0,
		},
	}
	require.NoError(t, db.Create(&records).Error)

	// Run migration
	require.NoError(t, postRemoveVirtualWorkoutSpeedRecordsMigrate(db))

	// Verify remaining records
	var remaining []model.WorkoutIntervalBest
	require.NoError(t, db.Find(&remaining).Error)

	// Outdoor speed record remains
	var outdoorSpeed []model.WorkoutIntervalBest
	require.NoError(t, db.Where("workout_id = ? AND type = ?", outdoor.ID, model.WorkoutIntervalBestTypeSpeed).Find(&outdoorSpeed).Error)
	assert.Len(t, outdoorSpeed, 1)

	// Outdoor power record remains
	var outdoorPower []model.WorkoutIntervalBest
	require.NoError(t, db.Where("workout_id = ? AND type = ?", outdoor.ID, model.WorkoutIntervalBestTypePower).Find(&outdoorPower).Error)
	assert.Len(t, outdoorPower, 1)

	// Virtual speed record deleted
	var virtualSpeed []model.WorkoutIntervalBest
	require.NoError(t, db.Where("workout_id = ? AND type = ?", virtual.ID, model.WorkoutIntervalBestTypeSpeed).Find(&virtualSpeed).Error)
	assert.Len(t, virtualSpeed, 0)

	// Virtual power record remains!
	var virtualPower []model.WorkoutIntervalBest
	require.NoError(t, db.Where("workout_id = ? AND type = ?", virtual.ID, model.WorkoutIntervalBestTypePower).Find(&virtualPower).Error)
	assert.Len(t, virtualPower, 1)

	// Indoor spin speed record deleted
	var spinSpeed []model.WorkoutIntervalBest
	require.NoError(t, db.Where("workout_id = ? AND type = ?", indoorSpin.ID, model.WorkoutIntervalBestTypeSpeed).Find(&spinSpeed).Error)
	assert.Len(t, spinSpeed, 0)
}
