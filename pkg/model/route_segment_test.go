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

func TestRouteSegment_FindMatches(t *testing.T) {
	rs, err := model.NewRouteSegment("", "finsepiste.gpx", []byte(finsepiste))
	assert.NoError(t, err)

	w1, err := model.NewWorkout(testAnonymousProfile(), model.WorkoutTypeAutoDetect, "", "match.gpx", []byte(track))
	assert.NoError(t, err)
	assert.Len(t, w1, 1)
	rp, wp := rs.Points.Points[0], w1[0].Records[0]
	minStart := wp.DistanceToPoint(rp)
	for i := range w1[0].Records {
		d := w1[0].Records[i].DistanceToPoint(rp)
		if d < minStart {
			minStart = d
		}
	}
	t.Logf(
		"route points=%d workout points=%d type=%s hasTracks=%v first_route=(%.5f,%.5f) first_workout=(%.5f,%.5f) min_start_distance=%.2fm",
		len(rs.Points.Points),
		len(w1[0].Records),
		w1[0].Type,
		w1[0].HasTracks(),
		rp.Lat,
		rp.Lng,
		wp.Lat(),
		wp.Lng(),
		minStart,
	)

	w1_1 := w1[0]
	assert.True(t, w1_1.Type.IsLocation())
	assert.True(t, w1_1.HasTracks())

	w2, err := model.NewWorkout(testAnonymousProfile(), model.WorkoutTypeAutoDetect, "", "nomatch.gpx", []byte(model.GpxSample1))
	assert.NoError(t, err)
	assert.Len(t, w2, 1)

	w2_1 := w2[0]
	assert.True(t, w2_1.Type.IsLocation())
	assert.True(t, w2_1.HasTracks())

	workouts := []*model.Workout{w1_1, w2_1}
	matches := rs.FindMatches(workouts)

	if !assert.Len(t, matches, 1) {
		return
	}

	assert.Len(t, matches[0].Workout.Records, 158)
}

func TestRouteSegment_StartingPoints_NoMatch(t *testing.T) {
	rs, err := model.NewRouteSegment("", "finsepiste.gpx", []byte(finsepiste))
	assert.NoError(t, err)

	w, err := model.NewWorkout(testAnonymousProfile(), model.WorkoutTypeAutoDetect, "", "nomatch.gpx", []byte(model.GpxSample1))
	assert.NoError(t, err)
	assert.Len(t, w, 1)

	w1 := w[0]
	sp := rs.StartingPoints(w1.Records)
	assert.Empty(t, sp)
}

func TestRouteSegment_StartingPoints_Match(t *testing.T) {
	rs, err := model.NewRouteSegment("", "finsepiste.gpx", []byte(finsepiste))
	assert.NoError(t, err)

	w, err := model.NewWorkout(testAnonymousProfile(), model.WorkoutTypeAutoDetect, "", "match.gpx", []byte(track))
	assert.NoError(t, err)
	assert.Len(t, w, 1)

	w1 := w[0]
	sp := rs.StartingPoints(w1.Records)
	assert.NotEmpty(t, sp)

	for _, p := range sp {
		assert.Less(t, w1.Records[p].DistanceToPoint(rs.Points.Points[0]), model.MaxDeltaMeter)
	}
}

func TestRouteSegment_StartingPoints_MatchSegment(t *testing.T) {
	rs, err := model.NewRouteSegment("", "finsepiste.gpx", []byte(finsepiste))
	assert.NoError(t, err)

	w, err := model.NewWorkout(testAnonymousProfile(), model.WorkoutTypeAutoDetect, "", "match.gpx", []byte(track))
	assert.NoError(t, err)
	assert.Len(t, w, 1)

	w1 := w[0]
	sp := rs.StartingPoints(w1.Records)
	assert.NotEmpty(t, sp)

	{
		last, ok := rs.MatchSegment(w1, 3, true)
		assert.Zero(t, last)
		assert.False(t, ok)
	}

	{
		last, ok := rs.MatchSegment(w1, 4, true)
		assert.NotZero(t, last)
		assert.True(t, ok)
	}
}

func TestRouteSegment_Match(t *testing.T) {
	rs, err := model.NewRouteSegment("", "finsepiste.gpx", []byte(finsepiste))
	assert.NoError(t, err)

	w, err := model.NewWorkout(testAnonymousProfile(), model.WorkoutTypeAutoDetect, "", "match.gpx", []byte(track))
	assert.NoError(t, err)
	assert.Len(t, w, 1)

	w1 := w[0]
	rsm := rs.Match(w1)
	if !assert.NotNil(t, rsm) {
		return
	}

	assert.Greater(t, rsm.Distance, 900.0)
	assert.True(t, rsm.MatchesDistance(rs.TotalDistance))
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
