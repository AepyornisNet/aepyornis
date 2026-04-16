package converters

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/muktihari/fit/decoder"
	"github.com/muktihari/fit/kit/datetime"
	"github.com/muktihari/fit/kit/semicircles"
	"github.com/muktihari/fit/profile/filedef"
	"github.com/muktihari/fit/profile/typedef"
	"github.com/spf13/cast"
	"github.com/tkrajina/gpxgo/gpx"
)

func ParseFit(content []byte, filename string) ([]*model.Workout, error) {
	dec := decoder.New(bytes.NewReader(content), decoder.WithIgnoreChecksum())

	f, err := dec.Decode()
	if err != nil {
		return nil, fmt.Errorf("failed to decode FIT file: %w", err)
	}

	act := filedef.NewActivity(f.Messages...)
	if len(act.Sessions) == 0 {
		return nil, errors.New("no sessions found")
	}

	activityTime := fitActivityStartTime(act)

	gpxFile := buildGPXFromActivity(act)
	data, records := mapDataFromActivity(act, gpxFile)
	laps := parseLaps(act)
	stats := parseWorkoutStats(act)
	_, totalDistance2D, _ := model.WorkoutTotalsFromRecords(records)

	workouts := make([]*model.Workout, 0, len(act.Sessions))

	for _, session := range act.Sessions {
		startTime := firstNonZeroTime(session.StartTime.Local(), activityTime)

		moveDuration := durationFromSeconds(session.TotalTimerTimeScaled())
		elapsedDuration := durationFromSeconds(session.TotalElapsedTimeScaled())
		pauseDuration := maxDuration(elapsedDuration-moveDuration, 0)

		clonedData := cloneMapData(data)
		if clonedData == nil {
			clonedData = &model.WorkoutGeoMeta{}
		}

		workoutType, found := model.WorkoutTypeFromData(session.Sport.String())
		customType := ""
		if !found {
			customType = session.Sport.String()
		}
		workoutName := formatFitWorkoutName(session.Sport.String(), startTime)
		subType := ""
		if session.SubSport != typedef.SubSportInvalid {
			subType = session.SubSport.String()
		}

		w := &model.Workout{
			Data:            clonedData,
			Date:            startTime,
			DateEnd:         startTime.Add(elapsedDuration),
			Name:            workoutName,
			Type:            workoutType,
			SubType:         subType,
			CustomType:      customType,
			Records:         append([]model.WorkoutRecord(nil), records...),
			TotalDistance:   session.TotalDistanceScaled(),
			TotalDistance2D: totalDistance2D,
			TotalDuration:   elapsedDuration,
			PauseDuration:   pauseDuration,
		}

		if w.Data != nil {
			w.Data.WorkoutData.MergeNonZero(model.WorkoutData{
				WorkoutStats: stats,
			})
		}

		w.Laps = append([]model.WorkoutLap(nil), laps...)
		setContentAndName(w, filename, "fit", content)
		w.UpdateAverages()
		w.UpdateExtraMetrics()

		workouts = append(workouts, w)
	}

	return workouts, nil
}

