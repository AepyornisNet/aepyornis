package converters

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/muktihari/fit/decoder"
	"github.com/muktihari/fit/kit/datetime"
	"github.com/muktihari/fit/kit/semicircles"
	"github.com/muktihari/fit/profile/filedef"
	"github.com/muktihari/fit/profile/mesgdef"
	"github.com/muktihari/fit/profile/typedef"
	"github.com/restayway/gogis"
	"gorm.io/datatypes"
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

	var (
		data    *model.WorkoutGeoMeta
		records []model.WorkoutRecord
		events  []model.WorkoutEvent
		laps    []model.WorkoutLap
		stats   model.WorkoutStats
	)

	if isSwimActivity(act) {
		poolLength := getPoolLength(act)
		lengths := extractSwimLengths(act, poolLength)
		data, records = mapDataFromSwimActivity(act, lengths, poolLength)
		events = parseWorkoutEvents(act, lengths)
		laps = parseFitSwimLaps(act, lengths, poolLength)
		stats = parseWorkoutStats(act, lengths)
	} else {
		data, records = mapDataFromActivity(act)
		events = parseWorkoutEvents(act, nil)
		laps = parseLaps(act)
		stats = parseWorkoutStats(act, nil)
	}

	_, totalDistance2D, _ := model.WorkoutTotalsFromRecords(records)

	workouts := make([]*model.Workout, 0, len(act.Sessions))

	for _, session := range act.Sessions {
		startTime := firstNonZeroTime(session.StartTime.Local(), activityTime)

		elapsedDuration, _, pauseDuration := deriveFitSessionDurations(
			session.TotalElapsedTime,
			session.TotalElapsedTimeScaled(),
			session.TotalTimerTime,
			session.TotalTimerTimeScaled(),
			laps,
			records,
		)

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

		totalDistance := 0.0
		if !math.IsNaN(session.TotalDistanceScaled()) && session.TotalDistanceScaled() > 0 {
			totalDistance = session.TotalDistanceScaled()
		} else if len(records) > 0 {
			totalDistance = records[len(records)-1].TotalDistance
		}

		if totalDistance2D == 0 && totalDistance > 0 {
			totalDistance2D = totalDistance
		}

		w := &model.Workout{
			Data:            clonedData,
			Stats:           &stats,
			Date:            startTime,
			DateEnd:         startTime.Add(elapsedDuration),
			Name:            workoutName,
			Creator:         act.FileId.Manufacturer.String(),
			Type:            workoutType,
			SubType:         subType,
			CustomType:      customType,
			Records:         append([]model.WorkoutRecord(nil), records...),
			Events:          append([]model.WorkoutEvent(nil), events...),
			TotalDistance:   totalDistance,
			TotalDistance2D: totalDistance2D,
			TotalDuration:   elapsedDuration,
			PauseDuration:   pauseDuration,
		}

		w.Laps = append([]model.WorkoutLap(nil), laps...)
		setContentAndName(w, filename, "fit", content)

		w.ProcessRawRecords()

		workouts = append(workouts, w)
	}

	return workouts, nil
}

func parseWorkoutEvents(act *filedef.Activity, lengths []fitSwimLength) []model.WorkoutEvent {
	var events []model.WorkoutEvent

	if act != nil && len(act.Events) > 0 {
		events = make([]model.WorkoutEvent, 0, len(act.Events)+len(lengths)*2)
		for _, e := range act.Events {
			if e == nil {
				continue
			}

			ts := e.Timestamp.Local()
			if !fitTimeIsValid(ts) {
				continue
			}

			events = append(events, model.WorkoutEvent{
				Timestamp:      ts,
				StartTimestamp: e.StartTimestamp.Local(),
				Event:          e.Event.String(),
				EventType:      e.EventType.String(),
				EventGroup:     e.EventGroup,
				Payload:        buildFitEventPayload(e),
			})
		}
	}

	for _, l := range lengths {
		if !l.isActive && l.elapsed > 0 {
			events = append(events,
				model.WorkoutEvent{
					Timestamp: l.start,
					Event:     "timer",
					EventType: "stop_all",
				},
				model.WorkoutEvent{
					Timestamp: l.end,
					Event:     "timer",
					EventType: "start",
				},
			)
		}
	}

	return events
}

