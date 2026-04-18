package model

import (
	"math"
	"slices"
	"time"

	"github.com/codingsince1985/geo-golang"
	"github.com/jovandeginste/workout-tracker/v2/pkg/geocoder"
	"github.com/labstack/gommon/log"
	"github.com/paulmach/orb"
	"github.com/tkrajina/gpxgo/gpx"
	"github.com/westphae/geomag/pkg/egm96"
	"gorm.io/gorm"
)

const UnknownLocation = "(unknown location)"

const (
	mapDataPointsInsertBatchSize = 500
	mapDataClimbsInsertBatchSize = 500
)

var correctAltitudeCreators = []string{
	"garmin", "Garmin", "Garmin Connect",
	"Apple Watch", "Open GPX Tracker for iOS",
	"StravaGPX iPhone", "StravaGPX",
	"Workout Tracker",
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

// MapDataRangeStats describes aggregate statistics for a contiguous slice of map points.
// It is intentionally rich so callers can derive per-item breakdowns as well as overall averages.
type MapDataRangeStats struct {
	WorkoutStats

	Distance       float64       // Distance covered in this range
	Duration       time.Duration // Total duration in this range (including pauses)
	MovingDuration time.Duration // Duration while moving (based on speed threshold)
	PauseDuration  time.Duration // Duration spent paused
}

// MapCenter is the center of the workout
type MapCenter struct {
	TZ  string  `json:"tz"`  // Timezone
	Lat float64 `json:"lat"` // Latitude
	Lng float64 `json:"lng"` // Longitude
}

func (m *MapCenter) ToOrbPoint() *orb.Point {
	return &orb.Point{m.Lng, m.Lat}
}

func PreloadWorkoutData(db *gorm.DB) *gorm.DB {
	return db.
		Preload("Stats").
		Preload("Data").
		Preload("Laps", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("sort_order ASC")
		}).
		Preload("Laps.Stats").
		Preload("Climbs", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("sort_order ASC")
		}).
		Preload("Attachments", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("sort_order ASC").Order("id ASC")
		})
}

func PreloadWorkoutDetails(db *gorm.DB) *gorm.DB {
	return PreloadWorkoutData(db).
		Preload("Events", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("sort_order ASC")
		}).
		Preload("Records", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("sort_order ASC")
		})
}

// StatsForRange aggregates statistics for a slice of points identified by start and end indices (inclusive).
// Returns false when the provided range is invalid or the data contains no points.
func StatsForRange(points []WorkoutRecord, startIdx, endIdx int) (MapDataRangeStats, bool) {
	stats := MapDataRangeStats{}

	if len(points) == 0 || startIdx < 0 || endIdx >= len(points) || startIdx > endIdx {
		return stats, false
	}

	firstElevation := points[startIdx].EnhancedElevation()
	stats.MinElevation = firstElevation
	stats.MaxElevation = firstElevation

	aggregator := newRangeAggregator(&stats, startIdx)
	aggregator.processMetrics(points, startIdx, endIdx)
	aggregator.processDurations(points, startIdx, endIdx)
	aggregator.finalize()
	return stats, true
}

type rangeAggregator struct {
	stats      *MapDataRangeStats
	minSetFrom int

	sumCadence float64
	cadCount   int
	maxCadence float64
	minCadence float64
	foundCad   bool

	sumHR   float64
	hrCnt   int
	maxHR   float64
	minHR   float64
	foundHR bool

	sumRR   float64
	rrCnt   int
	maxRR   float64
	minRR   float64
	foundRR bool

	sumPower   float64
	powerCnt   int
	maxPower   float64
	minPower   float64
	foundPower bool

	sumTemp   float64
	tempCnt   int
	minTemp   float64
	maxTemp   float64
	foundTemp bool

	sumSlope   float64
	slopeCnt   int
	foundSlope bool

	minSpeed   float64
	foundSpeed bool
}

func newRangeAggregator(stats *MapDataRangeStats, startIdx int) *rangeAggregator {
	return &rangeAggregator{stats: stats, minSetFrom: startIdx}
}

func (r *rangeAggregator) processMetrics(points []WorkoutRecord, startIdx, endIdx int) {
	for i := startIdx; i <= endIdx; i++ {
		p := points[i]

		r.handleElevation(p)
		r.handleSlope(p)
		r.handleUpDown(points, i, startIdx)
		r.handleCadence(p)
		r.handleHeartRate(p)
		r.handleRespirationRate(p)
		r.handlePower(p)
		r.handleTemperature(p)
	}
}

func (r *rangeAggregator) handleElevation(p WorkoutRecord) {
	ele := p.EnhancedElevation()

	r.stats.MinElevation = min(r.stats.MinElevation, ele)
	r.stats.MaxElevation = max(r.stats.MaxElevation, ele)
}

