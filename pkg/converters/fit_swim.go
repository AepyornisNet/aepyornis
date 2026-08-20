package converters

import (
	"database/sql"
	"math"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/muktihari/fit/profile/filedef"
	"github.com/muktihari/fit/profile/mesgdef"
	"github.com/muktihari/fit/profile/typedef"
)

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
		elapsed, timer := extractSwimLengthElapsedAndTimer(l)
		isActive := l.LengthType == typedef.LengthTypeActive
		speed := 0.0
		cadence := 0.0
		strokes := l.TotalStrokes

		if isActive {
			cumDist += poolLength
			speed = extractSwimLengthSpeed(l, poolLength, timer)
			cadence = extractSwimLengthCadence(l, timer, strokes)
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

func extractSwimLengthElapsedAndTimer(l *mesgdef.Length) (float64, float64) {
	elapsed := l.TotalElapsedTimeScaled()
	if math.IsNaN(elapsed) || elapsed < 0 {
		elapsed = 0
	}
	timer := l.TotalTimerTimeScaled()
	if math.IsNaN(timer) || timer < 0 {
		timer = elapsed
	}
	return elapsed, timer
}

func extractSwimLengthSpeed(l *mesgdef.Length, poolLength, timer float64) float64 {
	if l.AvgSpeed != math.MaxUint16 && !math.IsNaN(l.AvgSpeedScaled()) && l.AvgSpeedScaled() > 0 {
		return l.AvgSpeedScaled()
	}
	if timer > 0 {
		return poolLength / timer
	}
	return 0
}

func extractSwimLengthCadence(l *mesgdef.Length, timer float64, strokes uint16) float64 {
	if l.AvgSwimmingCadence != math.MaxUint8 && l.AvgSwimmingCadence > 0 {
		return float64(l.AvgSwimmingCadence)
	}
	if strokes != math.MaxUint16 && timer > 0 {
		return (float64(strokes) / timer) * 60.0
	}
	return 0
}

func interpolateSwimLengthMetrics(ts time.Time, currLen fitSwimLength, priorActiveDist, poolLength float64) (float64, float64, float64, bool) {
	switch {
	case ts.Before(currLen.start):
		return priorActiveDist, 0, 0, true
	case ts.After(currLen.end):
		totalD := priorActiveDist
		if currLen.isActive {
			totalD += poolLength
		}
		return totalD, 0, 0, true
	case currLen.isActive:
		frac := 1.0
		if currLen.timer > 0 {
			frac = math.Min(math.Max(ts.Sub(currLen.start).Seconds()/currLen.timer, 0.0), 1.0)
		}
		totalD := priorActiveDist + frac*poolLength
		return totalD, currLen.speed, currLen.cadence, false
	default:
		return priorActiveDist, 0, 0, true
	}
}

func advanceSwimLengthIndex(lengths []fitSwimLength, lenIdx int, ts time.Time, priorActiveDist, poolLength float64) (int, float64) {
	for lenIdx+1 < len(lengths) && ts.After(lengths[lenIdx].end) {
		if lengths[lenIdx].isActive {
			priorActiveDist += poolLength
		}
		lenIdx++
	}
	return lenIdx, priorActiveDist
}

// mapDataFromSwimActivity maps swim records and lengths into WorkoutRecords with distance,
// speed, cadence and pause states.
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
		if r == nil {
			continue
		}

		ts := r.Timestamp.Local()
		if ts.IsZero() {
			continue
		}

		lenIdx, priorActiveDist = advanceSwimLengthIndex(lengths, lenIdx, ts, priorActiveDist, poolLength)
		totalD, sp, cad, isPause := interpolateSwimLengthMetrics(ts, lengths[lenIdx], priorActiveDist, poolLength)

		deltaD := 0.0
		if i == 0 {
			prevDist = totalD
		} else if totalD >= prevDist {
			deltaD = totalD - prevDist
			prevDist = totalD
		}

		dt := time.Duration(0)
		if i+1 < len(act.Records) && act.Records[i+1] != nil {
			dt = max(act.Records[i+1].Timestamp.Sub(ts), 0)
		}
		totalDuration += dt

		elevation := extractRecordElevation(r)
		extra := extractRecordExtraMetrics(r, elevation)
		if sp > 0 {
			extra.Set("speed", sp)
		}
		if cad > 0 {
			extra.Set("cadence", cad)
		}

		elevationValue := elevation
		if math.IsNaN(elevationValue) {
			elevationValue = 0
		}

		points = append(points, model.WorkoutRecord{
			Time:            ts,
			Point:           extractRecordPoint(r),
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

func synthesizeSwimLengthMetricsAtOffset(l fitSwimLength, priorActiveDist, poolLength, tOffset float64) (float64, float64, float64, bool) {
	if !l.isActive {
		return priorActiveDist, 0, 0, true
	}

	frac := 1.0
	if l.timer > 0 {
		frac = math.Min(math.Max(tOffset/l.timer, 0.0), 1.0)
	}
	totalD := priorActiveDist + frac*poolLength
	return totalD, l.speed, l.cadence, false
}

// synthesizeRecordsFromSwimLengths synthesizes workout records from swim lengths when
// no 1Hz records are present in the FIT file.
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
		points = synthesizeSwimLengthPoints(l, priorActiveDist, poolLength, &prevDist, &totalDuration, points)
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

func synthesizeSwimLengthPoints(
	l fitSwimLength,
	priorActiveDist, poolLength float64,
	prevDist *float64,
	totalDuration *time.Duration,
	existingPoints []model.WorkoutRecord,
) []model.WorkoutRecord {
	if l.elapsed <= 0 {
		return existingPoints
	}

	seconds := max(int(math.Ceil(l.elapsed)), 1)
	points := existingPoints

	for s := 0; s <= seconds; s++ {
		tOffset := min(float64(s), l.elapsed)
		ts := l.start.Add(time.Duration(tOffset * float64(time.Second)))
		if ts.After(l.end) {
			ts = l.end
		}

		if len(points) > 0 && points[len(points)-1].Time.Equal(ts) {
			continue
		}

		totalD, sp, cad, isPause := synthesizeSwimLengthMetricsAtOffset(l, priorActiveDist, poolLength, tOffset)

		deltaD := 0.0
		if len(points) == 0 {
			*prevDist = totalD
		} else if totalD >= *prevDist {
			deltaD = totalD - *prevDist
			*prevDist = totalD
		}

		dt := time.Duration(0)
		if len(points) > 0 {
			dt = ts.Sub(points[len(points)-1].Time)
			points[len(points)-1].Duration = dt
		}
		*totalDuration += dt

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
			TotalDuration:   *totalDuration,
			ExtraMetrics:    extra,
			Pause:           sql.NullBool{Valid: true, Bool: isPause},
		})
	}

	return points
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

		pauseDuration := max(setDuration-timerDuration, 0)

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