func buildFitEventPayload(e *mesgdef.Event) datatypes.JSON {
	if e == nil {
		return nil
	}

	event := e.Event.String()
	switch event {
	case "timer":
		triggerType := typedef.TimerTrigger(e.Data)
		if triggerType == typedef.TimerTriggerInvalid {
			return nil
		}

		return mustJSONPayload(struct {
			Trigger string `json:"trigger"`
		}{
			Trigger: triggerType.String(),
		})
	case "front_gear_change":
		return mustJSONPayload(struct {
			FrontGearNum uint8 `json:"front_gear_num"`
			FrontGear    uint8 `json:"front_gear"`
		}{
			FrontGearNum: e.FrontGearNum,
			FrontGear:    e.FrontGear,
		})
	case "rear_gear_change":
		return mustJSONPayload(struct {
			RearGearNum uint8 `json:"rear_gear_num"`
			RearGear    uint8 `json:"rear_gear"`
		}{
			RearGearNum: e.RearGearNum,
			RearGear:    e.RearGear,
		})
	default:
		return nil
	}
}

func mustJSONPayload(v any) datatypes.JSON {
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}

	return datatypes.JSON(b)
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
			Stats: &model.WorkoutStats{
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

func parseWorkoutStats(act *filedef.Activity, lengths []fitSwimLength) model.WorkoutStats {
	if act == nil || len(act.Sessions) == 0 {
		return model.WorkoutStats{}
	}

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

	if len(lengths) > 0 {
		sumSpeed := 0.0
		sumCadence := 0.0
		activeCount := 0
		maxCadence := stats.MaxCadence
		maxSpeed := stats.MaxSpeed

		for _, l := range lengths {
			if l.isActive {
				activeCount++
				sumSpeed += l.speed
				sumCadence += l.cadence
				maxCadence = max(maxCadence, l.cadence)
				maxSpeed = max(maxSpeed, l.speed)
			}
		}

		if stats.AverageCadence == 0 && activeCount > 0 {
			stats.AverageCadence = sumCadence / float64(activeCount)
		}
		if stats.MaxCadence == 0 {
			stats.MaxCadence = maxCadence
		}
		if (stats.AverageSpeed == 0 || math.IsNaN(stats.AverageSpeed)) && activeCount > 0 {
			if session.TotalElapsedTimeScaled() > 0 && session.TotalDistanceScaled() > 0 {
				stats.AverageSpeed = session.TotalDistanceScaled() / session.TotalElapsedTimeScaled()
			} else {
				stats.AverageSpeed = sumSpeed / float64(activeCount)
			}
		}
		if stats.MaxSpeed == 0 || math.IsNaN(stats.MaxSpeed) {
			stats.MaxSpeed = maxSpeed
		}
	}

	return stats
}

func durationFromSeconds(seconds float64) time.Duration {
	if seconds <= 0 {
		return 0
	}

	return time.Duration(seconds * float64(time.Second))
}

func durationFromFITUint32(raw uint32, scaled float64) time.Duration {
	if raw == math.MaxUint32 {
		return 0
	}

	return durationFromSeconds(scaled)
}

func sumLapElapsedDuration(laps []model.WorkoutLap) time.Duration {
	total := time.Duration(0)
	for _, lap := range laps {
		total += lap.TotalDuration
	}

	return total
}

func sumLapMovingDuration(laps []model.WorkoutLap) time.Duration {
	total := time.Duration(0)
	for _, lap := range laps {
		total += maxDuration(lap.TotalDuration-lap.PauseDuration, 0)
	}

	return total
}

func movingDurationFromRecords(records []model.WorkoutRecord) time.Duration {
	if len(records) < 2 {
		return 0
	}

	stats, ok := model.StatsForRange(records, 0, len(records)-1)
	if !ok {
		return 0
	}

	return stats.MovingDuration
}

func elapsedDurationFromRecords(records []model.WorkoutRecord) time.Duration {
	_, _, duration := model.WorkoutTotalsFromRecords(records)

	return duration
}

func deriveFitSessionDurations(
	totalElapsedRaw uint32,
	totalElapsedScaled float64,
	totalTimerRaw uint32,
	totalTimerScaled float64,
	laps []model.WorkoutLap,
	records []model.WorkoutRecord,
) (time.Duration, time.Duration, time.Duration) {
	elapsed := durationFromFITUint32(totalElapsedRaw, totalElapsedScaled)
	if elapsed == 0 {
		elapsed = sumLapElapsedDuration(laps)
	}
	if elapsed == 0 {
		elapsed = elapsedDurationFromRecords(records)
	}

	moving := durationFromFITUint32(totalTimerRaw, totalTimerScaled)
	if moving == 0 {
		moving = sumLapMovingDuration(laps)
	}
	if moving == 0 {
		moving = movingDurationFromRecords(records)
	}

	if elapsed > 0 && moving > elapsed {
		moving = elapsed
	}

	pause := maxDuration(elapsed-moving, 0)

	return elapsed, moving, pause
}

func isSwimActivity(act *filedef.Activity) bool {
	if act == nil {
		return false
	}

	if len(act.Lengths) > 0 {
		return true
	}

	for _, s := range act.Sessions {
		if s.Sport == typedef.SportSwimming {
			return true
		}
	}

	return false
}

func getPoolLength(act *filedef.Activity) float64 {
	if act == nil {
		return 25.0
	}

	for _, s := range act.Sessions {
		if !math.IsNaN(s.PoolLengthScaled()) && s.PoolLengthScaled() > 0 {
			return s.PoolLengthScaled()
		}
	}

	for _, l := range act.Lengths {
		if l.LengthType == typedef.LengthTypeActive {
			sp := l.AvgSpeedScaled()
			tm := l.TotalTimerTimeScaled()
			if !math.IsNaN(sp) && sp > 0 && !math.IsNaN(tm) && tm > 0 {
				calcDist := math.Round(sp * tm)
				if calcDist > 0 {
					return calcDist
				}
			}
		}
	}

	return 25.0
}

type fitSwimLength struct {
	start    time.Time
	end      time.Time
	elapsed  float64
	timer    float64
	speed    float64
	cadence  float64
	strokes  uint16
	stroke   typedef.SwimStroke
	isActive bool
	cumDist  float64
}

func extractSwimLengths(act *filedef.Activity, poolLength float64) []fitSwimLength {
	if act == nil || len(act.Lengths) == 0 {
		return nil
	}

	lengths := make([]fitSwimLength, 0, len(act.Lengths))
	cumDist := 0.0

	for _, l := range act.Lengths {
		start := l.StartTime.Local()
		elapsed := l.TotalElapsedTimeScaled()
		if math.IsNaN(elapsed) || elapsed < 0 {
			elapsed = 0
		}
		timer := l.TotalTimerTimeScaled()
		if math.IsNaN(timer) || timer < 0 {
			timer = elapsed
		}

		isActive := l.LengthType == typedef.LengthTypeActive
		speed := 0.0
		cadence := 0.0
		strokes := l.TotalStrokes

		if isActive {
			cumDist += poolLength

			if l.AvgSpeed != math.MaxUint16 && !math.IsNaN(l.AvgSpeedScaled()) && l.AvgSpeedScaled() > 0 {
				speed = l.AvgSpeedScaled()
			} else if timer > 0 {
				speed = poolLength / timer
			}

			if l.AvgSwimmingCadence != math.MaxUint8 && l.AvgSwimmingCadence > 0 {
				cadence = float64(l.AvgSwimmingCadence)
			} else if strokes != math.MaxUint16 && timer > 0 {
				cadence = (float64(strokes) / timer) * 60.0
			}
		}

		end := start.Add(time.Duration(elapsed * float64(time.Second)))
		if end.Before(start) {
			end = start
		}

		lengths = append(lengths, fitSwimLength{
			start:    start,
			end:      end,
			elapsed:  elapsed,
			timer:    timer,
			speed:    speed,
			cadence:  cadence,
			strokes:  strokes,
			stroke:   l.SwimStroke,
			isActive: isActive,
			cumDist:  cumDist,
		})
	}

	return lengths
}

// mapDataFromSwimActivity maps swim records and lengths into WorkoutRecords with distance,
// speed, cadence and pause states.
//
//nolint:gocyclo
func mapDataFromSwimActivity(act *filedef.Activity, lengths []fitSwimLength, poolLength float64) (*model.WorkoutGeoMeta, []model.WorkoutRecord) {
	if act == nil || len(lengths) == 0 {
		return mapDataFromActivity(act)
	}

	if len(act.Records) == 0 {
		return synthesizeRecordsFromSwimLengths(lengths, poolLength)
	}

	points := make([]model.WorkoutRecord, 0, len(act.Records))
	var (
		totalDuration   time.Duration
		prevDist        float64
		lenIdx          int
		priorActiveDist float64
	)

	for i, r := range act.Records {
		ts := r.Timestamp.Local()
		if ts.IsZero() {
			continue
		}

		for lenIdx+1 < len(lengths) && ts.After(lengths[lenIdx].end) {
			if lengths[lenIdx].isActive {
				priorActiveDist += poolLength
			}
			lenIdx++
		}

		currLen := lengths[lenIdx]
		totalD := priorActiveDist
		sp := 0.0
		cad := 0.0
		isPause := false

		if ts.Before(currLen.start) {
			isPause = true
		} else if ts.After(currLen.end) {
			if currLen.isActive {
				totalD += poolLength
			}
			isPause = true
		} else if currLen.isActive {
			frac := 1.0
			if currLen.timer > 0 {
				frac = math.Min(math.Max(ts.Sub(currLen.start).Seconds()/currLen.timer, 0.0), 1.0)
			}
			totalD = priorActiveDist + frac*poolLength
			sp = currLen.speed
			cad = currLen.cadence
			isPause = false
		} else {
			isPause = true
		}

		deltaD := 0.0
		if i == 0 {
			prevDist = totalD
		} else if totalD >= prevDist {
			deltaD = totalD - prevDist
			prevDist = totalD
		}

		dt := time.Duration(0)
		if i+1 < len(act.Records) {
			dt = max(act.Records[i+1].Timestamp.Sub(ts), 0)
		}
		totalDuration += dt

		elevation := math.NaN()
		if r.EnhancedAltitude != math.MaxUint32 {
			elevation = r.EnhancedAltitudeScaled()
		} else if r.Altitude != math.MaxUint16 {
			elevation = r.AltitudeScaled()
		}

		extra := model.ExtraMetrics{}
		if !math.IsNaN(elevation) {
			extra.Set("elevation", elevation)
		}
		if r.HeartRate != math.MaxUint8 {
			extra.Set("heart-rate", float64(r.HeartRate))
		}
		if sp > 0 {
			extra.Set("speed", sp)
		}
		if cad > 0 {
			extra.Set("cadence", cad)
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

		elevationValue := elevation
		if math.IsNaN(elevationValue) {
			elevationValue = 0
		}

		lat := semicircles.ToDegrees(r.PositionLat)
		lng := semicircles.ToDegrees(r.PositionLong)
		var point *gogis.Point
		if !math.IsNaN(lat) && !math.IsNaN(lng) && (lat != 0 || lng != 0) {
			point = &gogis.Point{Lat: lat, Lng: lng}
		}

		points = append(points, model.WorkoutRecord{
			Time:            ts,
			Point:           point,
			Elevation:       elevationValue,
			Distance:        deltaD,
			Distance2D:      deltaD,
			TotalDistance:   totalD,
			TotalDistance2D: totalD,
			Duration:        dt,
			TotalDuration:   totalDuration,
			ExtraMetrics:    extra,
			Pause:           sql.NullBool{Valid: true, Bool: isPause},
		})
	}

	if len(points) == 0 {
		return nil, nil
	}

	data := &model.WorkoutGeoMeta{Center: model.MapCenter{}}
	data.UpdateExtraMetrics(points)

	return data, points
}

// synthesizeRecordsFromSwimLengths synthesizes workout records from swim lengths when
// no 1Hz records are present in the FIT file.
//
//nolint:gocyclo
func synthesizeRecordsFromSwimLengths(lengths []fitSwimLength, poolLength float64) (*model.WorkoutGeoMeta, []model.WorkoutRecord) {
	if len(lengths) == 0 {
		return nil, nil
	}

	points := make([]model.WorkoutRecord, 0, len(lengths)*10)
	var (
		totalDuration   time.Duration
		priorActiveDist float64
		prevDist        float64
	)

	for _, l := range lengths {
		if l.elapsed <= 0 {
			continue
		}

		seconds := int(math.Ceil(l.elapsed))
		if seconds < 1 {
			seconds = 1
		}

		for s := 0; s <= seconds; s++ {
			tOffset := float64(s)
			if tOffset > l.elapsed {
				tOffset = l.elapsed
			}

			ts := l.start.Add(time.Duration(tOffset * float64(time.Second)))
			if ts.After(l.end) {
				ts = l.end
			}

			// Avoid duplicate timestamp at boundary if identical to previous point
			if len(points) > 0 && points[len(points)-1].Time.Equal(ts) {
				continue
			}

			totalD := priorActiveDist
			sp := 0.0
			cad := 0.0
			isPause := false

			if l.isActive {
				frac := 1.0
				if l.timer > 0 {
					frac = math.Min(math.Max(tOffset/l.timer, 0.0), 1.0)
				}
				totalD = priorActiveDist + frac*poolLength
				sp = l.speed
				cad = l.cadence
			} else {
				isPause = true
			}

			deltaD := 0.0
			if len(points) == 0 {
				prevDist = totalD
			} else if totalD >= prevDist {
				deltaD = totalD - prevDist
				prevDist = totalD
			}

			dt := time.Duration(0)
			if len(points) > 0 {
				dt = ts.Sub(points[len(points)-1].Time)
				points[len(points)-1].Duration = dt
			}
			totalDuration += dt

			extra := model.ExtraMetrics{}
			if sp > 0 {
				extra.Set("speed", sp)
			}
			if cad > 0 {
				extra.Set("cadence", cad)
			}

			points = append(points, model.WorkoutRecord{
				Time:            ts,
				Distance:        deltaD,
				Distance2D:      deltaD,
				TotalDistance:   totalD,
				TotalDistance2D: totalD,
				Duration:        0,
				TotalDuration:   totalDuration,
				ExtraMetrics:    extra,
				Pause:           sql.NullBool{Valid: true, Bool: isPause},
			})
		}

		if l.isActive {
			priorActiveDist += poolLength
		}
	}

	if len(points) == 0 {
		return nil, nil
	}

	data := &model.WorkoutGeoMeta{Center: model.MapCenter{}}
	data.UpdateExtraMetrics(points)

	return data, points
}

// parseFitSwimLaps generates workout laps for swimming workouts.
// When session has only 1 global lap, it groups active lengths into intervals separated by rest lengths.
func parseFitSwimLaps(act *filedef.Activity, lengths []fitSwimLength, poolLength float64) []model.WorkoutLap {
	if len(act.Laps) > 1 {
		return parseLaps(act)
	}

	if len(lengths) == 0 {
		return parseLaps(act)
	}

	var (
		laps       []model.WorkoutLap
		currentSet []fitSwimLength
	)

	flushSet := func() {
		if len(currentSet) == 0 {
			return
		}

		firstLen := currentSet[0]
		lastLen := currentSet[len(currentSet)-1]
		setDist := float64(len(currentSet)) * poolLength
		setDuration := lastLen.end.Sub(firstLen.start)
		if setDuration < 0 {
			setDuration = 0
		}

		timerDuration := time.Duration(0)
		sumSpeed := 0.0
		sumCadence := 0.0
		maxCadence := 0.0
		maxSpeed := 0.0

		for _, l := range currentSet {
			timerDuration += time.Duration(l.timer * float64(time.Second))
			sumSpeed += l.speed
			sumCadence += l.cadence
			maxCadence = max(maxCadence, l.cadence)
			maxSpeed = max(maxSpeed, l.speed)
		}

		avgSpeed := 0.0
		if len(currentSet) > 0 {
			avgSpeed = sumSpeed / float64(len(currentSet))
		}
		if timerDuration.Seconds() > 0 {
			avgSpeed = setDist / timerDuration.Seconds()
		}

		avgCadence := 0.0
		if len(currentSet) > 0 {
			avgCadence = sumCadence / float64(len(currentSet))
		}

		pauseDuration := maxDuration(setDuration-timerDuration, 0)

		laps = append(laps, model.WorkoutLap{
			Start:         firstLen.start,
			Stop:          lastLen.end,
			TotalDistance: setDist,
			TotalDuration: setDuration,
			PauseDuration: pauseDuration,
			Stats: &model.WorkoutStats{
				AverageSpeed:        avgSpeed,
				AverageSpeedNoPause: avgSpeed,
				MaxSpeed:            maxSpeed,
				AverageCadence:      avgCadence,
				MaxCadence:          maxCadence,
			},
		})

		currentSet = nil
	}

	for _, l := range lengths {
		if l.isActive {
			currentSet = append(currentSet, l)
		} else {
			flushSet()
		}
	}
	flushSet()

	if len(laps) == 0 {
		return parseLaps(act)
	}

	return laps
}

// mapDataFromActivity converts a FIT activity into MapData, falling back to
// non-positional record data when coordinates are missing so charts and
// breakdowns remain available even without a map.
//
//nolint:gocyclo
func mapDataFromActivity(act *filedef.Activity) (*model.WorkoutGeoMeta, []model.WorkoutRecord) {
	if act == nil || len(act.Records) == 0 {
		return nil, nil
	}

	points := make([]model.WorkoutRecord, 0, len(act.Records))

	var (
		totalDuration time.Duration
		prevDistance  float64
	)

	for i, r := range act.Records {
		ts := r.Timestamp.Local()
		if ts.IsZero() {
			continue
		}

		dist := 0.0
		if r.Distance != math.MaxUint32 {
			dist = r.DistanceScaled()
		}

		deltaDist := 0.0
		if i == 0 {
			prevDistance = dist
		} else if dist >= prevDistance {
			deltaDist = dist - prevDistance
			prevDistance = dist
		}

		dt := time.Duration(0)
		if i+1 < len(act.Records) {
			dt = max(act.Records[i+1].Timestamp.Sub(ts), 0)
		}

		totalDuration += dt

		elevation := math.NaN()
		if r.EnhancedAltitude != math.MaxUint32 {
			elevation = r.EnhancedAltitudeScaled()
		} else if r.Altitude != math.MaxUint16 {
			elevation = r.AltitudeScaled()
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

		if r.EnhancedSpeed != math.MaxUint32 {
			extra.Set("speed", r.EnhancedSpeedScaled())
		} else if r.Speed != math.MaxUint16 {
			extra.Set("speed", r.SpeedScaled())
		}

		elevationValue := elevation
		if math.IsNaN(elevationValue) {
			elevationValue = 0
		}

		lat := semicircles.ToDegrees(r.PositionLat)
		lng := semicircles.ToDegrees(r.PositionLong)
		var point *gogis.Point
		if !math.IsNaN(lat) && !math.IsNaN(lng) && (lat != 0 || lng != 0) {
			point = &gogis.Point{Lat: lat, Lng: lng}
		}

		points = append(points, model.WorkoutRecord{
			Time:          ts,
			Point:         point,
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

	data := &model.WorkoutGeoMeta{Center: model.MapCenter{}}

	data.UpdateExtraMetrics(points)

	return data, points
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
