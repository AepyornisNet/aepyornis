package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// Route segment matching thresholds and constants
const (
	// MaxDeltaMeter is the maximum distance in meters that a point can be away from
	// the route segment for slope calculations and legacy fallbacks.
	MaxDeltaMeter = 20.0

	// RouteSegmentBoundingBoxExpansionDegrees is the bounding box buffer in degrees
	// used for the spatial index pre-filter (~1.1 km at the equator).
	RouteSegmentBoundingBoxExpansionDegrees = 0.01

	// RouteSegmentZoneBufferMeters is the buffer radius in meters around the start
	// and end points to detect entry and exit zones on the WGS84 ellipsoid.
	RouteSegmentZoneBufferMeters = 25.0

	// RouteSegmentMinPointInterval is the minimum number of points between start and
	// exit points required for a valid segment effort.
	RouteSegmentMinPointInterval = 5

	// RouteSegmentMinLengthFraction is the minimum fraction of the route's total length
	// an effort track must cover to be considered a match (0.85 = 85%).
	RouteSegmentMinLengthFraction = 0.85

	// RouteSegmentMaxHausdorffDistance is the maximum Hausdorff distance threshold in
	// projected units (EPSG:3857) to ensure overall shape similarity.
	RouteSegmentMaxHausdorffDistance = 50.0

	// RouteSegmentWorkoutBatchSize is the default number of workouts processed in a single
	// matching query batch to prevent high database memory consumption.
	RouteSegmentWorkoutBatchSize = 50
)

// RouteSegmentMatch is a match between a route segment and a workout
type RouteSegmentMatch struct {
	ID           uint64        `gorm:"primaryKey;autoIncrement" json:"id"`
	Workout      *Workout      `json:"workout"`
	RouteSegment *RouteSegment `json:"routeSegment"`

	RouteSegmentID uint64        `gorm:"not null;index" json:"routeSegmentID"` // The ID of the route segment
	WorkoutID      uint64        `gorm:"not null;index" json:"workoutID"`      // The ID of the workout
	FirstID        int           `json:"firstID"`                              // The index of the first point of the route
	LastID         int           `json:"lastID"`                               // The index of the last point of the route
	Distance       float64       `json:"distance"`                             // The total distance of the route segment for this workout
	Duration       time.Duration `json:"duration"`                             // The total duration of the route segment for this workout
}

func (rsm *RouteSegmentMatch) AverageSpeed() float64 {
	if rsm.Duration.Seconds() == 0 {
		return 0
	}
	return rsm.Distance / rsm.Duration.Seconds()
}

type matchQueryResult struct {
	WorkoutID   uint64    `gorm:"column:workout_id"`
	StartSort   int       `gorm:"column:start_sort"`
	EndSort     int       `gorm:"column:end_sort"`
	StartTime   time.Time `gorm:"column:start_time"`
	EndTime     time.Time `gorm:"column:end_time"`
	TrackLength float64   `gorm:"column:track_length"`
}