//gocyclo:ignore
func parseLaps(act *filedef.Activity) []model.WorkoutLap {
	laps := make([]model.WorkoutLap, 0, len(act.Laps))
	for _, lap := range act.Laps {
		elapsed := time.Duration(0)
		if lap.TotalElapsedTime != math.MaxUint32 {
			elapsed = time.Duration(lap.TotalElapsedTimeScaled() * float64(time.Second))
		}

		timer := time.Duration(0)
		if lap.TotalTimerTime != math.MaxUint32 {
			timer = time.Duration(lap.TotalTimerTimeScaled() * float64(time.Second))
		}

		totalDistance := 0.0
		if lap.TotalDistance != math.MaxUint32 {
			totalDistance = lap.TotalDistanceScaled()
		}

		lapStart := lap.StartTime.Local()
		lapStop := lapStart
		if !lapStart.IsZero() && elapsed > 0 {
			lapStop = lapStart.Add(elapsed)
		}

		pause := maxDuration(elapsed-timer, 0)

		minElevation := 0.0
		if lap.EnhancedMinAltitude != math.MaxUint32 {
			minElevation = lap.EnhancedMinAltitudeScaled()
		} else if lap.MinAltitude != math.MaxUint16 {
			minElevation = lap.MinAltitudeScaled()
		}

		maxElevation := 0.0
		if lap.EnhancedMaxAltitude != math.MaxUint32 {
			maxElevation = lap.EnhancedMaxAltitudeScaled()
		} else if lap.MaxAltitude != math.MaxUint16 {
			maxElevation = lap.MaxAltitudeScaled()
		}

		avgSpeed := 0.0
		if lap.EnhancedAvgSpeed != math.MaxUint32 {
			avgSpeed = lap.EnhancedAvgSpeedScaled()
		} else if lap.AvgSpeed != math.MaxUint16 {
			avgSpeed = lap.AvgSpeedScaled()
		}

		maxSpeed := 0.0
		if lap.EnhancedMaxSpeed != math.MaxUint32 {
			maxSpeed = lap.EnhancedMaxSpeedScaled()
		} else if lap.MaxSpeed != math.MaxUint16 {
			maxSpeed = lap.MaxSpeedScaled()
		}

		avgCadence := 0.0
		if lap.AvgCadence != math.MaxUint8 {
			avgCadence = float64(lap.AvgCadence)
		}

		maxCadence := 0.0
		if lap.MaxCadence != math.MaxUint8 {
			maxCadence = float64(lap.MaxCadence)
		}

		avgHeartRate := 0.0
		if lap.AvgHeartRate != math.MaxUint8 {
			avgHeartRate = float64(lap.AvgHeartRate)
		}

		maxHeartRate := 0.0
		if lap.MaxHeartRate != math.MaxUint8 {
			maxHeartRate = float64(lap.MaxHeartRate)
		}

		avgPower := 0.0
		if lap.AvgPower != math.MaxUint16 {
			avgPower = float64(lap.AvgPower)
		}

		maxPower := 0.0
		if lap.MaxPower != math.MaxUint16 {
			maxPower = float64(lap.MaxPower)
		}

		totalUp := 0.0
		if lap.TotalAscent != math.MaxUint16 {
			totalUp = float64(lap.TotalAscent)
		}

		totalDown := 0.0
		if lap.TotalDescent != math.MaxUint16 {
			totalDown = float64(lap.TotalDescent)
		}

		movingDuration := elapsed - pause
		avgSpeedNoPause := avgSpeed
		if totalDistance > 0 && movingDuration > 0 {
			avgSpeedNoPause = totalDistance / movingDuration.Seconds()
		}

		laps = append(laps, model.WorkoutLap{
			Start:         lapStart,
			Stop:          lapStop,
			TotalDistance: totalDistance,
			TotalDuration: elapsed,
			PauseDuration: pause,
			WorkoutStats: model.WorkoutStats{
				MinElevation:        minElevation,
				MaxElevation:        maxElevation,
				TotalUp:             totalUp,
				TotalDown:           totalDown,
				AverageSpeed:        avgSpeed,
				AverageSpeedNoPause: avgSpeedNoPause,
				MaxSpeed:            maxSpeed,
				AverageCadence:      avgCadence,
				MaxCadence:          maxCadence,
				AverageHeartRate:    avgHeartRate,
				MaxHeartRate:        maxHeartRate,
				AveragePower:        avgPower,
				MaxPower:            maxPower,
			},
		})
	}

	return laps
}

