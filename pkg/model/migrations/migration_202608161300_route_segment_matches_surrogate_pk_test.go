package migrations

import (
	"testing"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration_RouteSegmentMatchesSurrogatePK(t *testing.T) {
	db := model.TestDB(t)

	// Verify route_segment_matches has surrogate id column and can store multiple matches for same segment and workout
	require.True(t, db.Migrator().HasTable("route_segment_matches"))
	require.True(t, db.Migrator().HasColumn("route_segment_matches", "id"))

	profile := &model.Profile{Username: "testuser", DisplayName: "Test User"}
	require.NoError(t, db.Create(profile).Error)

	workout := &model.Workout{
		ProfileID: profile.ID,
		Name:      "Test Workout",
		Type:      model.WorkoutTypeRunning,
	}
	require.NoError(t, db.Create(workout).Error)

	rs := &model.RouteSegment{
		ProfileID: profile.ID,
		Name:      "Test Segment",
		Checksum:  []byte("checksum123"),
	}
	require.NoError(t, db.Create(rs).Error)

	// Insert multiple matches for the same (route_segment_id, workout_id) pair
	m1 := &model.RouteSegmentMatch{
		RouteSegmentID: rs.ID,
		WorkoutID:      workout.ID,
		FirstID:        0,
		LastID:         50,
		Distance:       1000.0,
		Duration:       5 * time.Minute,
	}
	m2 := &model.RouteSegmentMatch{
		RouteSegmentID: rs.ID,
		WorkoutID:      workout.ID,
		FirstID:        60,
		LastID:         110,
		Distance:       1000.0,
		Duration:       4 * time.Minute,
	}

	require.NoError(t, db.Create(m1).Error)
	require.NoError(t, db.Create(m2).Error)
	assert.NotZero(t, m1.ID)
	assert.NotZero(t, m2.ID)
	assert.NotEqual(t, m1.ID, m2.ID)

	var loaded []model.RouteSegmentMatch
	require.NoError(t, db.Where("route_segment_id = ? AND workout_id = ?", rs.ID, workout.ID).Order("id ASC").Find(&loaded).Error)
	assert.Len(t, loaded, 2)
	assert.Equal(t, m1.ID, loaded[0].ID)
	assert.Equal(t, m2.ID, loaded[1].ID)
}