const matchRouteSegmentWorkoutQuery = `
WITH route AS (
    SELECT 
        id,
        ST_Transform(ST_SetSRID(points, 4326), 3857) AS geom,
        ST_StartPoint(ST_SetSRID(points, 4326)) AS start_geom,
        ST_EndPoint(ST_SetSRID(points, 4326)) AS end_geom,
        ST_Length(points::geography) AS length_m,
        COALESCE(circular, false) AS is_circular,
        COALESCE(bidirectional, false) AS is_bidirectional
    FROM route_segments 
    WHERE id = ?
),
-- 1. Index-powered Bounding Box (Discards irrelevant points)
local_points AS (
    SELECT 
        w.workout_id, w.sort_order, w.time, w.point
    FROM workout_records w
    JOIN route_segments rs ON rs.id = ?
    WHERE w.workout_id = ?
      AND w.point IS NOT NULL 
      AND w.point && ST_Expand(ST_SetSRID(rs.points, 4326), ?)
),
-- 2. Tag zone entries (1 = Start, 2 = End, 0 = Outside)
tagged_zones AS (
    SELECT 
        p.workout_id, p.sort_order, p.time, p.point,
        CASE 
            WHEN ST_DWithin(p.point::geography, r.start_geom::geography, ?) THEN 1
            WHEN NOT r.is_circular AND ST_DWithin(p.point::geography, r.end_geom::geography, ?) THEN 2
            ELSE 0
        END AS zone_id,
        r.is_circular,
        r.is_bidirectional
    FROM local_points p CROSS JOIN route r
),
-- 3. Detect zone transitions (Gaps and Islands)
zone_steps AS (
    SELECT 
        *,
        CASE WHEN LAG(zone_id) OVER (PARTITION BY workout_id ORDER BY sort_order) = zone_id THEN 0 ELSE 1 END AS is_step
    FROM tagged_zones
),
zone_clusters AS (
    SELECT 
        *,
        SUM(is_step) OVER (PARTITION BY workout_id ORDER BY sort_order) AS cluster_id
    FROM zone_steps
),
-- 4. Summarize contiguous visits to zones
cluster_summary AS (
    SELECT 
        workout_id,
        cluster_id,
        zone_id,
        MIN(sort_order) AS first_sort,
        MAX(sort_order) AS last_sort,
        is_circular,
        is_bidirectional
    FROM zone_clusters
    GROUP BY workout_id, cluster_id, zone_id, is_circular, is_bidirectional
),
-- 5. Pair valid departure -> outside -> arrival transitions
cluster_pairs AS (
    SELECT 
        c.workout_id,
        c.first_sort AS start_sort,
        LEAD(c.first_sort, 2) OVER (PARTITION BY c.workout_id ORDER BY c.cluster_id) AS end_sort,
        c.zone_id AS start_zone,
        LEAD(c.zone_id, 1) OVER (PARTITION BY c.workout_id ORDER BY c.cluster_id) AS mid_zone,
        LEAD(c.zone_id, 2) OVER (PARTITION BY c.workout_id ORDER BY c.cluster_id) AS end_zone,
        c.is_circular,
        c.is_bidirectional
    FROM cluster_summary c
),
candidate_efforts AS (
    SELECT 
        workout_id,
        start_sort,
        end_sort
    FROM cluster_pairs
    WHERE mid_zone = 0 -- THE TRIPWIRE: Must leave both zones
      AND end_sort IS NOT NULL
      AND end_sort > start_sort + ? -- RouteSegmentMinPointInterval
      AND (
          -- Circular: Start -> Start
          (is_circular AND start_zone = 1 AND end_zone = 1)
          OR
          -- Standard: Start -> End
          (NOT is_circular AND NOT is_bidirectional AND start_zone = 1 AND end_zone = 2)
          OR
          -- Bidirectional: Start -> End OR End -> Start
          (NOT is_circular AND is_bidirectional AND (
              (start_zone = 1 AND end_zone = 2) OR 
              (start_zone = 2 AND end_zone = 1)
          ))
      )
),
-- 6. Reconstruct the geometry strictly for the isolated match
effort_tracks AS (
    SELECT 
        ce.workout_id,
        ce.start_sort,
        ce.end_sort,
        MIN(lp.time) AS start_time,
        MAX(lp.time) AS end_time,
        ST_MakeLine(ST_Transform(ST_SetSRID(lp.point, 4326), 3857) ORDER BY lp.sort_order) AS track_geom,
        ST_Length(ST_MakeLine(lp.point ORDER BY lp.sort_order)::geography) AS track_length
    FROM candidate_efforts ce
    JOIN local_points lp 
      ON lp.workout_id = ce.workout_id 
     AND lp.sort_order BETWEEN ce.start_sort AND ce.end_sort
    GROUP BY ce.workout_id, ce.start_sort, ce.end_sort
)
-- 7. Validate Distance and Shape
SELECT 
    e.workout_id,
    e.start_sort,
    e.end_sort,
    e.start_time,
    e.end_time,
    e.track_length
FROM effort_tracks e
CROSS JOIN route r
WHERE 
    e.track_length >= (r.length_m * ?) 
    AND ST_HausdorffDistance(e.track_geom, r.geom) < ? 
ORDER BY e.workout_id, e.start_time;
`

