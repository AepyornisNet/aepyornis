package converters

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
	"github.com/muktihari/fit/decoder"
	"github.com/muktihari/fit/kit/semicircles"
	"github.com/muktihari/fit/profile/filedef"
	"github.com/spf13/cast"
	"github.com/tkrajina/gpxgo/gpx"
)

func ParseFit(content []byte, filename string) ([]*database.Workout, error) {
	dec := decoder.New(bytes.NewReader(content), decoder.WithIgnoreChecksum())

	f, err := dec.Decode()
	if err != nil {
		return nil, fmt.Errorf("failed to decode FIT file: %w", err)
	}

	act := filedef.NewActivity(f.Messages...)
	if len(act.Sessions) == 0 {
		return nil, errors.New("no sessions found")
	}

	activityTime := act.Activity.LocalTimestamp
	if activityTime.IsZero() {
		activityTime = act.Sessions[0].StartTime.Local()
	}

	if activityTime.IsZero() {
		activityTime = act.FileId.TimeCreated.Local()
	}

	gpxFile := buildGPXFromActivity(act)
	data := database.MapDataFromGPX(gpxFile)
	laps := parseLaps(act)
	stats := parseWorkoutStats(act)

	workouts := make([]*database.Workout, 0, len(act.Sessions))

	for _, session := range act.Sessions {
		startTime := session.StartTime.Local()
		if startTime.IsZero() {
			startTime = activityTime
		}

		moveDuration := durationFromSeconds(session.TotalTimerTimeScaled())
		elapsedDuration := durationFromSeconds(session.TotalElapsedTimeScaled())
		pauseDuration := max(elapsedDuration-moveDuration, 0)

		w := &database.Workout{
			Data: cloneMapData(data),
			Date: startTime,
		}

		if w.Data != nil {
			w.Data.WorkoutData.MergeNonZero(database.WorkoutData{
				Name:          session.Sport.String() + " - " + startTime.Format(time.DateTime),
				Type:          session.Sport.String(),
				SubType:       session.SubSport.String(),
				Start:         startTime,
				Stop:          startTime.Add(elapsedDuration),
				TotalDistance: session.TotalDistanceScaled(),
				TotalDuration: elapsedDuration,
				PauseDuration: pauseDuration,
				WorkoutStats:  stats,
				Laps:          laps,
			})
		}

		w.Name = w.Data.WorkoutData.Name
		setContentAndName(w, filename, "fit", content)
		w.UpdateAverages()
		w.UpdateExtraMetrics()

		workouts = append(workouts, w)
	}

	return workouts, nil
}

//gocyclo:ignore
func parseLaps(act *filedef.Activity) []database.WorkoutLap {
	laps := make([]database.WorkoutLap, 0, len(act.Laps))
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

		pause := max(elapsed-timer, 0)

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

		laps = append(laps, database.WorkoutLap{
			Start:         lapStart,
			Stop:          lapStop,
			TotalDistance: totalDistance,
			TotalDuration: elapsed,
			PauseDuration: pause,
			WorkoutStats: database.WorkoutStats{
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

func parseWorkoutStats(act *filedef.Activity) database.WorkoutStats {
	session := act.Sessions[0]
	stats := database.WorkoutStats{}

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
	name := act.Sessions[0].Sport.String() + " - " + act.Activity.LocalTimestamp.Format(time.DateTime)
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

func cloneMapData(src *database.MapData) *database.MapData {
	if src == nil {
		return &database.MapData{}
	}

	clone := *src
	if src.Details != nil {
		clone.Details = &database.MapDataDetails{Points: src.Details.Points}
	}

	return &clone
}

func max(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}

	return b
}