func (r *rangeAggregator) handleSlope(p WorkoutRecord) {
	r.sumSlope += p.SlopeGrade
	r.slopeCnt++

	if !r.foundSlope {
		r.stats.MinSlope = p.SlopeGrade
		r.stats.MaxSlope = p.SlopeGrade
		r.foundSlope = true
		return
	}

	r.stats.MinSlope = min(r.stats.MinSlope, p.SlopeGrade)
	r.stats.MaxSlope = max(r.stats.MaxSlope, p.SlopeGrade)
}

func (r *rangeAggregator) handleUpDown(points []WorkoutRecord, idx, startIdx int) {
	if idx <= startIdx {
		return
	}

	curr := points[idx].EnhancedElevation()
	prev := points[idx-1].EnhancedElevation()
	delta := curr - prev
	if delta > 0 {
		r.stats.TotalUp += delta
		return
	}

	r.stats.TotalDown += -delta
}

func (r *rangeAggregator) handleCadence(p WorkoutRecord) {
	cad, ok := p.ExtraMetrics["cadence"]
	if !ok || cad <= 0 {
		return
	}

	r.sumCadence += cad
	r.cadCount++
	r.maxCadence = max(r.maxCadence, cad)

	if !r.foundCad || cad < r.minCadence {
		r.minCadence = cad
		r.foundCad = true
	}
}

func (r *rangeAggregator) handleHeartRate(p WorkoutRecord) {
	hr, ok := p.ExtraMetrics["heart-rate"]
	if !ok || hr <= 0 {
		return
	}

	r.sumHR += hr
	r.hrCnt++
	r.maxHR = max(r.maxHR, hr)

	if !r.foundHR || hr < r.minHR {
		r.minHR = hr
		r.foundHR = true
	}
}

func (r *rangeAggregator) handleRespirationRate(p WorkoutRecord) {
	rr, ok := p.ExtraMetrics["respiration-rate"]
	if !ok || rr <= 0 {
		return
	}

	r.sumRR += rr
	r.rrCnt++
	r.maxRR = max(r.maxRR, rr)

	if !r.foundRR || rr < r.minRR {
		r.minRR = rr
		r.foundRR = true
	}
}

func (r *rangeAggregator) handlePower(p WorkoutRecord) {
	power, ok := p.ExtraMetrics["power"]
	if !ok || power <= 0 {
		return
	}

	r.sumPower += power
	r.powerCnt++
	r.maxPower = max(r.maxPower, power)

	if !r.foundPower || power < r.minPower {
		r.minPower = power
		r.foundPower = true
	}
}

func (r *rangeAggregator) handleTemperature(p WorkoutRecord) {
	temp, ok := p.ExtraMetrics["temperature"]
	if !ok || math.IsNaN(temp) {
		return
	}

	r.sumTemp += temp
	r.tempCnt++

	if !r.foundTemp {
		r.foundTemp = true
		r.minTemp = temp
		r.maxTemp = temp
	}

	if temp < r.minTemp {
		r.minTemp = temp
	}

	r.maxTemp = max(r.maxTemp, temp)
}

func (r *rangeAggregator) processDurations(points []WorkoutRecord, startIdx, endIdx int) {
	for i := startIdx; i <= endIdx; i++ {
		p := points[i]

		r.stats.Distance += p.Distance
		r.stats.Duration += p.Duration

		speed := p.AverageSpeed()
		if metricSpeed, ok := p.ExtraMetrics["speed"]; ok && !math.IsNaN(metricSpeed) && metricSpeed > 0 {
			speed = metricSpeed
		}
		r.stats.MaxSpeed = max(r.stats.MaxSpeed, speed)

		if speed*3.6 >= 1.0 {
			r.stats.MovingDuration += p.Duration

			if !r.foundSpeed || speed < r.minSpeed {
				r.minSpeed = speed
				r.foundSpeed = true
			}
		} else {
			r.stats.PauseDuration += p.Duration
		}
	}
}

func (r *rangeAggregator) finalize() {
	if r.stats.Duration > 0 {
		r.stats.AverageSpeed = r.stats.Distance / r.stats.Duration.Seconds()
	}

	if r.stats.MovingDuration > 0 {
		r.stats.AverageSpeedNoPause = r.stats.Distance / r.stats.MovingDuration.Seconds()
	}

	if r.cadCount > 0 {
		r.stats.AverageCadence = r.sumCadence / float64(r.cadCount)
		r.stats.MaxCadence = r.maxCadence
		if r.foundCad {
			r.stats.MinCadence = r.minCadence
		}
	}

	if r.hrCnt > 0 {
		r.stats.AverageHeartRate = r.sumHR / float64(r.hrCnt)
		r.stats.MaxHeartRate = r.maxHR
		if r.foundHR {
			r.stats.MinHeartRate = r.minHR
		}
	}

	if r.rrCnt > 0 {
		r.stats.AverageRespirationRate = r.sumRR / float64(r.rrCnt)
		r.stats.MaxRespirationRate = r.maxRR
		if r.foundRR {
			r.stats.MinRespirationRate = r.minRR
		}
	}

	if r.powerCnt > 0 {
		r.stats.AveragePower = r.sumPower / float64(r.powerCnt)
		r.stats.MaxPower = r.maxPower
		if r.foundPower {
			r.stats.MinPower = r.minPower
		}
	}

	if r.tempCnt > 0 {
		r.stats.AverageTemperature = r.sumTemp / float64(r.tempCnt)
		if r.foundTemp {
			r.stats.MinTemperature = r.minTemp
			r.stats.MaxTemperature = r.maxTemp
		}
	}

	if r.slopeCnt > 0 {
		r.stats.AverageSlope = r.sumSlope / float64(r.slopeCnt)
	}

	if r.foundSpeed {
		r.stats.MinSpeed = r.minSpeed
	}
}