const candidateWorkoutsForRouteSegmentQuery = `
SELECT DISTINCT w.workout_id 
FROM workout_records w
JOIN route_segments rs ON rs.id = ?
WHERE w.point IS NOT NULL 
  	AND w.point && ST_Expand(ST_SetSRID(rs.points, 4326), ?)
ORDER BY w.workout_id ASC;
`

const matchRouteSegmentBatchedQuery = `
WITH route AS (
    SELECT 
        id,
        ST_Transform(ST_SetSRID(points, 4326), 3857) AS geom,
        ST_StartPoint(ST_SetSRID(points, 4326)) AS start_geom,
        ST_EndPoint(ST_SetSRID(points, 4326)) AS end_geom,
        ST_Length(points::geography) AS length_m,
        COALESCE(circular, false) AS is_circular,
        COALESCE(bidirectional, false) AS is_bidirectional
    FROM route_segments 
    WHERE id = ?
),
-- 1. Index-powered Bounding Box filtered by the workout batch
local_points AS (
    SELECT 
        w.workout_id, w.sort_order, w.time, w.point
    FROM workout_records w
    JOIN route_segments rs ON rs.id = ?
    WHERE w.workout_id IN (?)
      AND w.point IS NOT NULL 
      AND w.point && ST_Expand(ST_SetSRID(rs.points, 4326), ?)
),
-- 2. Tag zone entries (1 = Start, 2 = End, 0 = Outside)
tagged_zones AS (
    SELECT 
        p.workout_id, p.sort_order, p.time, p.point,
        CASE 
            WHEN ST_DWithin(p.point::geography, r.start_geom::geography, ?) THEN 1
            WHEN NOT r.is_circular AND ST_DWithin(p.point::geography, r.end_geom::geography, ?) THEN 2
            ELSE 0
        END AS zone_id,
        r.is_circular,
        r.is_bidirectional
    FROM local_points p CROSS JOIN route r
),
-- 3. Detect zone transitions (Gaps and Islands)
zone_steps AS (
    SELECT 
        *,
        CASE WHEN LAG(zone_id) OVER (PARTITION BY workout_id ORDER BY sort_order) = zone_id THEN 0 ELSE 1 END AS is_step
    FROM tagged_zones
),
zone_clusters AS (
    SELECT 
        *,
        SUM(is_step) OVER (PARTITION BY workout_id ORDER BY sort_order) AS cluster_id
    FROM zone_steps
),
-- 4. Summarize contiguous visits to zones
cluster_summary AS (
    SELECT 
        workout_id,
        cluster_id,
        zone_id,
        MIN(sort_order) AS first_sort,
        MAX(sort_order) AS last_sort,
        is_circular,
        is_bidirectional
    FROM zone_clusters
    GROUP BY workout_id, cluster_id, zone_id, is_circular, is_bidirectional
),
-- 5. Pair valid departure -> outside -> arrival transitions
cluster_pairs AS (
    SELECT 
        c.workout_id,
        c.first_sort AS start_sort,
        LEAD(c.first_sort, 2) OVER (PARTITION BY c.workout_id ORDER BY c.cluster_id) AS end_sort,
        c.zone_id AS start_zone,
        LEAD(c.zone_id, 1) OVER (PARTITION BY c.workout_id ORDER BY c.cluster_id) AS mid_zone,
        LEAD(c.zone_id, 2) OVER (PARTITION BY c.workout_id ORDER BY c.cluster_id) AS end_zone,
        c.is_circular,
        c.is_bidirectional
    FROM cluster_summary c
),
candidate_efforts AS (
    SELECT 
        workout_id,
        start_sort,
        end_sort
    FROM cluster_pairs
    WHERE mid_zone = 0 -- THE TRIPWIRE: Must leave both zones
      AND end_sort IS NOT NULL
      AND end_sort > start_sort + ? -- RouteSegmentMinPointInterval
      AND (
          -- Circular: Start -> Start
          (is_circular AND start_zone = 1 AND end_zone = 1)
          OR
          -- Standard: Start -> End
          (NOT is_circular AND NOT is_bidirectional AND start_zone = 1 AND end_zone = 2)
          OR
          -- Bidirectional: Start -> End OR End -> Start
          (NOT is_circular AND is_bidirectional AND (
              (start_zone = 1 AND end_zone = 2) OR 
              (start_zone = 2 AND end_zone = 1)
          ))
      )
),
-- 6. Reconstruct the geometry strictly for the isolated match
effort_tracks AS (
    SELECT 
        ce.workout_id,
        ce.start_sort,
        ce.end_sort,
        MIN(lp.time) AS start_time,
        MAX(lp.time) AS end_time,
        ST_MakeLine(ST_Transform(ST_SetSRID(lp.point, 4326), 3857) ORDER BY lp.sort_order) AS track_geom,
        ST_Length(ST_MakeLine(lp.point ORDER BY lp.sort_order)::geography) AS track_length
    FROM candidate_efforts ce
    JOIN local_points lp 
      ON lp.workout_id = ce.workout_id 
     AND lp.sort_order BETWEEN ce.start_sort AND ce.end_sort
    GROUP BY ce.workout_id, ce.start_sort, ce.end_sort
)
-- 7. Validate Distance and Shape
SELECT 
    e.workout_id,
    e.start_sort,
    e.end_sort,
    e.start_time,
    e.end_time,
    e.track_length
FROM effort_tracks e
CROSS JOIN route r
WHERE 
    e.track_length >= (r.length_m * ?) 
    AND ST_HausdorffDistance(e.track_geom, r.geom) < ? 
ORDER BY e.workout_id, e.start_time;
`