func parseWorkoutStats(act *filedef.Activity) model.WorkoutStats {
	session := act.Sessions[0]
	stats := model.WorkoutStats{}

	if session.AvgCadence != math.MaxUint8 {
		stats.AverageCadence = float64(session.AvgCadence)
	}

	if session.MaxCadence != math.MaxUint8 {
		stats.MaxCadence = float64(session.MaxCadence)
	}

	if session.AvgHeartRate != math.MaxUint8 {
		stats.AverageHeartRate = float64(session.AvgHeartRate)
	}

	if session.MaxHeartRate != math.MaxUint8 {
		stats.MaxHeartRate = float64(session.MaxHeartRate)
	}

	if session.EnhancedAvgSpeed != math.MaxUint32 {
		stats.AverageSpeed = session.EnhancedAvgSpeedScaled()
	} else if session.AvgSpeed != math.MaxUint16 {
		stats.AverageSpeed = session.AvgSpeedScaled()
	}

	if session.MaxSpeed != math.MaxUint16 {
		stats.MaxSpeed = session.MaxSpeedScaled()
	}

	if session.EnhancedMinAltitude != math.MaxUint32 {
		stats.MinElevation = session.EnhancedMinAltitudeScaled()
	} else if session.MinAltitude != math.MaxUint16 {
		stats.MinElevation = session.MinAltitudeScaled()
	}

	if session.EnhancedMaxAltitude != math.MaxUint32 {
		stats.MaxElevation = session.EnhancedMaxAltitudeScaled()
	} else if session.MaxAltitude != math.MaxUint16 {
		stats.MaxElevation = session.MaxAltitudeScaled()
	}

	if session.AvgPower != math.MaxUint16 {
		stats.AveragePower = float64(session.AvgPower)
	}

	if session.MaxPower != math.MaxUint16 {
		stats.MaxPower = float64(session.MaxPower)
	}

	if session.TotalAscent != math.MaxUint16 {
		stats.TotalUp = float64(session.TotalAscent)
	}

	if session.TotalDescent != math.MaxUint16 {
		stats.TotalDown = float64(session.TotalDescent)
	}

	return stats
}

func durationFromSeconds(seconds float64) time.Duration {
	if seconds <= 0 {
		return 0
	}

	return time.Duration(seconds * float64(time.Second))
}

func buildGPXFromActivity(act *filedef.Activity) *gpx.GPX {
	name := formatFitWorkoutName(act.Sessions[0].Sport.String(), fitActivityStartTime(act))
	gpxFile := &gpx.GPX{
		Name:    name,
		Time:    &act.FileId.TimeCreated,
		Creator: act.FileId.Manufacturer.String(),
	}

	if len(act.Sessions) > 0 {
		s := act.Sessions[0]
		gpxFile.AppendTrack(&gpx.GPXTrack{
			Name: s.SportProfileName,
			Type: s.Sport.String(),
		})
	}

	for _, r := range act.Records {
		p := &gpx.GPXPoint{
			Timestamp: r.Timestamp,
			Point: gpx.Point{
				Latitude:  semicircles.ToDegrees(r.PositionLat),
				Longitude: semicircles.ToDegrees(r.PositionLong),
			},
		}

		if math.IsNaN(p.Latitude) || math.IsNaN(p.Longitude) {
			continue
		}

		if r.EnhancedAltitude != math.MaxUint32 {
			p.Elevation = *gpx.NewNullableFloat64(r.EnhancedAltitudeScaled())
		}

		gpxExtensionData := map[string]string{}
		if r.Cadence != math.MaxUint8 {
			gpxExtensionData["cadence"] = cast.ToString(r.Cadence)
		}

		if r.HeartRate != math.MaxUint8 {
			gpxExtensionData["heart-rate"] = cast.ToString(r.HeartRate)
		}

		if r.EnhancedRespirationRate != math.MaxUint16 {
			gpxExtensionData["respiration-rate"] = cast.ToString(r.EnhancedRespirationRateScaled())
		} else if r.RespirationRate != math.MaxUint8 {
			gpxExtensionData["respiration-rate"] = cast.ToString(r.RespirationRate)
		}

		if r.EnhancedSpeed != math.MaxUint32 {
			gpxExtensionData["speed"] = cast.ToString(r.EnhancedSpeedScaled())
		} else if r.Speed != math.MaxUint16 {
			gpxExtensionData["speed"] = cast.ToString(r.SpeedScaled())
		}

		if r.Temperature != math.MaxInt8 {
			gpxExtensionData["temperature"] = cast.ToString(r.Temperature)
		}

		if r.Power != math.MaxUint16 {
			gpxExtensionData["power"] = cast.ToString(r.Power)
		}

		for key, value := range gpxExtensionData {
			p.Extensions.Nodes = append(p.Extensions.Nodes, gpx.ExtensionNode{
				XMLName: xml.Name{Local: key}, Data: value,
			})
		}

		gpxFile.AppendPoint(p)
	}

	return gpxFile
}

