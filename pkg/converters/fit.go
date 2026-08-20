package converters

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/muktihari/fit/decoder"
	"github.com/muktihari/fit/kit/datetime"
	"github.com/muktihari/fit/profile/filedef"
	"github.com/muktihari/fit/profile/typedef"
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
		total += max(lap.TotalDuration-lap.PauseDuration, 0)
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

	pause := max(elapsed-moving, 0)

	return elapsed, moving, pause
}

func cloneMapData(src *model.WorkoutGeoMeta) *model.WorkoutGeoMeta {
	if src == nil {
		return &model.WorkoutGeoMeta{}
	}

	clone := *src

	return &clone
}

func fitActivityStartTime(act *filedef.Activity) time.Time {
	if act == nil {
		return time.Time{}
	}

	if act.Activity != nil {
		if t := firstNonZeroTime(act.Activity.LocalTimestamp.Local(), act.Activity.Timestamp.Local()); !t.IsZero() {
			return t
		}
	}

	if t := firstValidSessionOrLapTime(act); !t.IsZero() {
		return t
	}

	if t := firstValidLengthOrRecordTime(act); !t.IsZero() {
		return t
	}

	if fitTimeIsValid(act.FileId.TimeCreated.Local()) {
		return act.FileId.TimeCreated.Local()
	}

	return time.Time{}
}

func firstValidSessionOrLapTime(act *filedef.Activity) time.Time {
	for _, s := range act.Sessions {
		if s != nil {
			if t := firstNonZeroTime(s.StartTime.Local(), s.Timestamp.Local()); !t.IsZero() {
				return t
			}
		}
	}

	for _, l := range act.Laps {
		if l != nil {
			if t := firstNonZeroTime(l.StartTime.Local(), l.Timestamp.Local()); !t.IsZero() {
				return t
			}
		}
	}

	return time.Time{}
}

func firstValidLengthOrRecordTime(act *filedef.Activity) time.Time {
	for _, l := range act.Lengths {
		if l != nil && fitTimeIsValid(l.StartTime.Local()) {
			return l.StartTime.Local()
		}
	}

	for _, r := range act.Records {
		if r != nil && fitTimeIsValid(r.Timestamp.Local()) {
			return r.Timestamp.Local()
		}
	}

	return time.Time{}
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