// FindCandidateWorkoutsForRouteSegment finds all workout IDs that intersect the bounding box of the route segment.
func FindCandidateWorkoutsForRouteSegment(db *gorm.DB, routeSegmentID uint64) ([]uint64, error) {
	if db == nil {
		return nil, errors.New("nil db")
	}

	var workoutIDs []uint64
	err := db.Raw(
		candidateWorkoutsForRouteSegmentQuery,
		routeSegmentID,
		RouteSegmentBoundingBoxExpansionDegrees,
	).Scan(&workoutIDs).Error
	if err != nil {
		return nil, err
	}
	return workoutIDs, nil
}

// FindRouteSegmentMatchesInBatches finds all matching workouts for a given route segment in batches of workouts
// to prevent excessive database memory consumption.
func FindRouteSegmentMatchesInBatches(db *gorm.DB, routeSegmentID uint64, batchSize int) ([]*RouteSegmentMatch, error) {
	if db == nil {
		return nil, errors.New("nil db")
	}
	if batchSize <= 0 {
		batchSize = RouteSegmentWorkoutBatchSize
	}

	workoutIDs, err := FindCandidateWorkoutsForRouteSegment(db, routeSegmentID)
	if err != nil {
		return nil, err
	}
	if len(workoutIDs) == 0 {
		return nil, nil
	}

	var allMatches []*RouteSegmentMatch
	for i := 0; i < len(workoutIDs); i += batchSize {
		end := i + batchSize
		if end > len(workoutIDs) {
			end = len(workoutIDs)
		}
		batch := workoutIDs[i:end]

		var results []matchQueryResult
		if err := db.Raw(
			matchRouteSegmentBatchedQuery,
			routeSegmentID,
			routeSegmentID,
			batch,
			RouteSegmentBoundingBoxExpansionDegrees,
			RouteSegmentZoneBufferMeters,
			RouteSegmentZoneBufferMeters,
			RouteSegmentMinPointInterval,
			RouteSegmentMinLengthFraction,
			RouteSegmentMaxHausdorffDistance,
		).Scan(&results).Error; err != nil {
			return nil, err
		}

		for _, r := range results {
			var dur time.Duration
			if !r.EndTime.IsZero() && !r.StartTime.IsZero() && r.EndTime.After(r.StartTime) {
				dur = r.EndTime.Sub(r.StartTime)
			}
			allMatches = append(allMatches, &RouteSegmentMatch{
				RouteSegmentID: routeSegmentID,
				WorkoutID:      r.WorkoutID,
				FirstID:        r.StartSort,
				LastID:         r.EndSort,
				Distance:       r.TrackLength,
				Duration:       dur,
			})
		}
	}

	return allMatches, nil
}