// mapDataFromActivity converts a FIT activity into MapData, falling back to
// non-positional record data when coordinates are missing so charts and
// breakdowns remain available even without a map.
func mapDataFromActivity(act *filedef.Activity, gpxFile *gpx.GPX) (*model.WorkoutGeoMeta, []model.WorkoutRecord) {
	data, records := model.MapDataAndRecordsFromGPX(gpxFile)

	if data != nil && len(records) > 0 {
		return data, records
	}

	return buildMapDataWithoutPositions(act)
}

// buildMapDataWithoutPositions constructs minimal map data using FIT records
// that may lack latitude/longitude. Coordinates are left at zero to avoid
// rendering a map, while time/distance/metrics remain available for charts
// and breakdowns.
//
//nolint:gocyclo // branching covers optional FIT metrics without positions
func buildMapDataWithoutPositions(act *filedef.Activity) (*model.WorkoutGeoMeta, []model.WorkoutRecord) {
	if act == nil || len(act.Records) == 0 {
		return nil, nil
	}

	points := make([]model.WorkoutRecord, 0, len(act.Records))

	var (
		totalDistance float64
		totalDuration time.Duration
		pauseDuration time.Duration
		maxSpeed      float64
		minElevation  = math.MaxFloat64
		maxElevation  = -math.MaxFloat64
		prevTime      time.Time
		prevDistance  float64
	)

	for i, r := range act.Records {
		ts := r.Timestamp.Local()
		if ts.IsZero() {
			continue
		}

		// Distances are scaled by 100 (meters)
		dist := 0.0
		if r.Distance != math.MaxUint32 {
			dist = float64(r.Distance) / 100
		}

		deltaDist := 0.0
		if i == 0 {
			prevDistance = dist
		} else if dist >= prevDistance {
			deltaDist = dist - prevDistance
			prevDistance = dist
		}

		dt := time.Duration(0)
		if !prevTime.IsZero() {
			dt = ts.Sub(prevTime)
			if dt < 0 {
				dt = 0
			}
		}
		prevTime = ts

		totalDistance = dist
		totalDuration += dt

		speed := 0.0
		if r.EnhancedSpeed != math.MaxUint32 {
			speed = r.EnhancedSpeedScaled()
		} else if r.Speed != math.MaxUint16 {
			speed = r.SpeedScaled()
		}
		maxSpeed = math.Max(maxSpeed, speed)

		if speed*3.6 < 1.0 {
			pauseDuration += dt
		}

		elevation := math.NaN()
		if r.EnhancedAltitude != math.MaxUint32 {
			elevation = r.EnhancedAltitudeScaled()
		} else if r.Altitude != math.MaxUint16 {
			elevation = r.AltitudeScaled()
		}
		if !math.IsNaN(elevation) {
			if elevation < minElevation {
				minElevation = elevation
			}
			if elevation > maxElevation {
				maxElevation = elevation
			}
		}

		extra := model.ExtraMetrics{}
		if !math.IsNaN(elevation) {
			extra.Set("elevation", elevation)
		}
		if r.Cadence != math.MaxUint8 {
			extra.Set("cadence", float64(r.Cadence))
		}
		if r.HeartRate != math.MaxUint8 {
			extra.Set("heart-rate", float64(r.HeartRate))
		}
		if r.EnhancedRespirationRate != math.MaxUint16 {
			extra.Set("respiration-rate", float64(r.EnhancedRespirationRateScaled()))
		} else if r.RespirationRate != math.MaxUint8 {
			extra.Set("respiration-rate", float64(r.RespirationRate))
		}
		if r.Power != math.MaxUint16 {
			extra.Set("power", float64(r.Power))
		}
		if r.Temperature != math.MaxInt8 {
			extra.Set("temperature", float64(r.Temperature))
		}
		if speed > 0 {
			extra.Set("speed", speed)
		}

		elevationValue := elevation
		if math.IsNaN(elevationValue) {
			elevationValue = 0
		}

		points = append(points, model.WorkoutRecord{
			Time:          ts,
			Lat:           0,
			Lng:           0,
			Elevation:     elevationValue,
			Distance:      deltaDist,
			TotalDistance: dist,
			Duration:      dt,
			TotalDuration: totalDuration,
			ExtraMetrics:  extra,
		})
	}

	// If no points survived, bail out to avoid empty details
	if len(points) == 0 {
		return nil, nil
	}

	// Normalize elevation bounds when none are present
	if minElevation == math.MaxFloat64 {
		minElevation = 0
	}
	if maxElevation == -math.MaxFloat64 {
		maxElevation = 0
	}

	data := &model.WorkoutGeoMeta{
		Creator: act.FileId.Manufacturer.String(),
		Center:  model.MapCenter{},
		WorkoutData: model.WorkoutData{
			WorkoutStats: model.WorkoutStats{
				MinElevation:        minElevation,
				MaxElevation:        maxElevation,
				AverageSpeed:        safeDivide(totalDistance, totalDuration),
				AverageSpeedNoPause: safeDivide(totalDistance, totalDuration-pauseDuration),
				MaxSpeed:            maxSpeed,
			},
		},
	}

	data.UpdateExtraMetrics(points)
	sanitizeMapData(data)

	return data, points
}

