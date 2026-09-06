package migrations

import (
	"testing"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration_RouteSegmentMatchesSurrogatePKFix(t *testing.T) {
	db := model.TestDB(t)

	// Verify route_segment_matches has surrogate id column and can store multiple matches for same segment and workout
	require.True(t, db.Migrator().HasTable("route_segment_matches"))
	require.True(t, db.Migrator().HasColumn("route_segment_matches", "id"))

	profile := &model.Profile{Username: "testuser2", DisplayName: "Test User 2"}
	require.NoError(t, db.Create(profile).Error)

	workout := &model.Workout{
		ProfileID: profile.ID,
		Name:      "Test Workout 2",
		Type:      model.WorkoutTypeRunning,
	}
	require.NoError(t, db.Create(workout).Error)

	rs := &model.RouteSegment{
		ProfileID: profile.ID,
		Name:      "Test Segment 2",
		Checksum:  []byte("checksum456"),
	}
	require.NoError(t, db.Create(rs).Error)

	// Insert multiple matches for the same (route_segment_id, workout_id) pair in a single batch
	matches := []*model.RouteSegmentMatch{
		{
			RouteSegmentID: rs.ID,
			WorkoutID:      workout.ID,
			FirstID:        0,
			LastID:         50,
			Distance:       1000.0,
			Duration:       5 * time.Minute,
		},
		{
			RouteSegmentID: rs.ID,
			WorkoutID:      workout.ID,
			FirstID:        60,
			LastID:         110,
			Distance:       1000.0,
			Duration:       4 * time.Minute,
		},
		{
			RouteSegmentID: rs.ID,
			WorkoutID:      workout.ID,
			FirstID:        120,
			LastID:         170,
			Distance:       1000.0,
			Duration:       4 * time.Minute,
		},
	}

	require.NoError(t, db.Create(&matches).Error)
	assert.NotZero(t, matches[0].ID)
	assert.NotZero(t, matches[1].ID)
	assert.NotZero(t, matches[2].ID)
	assert.NotEqual(t, matches[0].ID, matches[1].ID)
	assert.NotEqual(t, matches[1].ID, matches[2].ID)

	var loaded []model.RouteSegmentMatch
	require.NoError(t, db.Where("route_segment_id = ? AND workout_id = ?", rs.ID, workout.ID).Order("id ASC").Find(&loaded).Error)
	assert.Len(t, loaded, 3)
}