// center returns the center point (lat, lng) of gpx points
func center(gpxContent *gpx.GPX) MapCenter {
	points := allGPXPoints(gpxContent)

	if len(points) == 0 {
		return MapCenter{}
	}

	lat, lng := 0.0, 0.0

	for _, pt := range points {
		lat += pt.Point.Latitude
		lng += pt.Point.Longitude
	}

	size := float64(len(points))

	mc := MapCenter{
		Lat: lat / size,
		Lng: lng / size,
	}

	mc.updateTimezone()

	return mc
}

func (m *MapCenter) updateTimezone() {
	m.TZ = ""

	if tzFinder != nil {
		m.TZ = tzFinder.GetTimezoneName(m.Lng, m.Lat)
	}

	if m.TZ == "" {
		m.TZ = time.UTC.String()
	}
}

func (m *MapCenter) IsZero() bool {
	return m.Lat == 0 && m.Lng == 0
}

func (m *MapCenter) Address() *geo.Address {
	if m.IsZero() {
		return nil
	}

	r, err := geocoder.Reverse(geocoder.Query{
		Lat:    m.Lat,
		Lon:    m.Lng,
		Format: "json",
	})
	if err != nil {
		log.Warn("Error performing reverse geocode: ", err)
		return nil
	}

	return r
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

func createMapData(gpxContent *gpx.GPX) *WorkoutGeoMeta {
	if len(gpxContent.Tracks) == 0 {
		return nil
	}

	// Now reduce the whole GPX to a single track to calculate the center
	gpxContent.ReduceGpxToSingleTrack()
	mapCenter := center(gpxContent)

	data := &WorkoutGeoMeta{
		Center: mapCenter,
	}

	return data
}

func MapDataAndRecordsFromGPX(gpxContent *gpx.GPX) (*WorkoutGeoMeta, []WorkoutRecord) {
	data := createMapData(gpxContent)

	points := allGPXPoints(gpxContent)
	if len(points) == 0 {
		return data, nil
	}

	records := make([]WorkoutRecord, 0, len(points))

	totalDist := 0.0
	totalDist2D := 0.0
	totalTime := 0.0
	prevPoint := points[0]

	for i, pt := range points {
		if !pointHasDistance(pt) {
			continue
		}

		dist := 0.0
		dist2D := 0.0
		t := 0.0

		if i > 0 {
			dist = distance3DBetween(prevPoint, pt)
			dist2D = distance2DBetween(prevPoint, pt)
			t = pt.TimeDiff(&prevPoint)

			prevPoint = pt

			totalDist += dist
			totalDist2D += dist2D
			totalTime += t
		}

		extraMetrics := ExtraMetrics{}
		extraMetrics.Set("elevation", correctAltitude(gpxContent.Creator, pt.Point.Latitude, pt.Point.Longitude, pt.Elevation.Value()))
		extraMetrics.ParseGPXExtensions(pt.Extensions)

		records = append(records, WorkoutRecord{
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

func WorkoutStatsFromRecords(points []WorkoutRecord) WorkoutStats {
	if len(points) < 2 {
		return WorkoutStats{}
	}

	stats, ok := StatsForRange(points, 0, len(points)-1)
	if !ok {
		return WorkoutStats{}
	}

	return stats.WorkoutStats
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

func WorkoutTotalsFromRecords(records []WorkoutRecord) (float64, float64, time.Duration) {
	if len(records) == 0 {
		return 0, 0, 0
	}

	last := records[len(records)-1]

	return last.TotalDistance, last.TotalDistance2D, last.TotalDuration
}

func WorkoutEndFromRecords(records []WorkoutRecord) time.Time {
	if len(records) == 0 {
		return time.Time{}
	}

	return records[len(records)-1].Time
}

func WorkoutPauseDurationFromAverages(totalDistance float64, totalDuration time.Duration, averageSpeedNoPause float64) time.Duration {
	if totalDistance <= 0 || totalDuration <= 0 || averageSpeedNoPause <= 0 {
		return 0
	}

	movingDuration := time.Duration((totalDistance / averageSpeedNoPause) * float64(time.Second))
	if movingDuration >= totalDuration {
		return 0
	}

	return totalDuration - movingDuration
}

func MapDataFromGPX(gpxContent *gpx.GPX) *WorkoutGeoMeta {
	data, _ := MapDataAndRecordsFromGPX(gpxContent)

	return data
}