// FindRouteSegmentMatches finds all matching workouts for a given route segment using PostGIS in batches.
func FindRouteSegmentMatches(db *gorm.DB, routeSegmentID uint64) ([]*RouteSegmentMatch, error) {
	return FindRouteSegmentMatchesInBatches(db, routeSegmentID, RouteSegmentWorkoutBatchSize)
}

// FindRouteSegmentWorkoutMatches finds matches between a specific route segment and a specific workout.
func FindRouteSegmentWorkoutMatches(db *gorm.DB, routeSegmentID uint64, workoutID uint64) ([]*RouteSegmentMatch, error) {
	if db == nil {
		return nil, errors.New("nil db")
	}

	var results []matchQueryResult
	if err := db.Raw(
		matchRouteSegmentWorkoutQuery,
		routeSegmentID,
		routeSegmentID,
		workoutID,
		RouteSegmentBoundingBoxExpansionDegrees,
		RouteSegmentZoneBufferMeters,
		RouteSegmentZoneBufferMeters,
		RouteSegmentMinPointInterval,
		RouteSegmentMinLengthFraction,
		RouteSegmentMaxHausdorffDistance,
	).Scan(&results).Error; err != nil {
		return nil, err
	}

	matches := make([]*RouteSegmentMatch, len(results))
	for i, r := range results {
		var dur time.Duration
		if !r.EndTime.IsZero() && !r.StartTime.IsZero() && r.EndTime.After(r.StartTime) {
			dur = r.EndTime.Sub(r.StartTime)
		}
		matches[i] = &RouteSegmentMatch{
			RouteSegmentID: routeSegmentID,
			WorkoutID:      r.WorkoutID,
			FirstID:        r.StartSort,
			LastID:         r.EndSort,
			Distance:       r.TrackLength,
			Duration:       dur,
		}
	}

	return matches, nil
}

// RematchRouteSegment executes the PostGIS matching query and updates the database records for this route segment.
func RematchRouteSegment(db *gorm.DB, routeSegmentID uint64) error {
	matches, err := FindRouteSegmentMatches(db, routeSegmentID)
	if err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := replaceRouteSegmentMatches(tx, routeSegmentID, matches); err != nil {
			return err
		}
		return tx.Model(&RouteSegment{}).Where("id = ?", routeSegmentID).Update("dirty", false).Error
	})
}

// FindWorkoutRouteSegmentMatches finds all matching route segments for a given workout.
func FindWorkoutRouteSegmentMatches(db *gorm.DB, workoutID uint64) ([]*RouteSegmentMatch, error) {
	if db == nil {
		return nil, errors.New("nil db")
	}

	var segmentIDs []uint64
	err := db.Raw(`
		SELECT rs.id 
		FROM route_segments rs
		WHERE rs.points IS NOT NULL
		  AND EXISTS (
		      SELECT 1 FROM workout_records w
		      WHERE w.workout_id = ?
		        AND w.point IS NOT NULL
		        AND w.point && ST_Expand(ST_SetSRID(rs.points, 4326), ?)
		  )
		ORDER BY rs.id ASC
	`, workoutID, RouteSegmentBoundingBoxExpansionDegrees).Scan(&segmentIDs).Error
	if err != nil {
		return nil, err
	}

	var allMatches []*RouteSegmentMatch
	for _, segID := range segmentIDs {
		matches, err := FindRouteSegmentWorkoutMatches(db, segID, workoutID)
		if err != nil {
			return nil, err
		}
		allMatches = append(allMatches, matches...)
	}

	return allMatches, nil
}

// RematchWorkout executes matching for a workout and replaces its matches in the database.
func RematchWorkout(db *gorm.DB, workoutID uint64) error {
	matches, err := FindWorkoutRouteSegmentMatches(db, workoutID)
	if err != nil {
		return err
	}

	return replaceWorkoutRouteSegmentMatches(db, workoutID, matches)
}
