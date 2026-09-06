package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// MaxDeltaMeter is the maximum distance in meters that a point can be away from
// the route segment
const MaxDeltaMeter = 20.0

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

const matchRouteSegmentQuery = `
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
        w.workout_id, w.sort_order, w.time, w.point,
        ST_Transform(ST_SetSRID(w.point, 4326), 3857) AS pt
    FROM workout_records w
    JOIN route_segments rs ON rs.id = ?
    WHERE w.point IS NOT NULL 
      AND w.point && ST_Expand(ST_SetSRID(rs.points, 4326), 0.01)
),
-- 2. Tag zone entries (25 true meters on WGS84 ellipsoid)
tagged_zones AS (
    SELECT 
        p.workout_id, p.sort_order, p.time, p.pt, p.point,
        ST_DWithin(p.point::geography, r.start_geom::geography, 25.0) AS in_start,
        ST_DWithin(p.point::geography, r.end_geom::geography, 25.0) AS in_end,
        r.is_circular,
        r.is_bidirectional
    FROM local_points p CROSS JOIN route r
),
-- 3. The Core Logic: Pairing valid entries and exits using a tripwire
valid_segments AS (
    SELECT 
        s.workout_id,
        s.sort_order AS start_sort,
        (
            SELECT MIN(e.sort_order) 
            FROM tagged_zones e 
            WHERE e.workout_id = s.workout_id 
              AND e.sort_order > s.sort_order + 5
              AND (
                  -- Standard: Start -> End
                  (NOT e.is_circular AND NOT e.is_bidirectional AND s.in_start AND e.in_end)
                  OR
                  -- Bidirectional: Start -> End OR End -> Start
                  (NOT e.is_circular AND e.is_bidirectional AND ((s.in_start AND e.in_end) OR (s.in_end AND e.in_start)))
                  OR 
                  -- Circular: Start -> Start (Direction doesn't matter)
                  (e.is_circular AND s.in_start AND e.in_start)
              )
              -- THE TRIPWIRE: They must have left both zones at least once during this interval
              AND EXISTS (
                  SELECT 1 FROM tagged_zones m 
                  WHERE m.workout_id = s.workout_id 
                    AND m.sort_order > s.sort_order 
                    AND m.sort_order < e.sort_order
                    AND NOT m.in_start AND NOT m.in_end
              )
        ) AS end_sort
    FROM tagged_zones s
    WHERE s.in_start OR (s.is_bidirectional AND s.in_end)
),
-- 4. Deduplicate multiple stationary points grouping to the same exit
distinct_efforts AS (
    SELECT 
        workout_id,
        MIN(start_sort) AS start_sort,
        end_sort
    FROM valid_segments
    WHERE end_sort IS NOT NULL
    GROUP BY workout_id, end_sort
),
-- 5. Reconstruct the geometry strictly for the isolated match
effort_tracks AS (
    SELECT 
        de.workout_id,
        de.start_sort,
        de.end_sort,
        MIN(lp.time) AS start_time,
        MAX(lp.time) AS end_time,
        ST_MakeLine(lp.pt ORDER BY lp.sort_order) AS track_geom,
        ST_Length(ST_MakeLine(lp.point ORDER BY lp.sort_order)::geography) AS track_length
    FROM distinct_efforts de
    JOIN local_points lp 
      ON lp.workout_id = de.workout_id 
     AND lp.sort_order BETWEEN de.start_sort AND de.end_sort
    GROUP BY de.workout_id, de.start_sort, de.end_sort
)
-- 6. Validate Distance and Shape
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
    e.track_length >= (r.length_m * 0.85) 
    -- Hausdorff ignores direction, making it universally valid for bidirectional and circular shape checks
    AND ST_HausdorffDistance(e.track_geom, r.geom) < 50.0 
ORDER BY e.workout_id, e.start_time;
`

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
        w.workout_id, w.sort_order, w.time, w.point,
        ST_Transform(ST_SetSRID(w.point, 4326), 3857) AS pt
    FROM workout_records w
    JOIN route_segments rs ON rs.id = ?
    WHERE w.workout_id = ?
      AND w.point IS NOT NULL 
      AND w.point && ST_Expand(ST_SetSRID(rs.points, 4326), 0.01)
),
-- 2. Tag zone entries (25 true meters on WGS84 ellipsoid)
tagged_zones AS (
    SELECT 
        p.workout_id, p.sort_order, p.time, p.pt, p.point,
        ST_DWithin(p.point::geography, r.start_geom::geography, 25.0) AS in_start,
        ST_DWithin(p.point::geography, r.end_geom::geography, 25.0) AS in_end,
        r.is_circular,
        r.is_bidirectional
    FROM local_points p CROSS JOIN route r
),
-- 3. The Core Logic: Pairing valid entries and exits using a tripwire
valid_segments AS (
    SELECT 
        s.workout_id,
        s.sort_order AS start_sort,
        (
            SELECT MIN(e.sort_order) 
            FROM tagged_zones e 
            WHERE e.workout_id = s.workout_id 
              AND e.sort_order > s.sort_order + 5
              AND (
                  -- Standard: Start -> End
                  (NOT e.is_circular AND NOT e.is_bidirectional AND s.in_start AND e.in_end)
                  OR
                  -- Bidirectional: Start -> End OR End -> Start
                  (NOT e.is_circular AND e.is_bidirectional AND ((s.in_start AND e.in_end) OR (s.in_end AND e.in_start)))
                  OR 
                  -- Circular: Start -> Start (Direction doesn't matter)
                  (e.is_circular AND s.in_start AND e.in_start)
              )
              -- THE TRIPWIRE: They must have left both zones at least once during this interval
              AND EXISTS (
                  SELECT 1 FROM tagged_zones m 
                  WHERE m.workout_id = s.workout_id 
                    AND m.sort_order > s.sort_order 
                    AND m.sort_order < e.sort_order
                    AND NOT m.in_start AND NOT m.in_end
              )
        ) AS end_sort
    FROM tagged_zones s
    WHERE s.in_start OR (s.is_bidirectional AND s.in_end)
),
-- 4. Deduplicate multiple stationary points grouping to the same exit
distinct_efforts AS (
    SELECT 
        workout_id,
        MIN(start_sort) AS start_sort,
        end_sort
    FROM valid_segments
    WHERE end_sort IS NOT NULL
    GROUP BY workout_id, end_sort
),
-- 5. Reconstruct the geometry strictly for the isolated match
effort_tracks AS (
    SELECT 
        de.workout_id,
        de.start_sort,
        de.end_sort,
        MIN(lp.time) AS start_time,
        MAX(lp.time) AS end_time,
        ST_MakeLine(lp.pt ORDER BY lp.sort_order) AS track_geom,
        ST_Length(ST_MakeLine(lp.point ORDER BY lp.sort_order)::geography) AS track_length
    FROM distinct_efforts de
    JOIN local_points lp 
      ON lp.workout_id = de.workout_id 
     AND lp.sort_order BETWEEN de.start_sort AND de.end_sort
    GROUP BY de.workout_id, de.start_sort, de.end_sort
)
-- 6. Validate Distance and Shape
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
    e.track_length >= (r.length_m * 0.85) 
    -- Hausdorff ignores direction, making it universally valid for bidirectional and circular shape checks
    AND ST_HausdorffDistance(e.track_geom, r.geom) < 50.0 
ORDER BY e.workout_id, e.start_time;
`

// FindRouteSegmentMatches finds all matching workouts for a given route segment using PostGIS.
func FindRouteSegmentMatches(db *gorm.DB, routeSegmentID uint64) ([]*RouteSegmentMatch, error) {
	if db == nil {
		return nil, errors.New("nil db")
	}

	var results []matchQueryResult
	if err := db.Raw(matchRouteSegmentQuery, routeSegmentID, routeSegmentID).Scan(&results).Error; err != nil {
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

// FindRouteSegmentWorkoutMatches finds matches between a specific route segment and a specific workout.
func FindRouteSegmentWorkoutMatches(db *gorm.DB, routeSegmentID uint64, workoutID uint64) ([]*RouteSegmentMatch, error) {
	if db == nil {
		return nil, errors.New("nil db")
	}

	var results []matchQueryResult
	if err := db.Raw(matchRouteSegmentWorkoutQuery, routeSegmentID, routeSegmentID, workoutID).Scan(&results).Error; err != nil {
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
		        AND w.point && ST_Expand(ST_SetSRID(rs.points, 4326), 0.01)
		  )
		ORDER BY rs.id ASC
	`, workoutID).Scan(&segmentIDs).Error
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
