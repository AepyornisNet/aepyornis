package model_test

import (
	"math"
	"testing"
	"time"

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

func TestRouteSegment_MultiLapEfforts(t *testing.T) {
	db := model.TestDB(t)

	rs, err := model.NewRouteSegment("", "linear.gpx", []byte(finsepiste))
	require.NoError(t, err)
	rs.Points.Points = rs.Points.Points[:10]
	rs.Circular = false
	rs.Bidirectional = true
	require.NoError(t, rs.Create(db))

	n := len(rs.Points.Points)
	// Build a single workout that does 3 laps: forward, reverse, forward
	records := make([]model.WorkoutRecord, 0, n*3)
	// Lap 1 (forward)
	for i := 0; i < n; i++ {
		p := rs.Points.Points[i]
		records = append(records, model.WorkoutRecord{Point: &p})
	}
	// Lap 2 (reverse)
	for i := 0; i < n; i++ {
		p := rs.Points.Points[n-1-i]
		records = append(records, model.WorkoutRecord{Point: &p})
	}
	// Lap 3 (forward)
	for i := 0; i < n; i++ {
		p := rs.Points.Points[i]
		records = append(records, model.WorkoutRecord{Point: &p})
	}

	multiLapGPX, err := model.RouteSegmentFromPoints(&model.Workout{Records: records}, 1, len(records))
	require.NoError(t, err)

	w, err := model.NewWorkout(testAnonymousProfile(), model.WorkoutTypeAutoDetect, "", "multilap.gpx", multiLapGPX)
	require.NoError(t, err)
	require.Len(t, w, 1)
	require.NoError(t, w[0].Save(db))

	matches, err := model.FindRouteSegmentMatches(db, rs.ID)
	require.NoError(t, err)
	assert.Len(t, matches, 3)
}

func generateTrackPoints(laps int, pointsPerLap int) []model.WorkoutRecord {
	records := make([]model.WorkoutRecord, 0, laps*pointsPerLap)
	centerLat := 50.0
	centerLng := 4.0
	mPerDegLat := 111320.0
	mPerDegLng := 111320.0 * math.Cos(centerLat*math.Pi/180.0)

	radius := 31.83     // meters
	straightLen := 100.0 // meters

	totalPerLap := 2.0*straightLen + 2.0*math.Pi*radius // ~400m
	t0 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	for lap := 0; lap < laps; lap++ {
		for i := 0; i < pointsPerLap; i++ {
			s := float64(i) / float64(pointsPerLap) * totalPerLap
			var x, y float64

			if s < straightLen {
				x = -straightLen/2.0 + s
				y = -radius
			} else if s < straightLen+math.Pi*radius {
				theta := -math.Pi/2.0 + (s-straightLen)/radius
				x = straightLen/2.0 + radius*math.Cos(theta)
				y = radius*math.Sin(theta)
			} else if s < 2.0*straightLen+math.Pi*radius {
				sTop := s - (straightLen + math.Pi*radius)
				x = straightLen/2.0 - sTop
				y = radius
			} else {
				theta := math.Pi/2.0 + (s-(2.0*straightLen+math.Pi*radius))/radius
				x = -straightLen/2.0 + radius*math.Cos(theta)
				y = radius*math.Sin(theta)
			}

			lat := centerLat + y/mPerDegLat
			lng := centerLng + x/mPerDegLng
			p := gogis.Point{Lat: lat, Lng: lng}
			records = append(records, model.WorkoutRecord{
				Point:     &p,
				Time:      t0.Add(time.Duration(len(records)) * time.Second),
				SortOrder: len(records),
			})
		}
	}
	return records
}

func TestRouteSegment_TrackMatchingMultiLapBenchmark(t *testing.T) {
	db := model.TestDB(t)

	// 1 lap template for route segment (80 points ~ 400m)
	lapRecords := generateTrackPoints(1, 80)
	gpxContent, err := model.RouteSegmentFromPoints(&model.Workout{Records: lapRecords}, 1, len(lapRecords))
	require.NoError(t, err)

	rs, err := model.NewRouteSegment("400m Track", "track.gpx", gpxContent)
	require.NoError(t, err)
	rs.Circular = true
	rs.Bidirectional = false
	require.NoError(t, rs.Create(db))

	// Create 2 workouts:
	// Workout 1: 25 laps (10,000m) = 2,000 points
	// Workout 2: 10 laps (4,000m) = 800 points
	w1Records := generateTrackPoints(25, 80)
	w1GPX, err := model.RouteSegmentFromPoints(&model.Workout{Records: w1Records}, 1, len(w1Records))
	require.NoError(t, err)
	w1, err := model.NewWorkout(testAnonymousProfile(), model.WorkoutTypeAutoDetect, "", "w1.gpx", w1GPX)
	require.NoError(t, err)
	require.NoError(t, w1[0].Save(db))

	w2Records := generateTrackPoints(10, 80)
	w2GPX, err := model.RouteSegmentFromPoints(&model.Workout{Records: w2Records}, 1, len(w2Records))
	require.NoError(t, err)
	w2, err := model.NewWorkout(testAnonymousProfile(), model.WorkoutTypeAutoDetect, "", "w2.gpx", w2GPX)
	require.NoError(t, err)
	require.NoError(t, w2[0].Save(db))

	// FindRouteSegmentMatches should find all 35 laps across both workouts in < 500ms
	start := time.Now()
	matches, err := model.FindRouteSegmentMatches(db, rs.ID)
	duration := time.Since(start)
	require.NoError(t, err)
	assert.Len(t, matches, 35)
	assert.Less(t, duration, 500*time.Millisecond)

	// Also test single workout matcher FindRouteSegmentWorkoutMatches
	w1Matches, err := model.FindRouteSegmentWorkoutMatches(db, rs.ID, w1[0].ID)
	require.NoError(t, err)
	assert.Len(t, w1Matches, 25)

	w2Matches, err := model.FindRouteSegmentWorkoutMatches(db, rs.ID, w2[0].ID)
	require.NoError(t, err)
	assert.Len(t, w2Matches, 10)

	// Test candidate discovery and explicit batching with batchSize = 1
	candidates, err := model.FindCandidateWorkoutsForRouteSegment(db, rs.ID)
	require.NoError(t, err)
	assert.Len(t, candidates, 2)

	batchedMatches, err := model.FindRouteSegmentMatchesInBatches(db, rs.ID, 1)
	require.NoError(t, err)
	assert.Len(t, batchedMatches, 35)
}

