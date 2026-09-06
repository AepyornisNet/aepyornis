package model_test

import (
	"testing"

	_ "github.com/AepyornisNet/aepyornis/pkg/converters"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/restayway/gogis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testAnonymousProfile() *model.Profile {
	return &model.Profile{Username: "anonymous", DisplayName: "Anonymous"}
}

func TestRouteSegment_Parse(t *testing.T) {
	{
		rs, err := model.NewRouteSegment("", "meer.gpx", []byte(meer))
		assert.NoError(t, err)
		assert.NotNil(t, rs)
		assert.Greater(t, rs.TotalDistance, 1800.0)
	}

	{
		rs, err := model.NewRouteSegment("", "finsepiste.gpx", []byte(finsepiste))
		assert.NoError(t, err)
		assert.NotNil(t, rs)
		assert.Greater(t, rs.TotalDistance, 900.0)
	}
}

func TestRouteSegment_FindMatches_PostGIS(t *testing.T) {
	db := model.TestDB(t)

	rs, err := model.NewRouteSegment("", "finsepiste.gpx", []byte(finsepiste))
	require.NoError(t, err)
	require.NoError(t, rs.Create(db))

	// Create matching workout that traverses the complete route
	matchGPX, err := model.RouteSegmentFromPoints(&model.Workout{
		Records: func() []model.WorkoutRecord {
			records := make([]model.WorkoutRecord, len(rs.Points.Points))
			for i, pt := range rs.Points.Points {
				p := pt
				records[i] = model.WorkoutRecord{Point: &p}
			}
			return records
		}(),
	}, 1, len(rs.Points.Points))
	require.NoError(t, err)

	w1, err := model.NewWorkout(testAnonymousProfile(), model.WorkoutTypeAutoDetect, "", "match.gpx", matchGPX)
	require.NoError(t, err)
	require.Len(t, w1, 1)
	require.NoError(t, w1[0].Save(db))

	w2, err := model.NewWorkout(testAnonymousProfile(), model.WorkoutTypeAutoDetect, "", "nomatch.gpx", []byte(model.GpxSample1))
	require.NoError(t, err)
	require.Len(t, w2, 1)
	require.NoError(t, w2[0].Save(db))

	matches, err := model.FindRouteSegmentMatches(db, rs.ID)
	require.NoError(t, err)
	require.Len(t, matches, 1)

	assert.Equal(t, w1[0].ID, matches[0].WorkoutID)
	assert.Equal(t, rs.ID, matches[0].RouteSegmentID)
	assert.Greater(t, matches[0].Distance, 900.0)
	assert.Equal(t, 0, matches[0].FirstID)
	assert.Greater(t, matches[0].LastID, matches[0].FirstID)
}

func TestRouteSegment_BidirectionalMatching(t *testing.T) {
	db := model.TestDB(t)

	rsLinear, err := model.NewRouteSegment("", "linear.gpx", []byte(finsepiste))
	require.NoError(t, err)
	// Take a linear subsegment (first 10 points)
	rsLinear.Points.Points = rsLinear.Points.Points[:10]
	rsLinear.Circular = false
	rsLinear.Bidirectional = false
	require.NoError(t, rsLinear.Create(db))

	forwardGPX, err := model.RouteSegmentFromPoints(&model.Workout{
		Records: func() []model.WorkoutRecord {
			records := make([]model.WorkoutRecord, len(rsLinear.Points.Points))
			for i, pt := range rsLinear.Points.Points {
				p := pt
				records[i] = model.WorkoutRecord{Point: &p}
			}
			return records
		}(),
	}, 1, len(rsLinear.Points.Points))
	require.NoError(t, err)

	wFwd, err := model.NewWorkout(testAnonymousProfile(), model.WorkoutTypeAutoDetect, "", "fwd.gpx", forwardGPX)
	require.NoError(t, err)
	require.Len(t, wFwd, 1)
	require.NoError(t, wFwd[0].Save(db))

	reverseGPX, err := model.RouteSegmentFromPoints(&model.Workout{
		Records: func() []model.WorkoutRecord {
			n := len(rsLinear.Points.Points)
			records := make([]model.WorkoutRecord, n)
			for i := 0; i < n; i++ {
				p := rsLinear.Points.Points[n-1-i]
				records[i] = model.WorkoutRecord{Point: &p}
			}
			return records
		}(),
	}, 1, len(rsLinear.Points.Points))
	require.NoError(t, err)

	wRev, err := model.NewWorkout(testAnonymousProfile(), model.WorkoutTypeAutoDetect, "", "rev.gpx", reverseGPX)
	require.NoError(t, err)
	require.Len(t, wRev, 1)
	require.NoError(t, wRev[0].Save(db))

	// When Bidirectional is false: only forward matches
	matches1, err := model.FindRouteSegmentMatches(db, rsLinear.ID)
	require.NoError(t, err)
	assert.Len(t, matches1, 1)
	assert.Equal(t, wFwd[0].ID, matches1[0].WorkoutID)

	// When Bidirectional is true: both forward and reverse match
	rsLinear.Bidirectional = true
	require.NoError(t, rsLinear.Save(db))

	matches2, err := model.FindRouteSegmentMatches(db, rsLinear.ID)
	require.NoError(t, err)
	assert.Len(t, matches2, 2)
}

func TestRouteSegment_RematchRouteSegment(t *testing.T) {
	db := model.TestDB(t)

	rs, err := model.NewRouteSegment("", "finsepiste.gpx", []byte(finsepiste))
	require.NoError(t, err)
	rs.Dirty = true
	require.NoError(t, rs.Create(db))

	matchGPX, err := model.RouteSegmentFromPoints(&model.Workout{
		Records: func() []model.WorkoutRecord {
			records := make([]model.WorkoutRecord, len(rs.Points.Points))
			for i, pt := range rs.Points.Points {
				p := pt
				records[i] = model.WorkoutRecord{Point: &p}
			}
			return records
		}(),
	}, 1, len(rs.Points.Points))
	require.NoError(t, err)

	w1, err := model.NewWorkout(testAnonymousProfile(), model.WorkoutTypeAutoDetect, "", "match.gpx", matchGPX)
	require.NoError(t, err)
	require.Len(t, w1, 1)
	require.NoError(t, w1[0].Save(db))

	require.NoError(t, model.RematchRouteSegment(db, rs.ID))

	var matches []*model.RouteSegmentMatch
	require.NoError(t, db.Where("route_segment_id = ?", rs.ID).Find(&matches).Error)
	require.Len(t, matches, 1)
	assert.Equal(t, w1[0].ID, matches[0].WorkoutID)

	var reloaded model.RouteSegment
	require.NoError(t, db.First(&reloaded, rs.ID).Error)
	assert.False(t, reloaded.Dirty)
}

func TestRouteSegment_FindWorkoutRouteSegmentMatches(t *testing.T) {
	db := model.TestDB(t)

	rs, err := model.NewRouteSegment("", "finsepiste.gpx", []byte(finsepiste))
	require.NoError(t, err)
	require.NoError(t, rs.Create(db))

	matchGPX, err := model.RouteSegmentFromPoints(&model.Workout{
		Records: func() []model.WorkoutRecord {
			records := make([]model.WorkoutRecord, len(rs.Points.Points))
			for i, pt := range rs.Points.Points {
				p := pt
				records[i] = model.WorkoutRecord{Point: &p}
			}
			return records
		}(),
	}, 1, len(rs.Points.Points))
	require.NoError(t, err)

	w1, err := model.NewWorkout(testAnonymousProfile(), model.WorkoutTypeAutoDetect, "", "match.gpx", matchGPX)
	require.NoError(t, err)
	require.Len(t, w1, 1)
	require.NoError(t, w1[0].Save(db))

	matches, err := model.FindWorkoutRouteSegmentMatches(db, w1[0].ID)
	require.NoError(t, err)
	require.Len(t, matches, 1)
	assert.Equal(t, rs.ID, matches[0].RouteSegmentID)

	w2, err := model.NewWorkout(testAnonymousProfile(), model.WorkoutTypeAutoDetect, "", "nomatch.gpx", []byte(model.GpxSample1))
	require.NoError(t, err)
	require.Len(t, w2, 1)
	require.NoError(t, w2[0].Save(db))

	matches2, err := model.FindWorkoutRouteSegmentMatches(db, w2[0].ID)
	require.NoError(t, err)
	assert.Empty(t, matches2)
}

func TestRouteSegment_DatabaseSaveAndGet(t *testing.T) {
	db := model.TestDB(t)

	rs, err := model.NewRouteSegment("test notes", "finsepiste.gpx", []byte(finsepiste))
	assert.NoError(t, err)
	require.NoError(t, rs.Create(db))
	assert.NotZero(t, rs.ID)

	var loaded model.RouteSegment
	require.NoError(t, db.First(&loaded, rs.ID).Error)
	assert.Equal(t, rs.Name, loaded.Name)
	assert.Equal(t, len(rs.Points.Points), len(loaded.Points.Points))
	if len(rs.Points.Points) > 0 {
		assert.InDelta(t, rs.Points.Points[0].Lat, loaded.Points.Points[0].Lat, 0.0001)
		assert.InDelta(t, rs.Points.Points[0].Lng, loaded.Points.Points[0].Lng, 0.0001)
	}
}

func TestRouteSegment_RouteSegmentFromPoints_FiltersInvalidCoordinates(t *testing.T) {
	workout := &model.Workout{
		Records: []model.WorkoutRecord{
			{Point: nil},
			{Point: &gogis.Point{Lat: 0, Lng: 0}},
			{Point: &gogis.Point{Lat: 50.95786, Lng: 4.72410}},
			{Point: &gogis.Point{Lat: 0, Lng: 0}},
			{Point: &gogis.Point{Lat: 50.95816, Lng: 4.72391}},
			{Point: &gogis.Point{Lat: 50.95900, Lng: 4.72500}},
			{Point: nil},
		},
	}

	content, err := model.RouteSegmentFromPoints(workout, 1, 3)
	require.NoError(t, err)
	require.NotEmpty(t, content)

	rs, err := model.NewRouteSegment("", "test.gpx", content)
	require.NoError(t, err)
	assert.Len(t, rs.Points.Points, 3)
	assert.InDelta(t, 50.95786, rs.Points.Points[0].Lat, 0.0001)
	assert.InDelta(t, 50.95816, rs.Points.Points[1].Lat, 0.0001)
	assert.InDelta(t, 50.95900, rs.Points.Points[2].Lat, 0.0001)
}