func safeDivide(distance float64, d time.Duration) float64 {
	if d <= 0 {
		return 0
	}
	return distance / d.Seconds()
}

func sanitizeMapData(data *model.WorkoutGeoMeta) {
	if data == nil {
		return
	}

	if math.IsNaN(data.MinElevation) {
		data.MinElevation = 0
	}

	if math.IsNaN(data.MaxElevation) {
		data.MaxElevation = 0
	}

	if math.IsNaN(data.TotalDown) {
		data.TotalDown = 0
	}

	if math.IsNaN(data.TotalUp) {
		data.TotalUp = 0
	}
}

func cloneMapData(src *model.WorkoutGeoMeta) *model.WorkoutGeoMeta {
	if src == nil {
		return &model.WorkoutGeoMeta{}
	}

	clone := *src

	return &clone
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}

	return b
}

func fitActivityStartTime(act *filedef.Activity) time.Time {
	if act == nil {
		return time.Time{}
	}

	if t := act.Activity.LocalTimestamp.Local(); fitTimeIsValid(t) {
		return t
	}

	for _, s := range act.Sessions {
		if t := s.StartTime.Local(); fitTimeIsValid(t) {
			return t
		}
	}

	for _, l := range act.Laps {
		if t := l.StartTime.Local(); fitTimeIsValid(t) {
			return t
		}
	}

	for _, r := range act.Records {
		if t := r.Timestamp.Local(); fitTimeIsValid(t) {
			return t
		}
	}

	return act.FileId.TimeCreated.Local()
}

// fitTimeIsValid reports whether t is a plausible FIT timestamp.
// The FIT library decodes an unset uint32(0) field as the FIT epoch
// (1989-12-31 00:00:00 UTC) rather than Go's zero time, so we must
// reject both Go's zero time and the FIT epoch itself.
func fitTimeIsValid(t time.Time) bool {
	return !t.IsZero() && t.After(datetime.Epoch())
}

func firstNonZeroTime(candidates ...time.Time) time.Time {
	for _, t := range candidates {
		if fitTimeIsValid(t) {
			return t
		}
	}

	return time.Time{}
}

func formatFitWorkoutName(sport string, at time.Time) string {
	if sport == "" {
		sport = "workout"
	}

	if !fitTimeIsValid(at) {
		return sport
	}

	return sport + " - " + at.Format(time.DateTime)
}
