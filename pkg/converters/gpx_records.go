package converters

import (
	"math"
	"slices"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/tkrajina/gpxgo/gpx"
	"github.com/westphae/geomag/pkg/egm96"
)

var correctAltitudeCreators = []string{
	"garmin", "Garmin", "Garmin Connect",
	"Apple Watch", "Open GPX Tracker for iOS",
	"StravaGPX iPhone", "StravaGPX",
	"Workout Tracker",
}

func MapDataAndRecordsFromGPX(gpxContent *gpx.GPX) (*model.WorkoutGeoMeta, []model.WorkoutRecord) {
	data := createMapData(gpxContent)

	points := allGPXPoints(gpxContent)
	if len(points) == 0 {
		return data, nil
	}

	records := make([]model.WorkoutRecord, 0, len(points))

	totalDist := 0.0
	totalDist2D := 0.0
	totalTime := 0.0

	for i, pt := range points {
		if !pointHasDistance(pt) {
			continue
		}

		dist := 0.0
		dist2D := 0.0
		t := 0.0

		if i+1 < len(points) {
			dist = distance3DBetween(pt, points[i+1])
			dist2D = distance2DBetween(pt, points[i+1])
			t = pt.TimeDiff(&points[i+1])

			totalDist += dist
			totalDist2D += dist2D
			totalTime += t
		}

		extraMetrics := model.ExtraMetrics{}
		extraMetrics.Set("elevation", correctAltitude(gpxContent.Creator, pt.Point.Latitude, pt.Point.Longitude, pt.Elevation.Value()))
		extraMetrics.ParseGPXExtensions(pt.Extensions)

		records = append(records, model.WorkoutRecord{
			Lat:             pt.Point.Latitude,
			Lng:             pt.Point.Longitude,
			Elevation:       pt.Elevation.Value(),
			Time:            pt.Timestamp,
			Distance:        dist,
			Distance2D:      dist2D,
			TotalDistance:   totalDist,
			TotalDistance2D: totalDist2D,
			Duration:        time.Duration(t) * time.Second,
			TotalDuration:   time.Duration(totalTime) * time.Second,
			ExtraMetrics:    extraMetrics,
		})
	}

	return data, records
}

func GPXName(gpxContent *gpx.GPX) string {
	if gpxContent == nil {
		return ""
	}

	if len(gpxContent.Tracks) > 0 && gpxContent.Tracks[0].Name != "" {
		return gpxContent.Tracks[0].Name
	}

	return gpxContent.Name
}

func GPXType(gpxContent *gpx.GPX) string {
	if gpxContent == nil || len(gpxContent.Tracks) == 0 {
		return ""
	}

	return gpxContent.Tracks[0].Type
}

func createMapData(gpxContent *gpx.GPX) *model.WorkoutGeoMeta {
	if len(gpxContent.Tracks) == 0 {
		return nil
	}

	// Now reduce the whole GPX to a single track to calculate the center
	gpxContent.ReduceGpxToSingleTrack()
	mapCenter := center(gpxContent)

	data := &model.WorkoutGeoMeta{
		Center: mapCenter,
	}

	return data
}

// allGPXPoints returns the first track segment's points
func allGPXPoints(gpxContent *gpx.GPX) []gpx.GPXPoint {
	if gpxContent == nil {
		return nil
	}

	var points []gpx.GPXPoint

	for _, track := range gpxContent.Tracks {
		for _, segment := range track.Segments {
			for _, p := range segment.Points {
				if !pointHasDistance(p) {
					continue
				}

				points = append(points, p)
			}
		}
	}

	return points
}

func pointHasDistance(p gpx.GPXPoint) bool {
	if math.IsNaN(p.Latitude) || math.IsNaN(p.Longitude) {
		return false
	}

	return true
}

// Determines the date to use for the workout
func GPXDate(gpxContent *gpx.GPX) *time.Time {
	// Use the first track's first segment's timestamp if it exists
	// This is the best time to use as a start time, since converters shouldn't
	// touch this timestamp
	if len(gpxContent.Tracks) > 0 {
		if t := gpxContent.Tracks[0]; len(t.Segments) > 0 {
			if s := t.Segments[0]; len(s.Points) > 0 {
				if !s.Points[0].Timestamp.IsZero() {
					return &s.Points[0].Timestamp
				}
			}
		}
	}

	// Otherwise, return the timestamp from the metadata, use that (not all apps have
	// this, notably Workoutdoors doesn't)
	// If this is nil, this should result in an error and the user will be alerted.
	return gpxContent.Time
}

func distance2DBetween(p1 gpx.GPXPoint, p2 gpx.GPXPoint) float64 {
	return p2.Distance2D(&p1)
}

func distance3DBetween(p1 gpx.GPXPoint, p2 gpx.GPXPoint) float64 {
	return p2.Distance3D(&p1)
}

func correctAltitude(creator string, lat, long, alt float64) float64 {
	if !creatorNeedsCorrection(creator) {
		return alt
	}

	lat = normalizeDegrees(lat)
	long = normalizeDegrees(long)

	loc := egm96.NewLocationGeodetic(lat, long, alt)

	h, err := loc.HeightAboveMSL()
	if err != nil {
		return alt
	}

	return h
}

// center returns the center point (lat, lng) of gpx points
func center(gpxContent *gpx.GPX) model.MapCenter {
	points := allGPXPoints(gpxContent)

	if len(points) == 0 {
		return model.MapCenter{}
	}

	lat, lng := 0.0, 0.0

	for _, pt := range points {
		lat += pt.Point.Latitude
		lng += pt.Point.Longitude
	}

	size := float64(len(points))

	mc := model.MapCenter{
		Lat: lat / size,
		Lng: lng / size,
	}

	mc.UpdateTimezone()

	return mc
}

func creatorNeedsCorrection(creator string) bool {
	return !slices.Contains(correctAltitudeCreators, creator)
}

func normalizeDegrees(val float64) float64 {
	if val < 0 {
		return val + 360
	}

	return val
}
