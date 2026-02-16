package activitypub

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/muktihari/fit/encoder"
	"github.com/muktihari/fit/profile/filedef"
	"github.com/muktihari/fit/profile/mesgdef"
	"github.com/muktihari/fit/profile/typedef"
)

var ErrWorkoutMissingData = errors.New("workout has no data")

const FitMIMEType = "application/vnd.ant.fit"

func GenerateWorkoutFIT(workout *model.Workout) ([]byte, error) {
	if workout == nil || workout.Data == nil {
		return nil, ErrWorkoutMissingData
	}

	start := workout.Date.UTC()
	if start.IsZero() {
		start = time.Now().UTC()
	}

	totalDuration := workout.TotalDuration()
	if totalDuration <= 0 {
		totalDuration = time.Second
	}

	end := start.Add(totalDuration)

	activity := filedef.NewActivity()
	activity.FileId.
		SetType(typedef.FileActivity).
		SetTimeCreated(start).
		SetManufacturer(typedef.ManufacturerDevelopment).
		SetProduct(0).
		SetProductName("Workout Tracker")

	activity.Records = buildWorkoutRecords(workout, start)
	activity.Laps = buildWorkoutLaps(workout, start, end)

	timerDuration := max(totalDuration-workout.PauseDuration(), 0)
	session := mesgdef.NewSession(nil).
		SetTimestamp(end).
		SetStartTime(start).
		SetSport(fitSportForWorkout(workout)).
		SetSubSport(fitSubSportForWorkout(workout)).
		SetTotalDistanceScaled(workout.TotalDistance()).
		SetTotalElapsedTimeScaled(totalDuration.Seconds()).
		SetTotalTimerTimeScaled(timerDuration.Seconds()).
		SetAvgSpeedScaled(workout.AverageSpeed()).
		SetTotalAscent(clampUint16(math.Round(workout.TotalUp()))).
		SetTotalDescent(clampUint16(math.Round(workout.TotalDown()))).
		SetSportProfileName(sportProfileName(workout))

	if workout.MaxSpeed() > 0 {
		session.SetMaxSpeedScaled(workout.MaxSpeed())
	}

	if workout.Data.AverageCadence > 0 {
		session.SetAvgCadence(clampUint8(math.Round(workout.Data.AverageCadence)))
	}

	if workout.Data.MaxCadence > 0 {
		session.SetMaxCadence(clampUint8(math.Round(workout.Data.MaxCadence)))
	}

	if workout.Data.AveragePower > 0 {
		session.SetAvgPower(clampUint16(math.Round(workout.Data.AveragePower)))
	}

	if workout.Data.MaxPower > 0 {
		session.SetMaxPower(clampUint16(math.Round(workout.Data.MaxPower)))
	}

	activity.Sessions = append(activity.Sessions, session)
	activity.Activity = mesgdef.NewActivity(nil).
		SetType(typedef.ActivityManual).
		SetTimestamp(end).
		SetLocalTimestamp(end.Local()).
		SetNumSessions(1)

	fitData := activity.ToFIT(nil)
	buf := bytes.NewBuffer(nil)
	enc := encoder.New(buf)
	if err := enc.Encode(&fitData); err != nil {
		return nil, fmt.Errorf("failed to encode FIT file: %w", err)
	}

	return buf.Bytes(), nil
}

func WorkoutFITFilename(workout *model.Workout) string {
	if workout == nil {
		return "workout.fit"
	}

	return fmt.Sprintf("workout-%d.fit", workout.ID)
}

func WorkoutNoteContent(workout *model.Workout) string {
	if workout == nil {
		return "Workout"
	}

	parts := []string{fmt.Sprintf("%s", workout.Name)}
	if d := workout.TotalDistance(); d > 0 {
		parts = append(parts, fmt.Sprintf("distance: %.2f km", d/1000.0))
	}

	if dur := workout.TotalDuration(); dur > 0 {
		parts = append(parts, fmt.Sprintf("duration: %s", dur.Round(time.Second).String()))
	}

	if speed := workout.AverageSpeed(); speed > 0 {
		parts = append(parts, fmt.Sprintf("avg speed: %.2f km/h", speed*3.6))
	}

	if up := workout.TotalUp(); up > 0 {
		parts = append(parts, fmt.Sprintf("elevation gain: %.0f m", up))
	}

	if reps := workout.Repetitions(); reps > 0 {
		parts = append(parts, fmt.Sprintf("repetitions: %d", reps))
	}

	if weight := workout.Weight(); weight > 0 {
		parts = append(parts, fmt.Sprintf("weight: %.2f kg", weight))
	}

	return strings.Join(parts, "\n")
}

func buildWorkoutRecords(workout *model.Workout, fallbackStart time.Time) []*mesgdef.Record {
	if workout == nil || workout.Data == nil || workout.Data.Details == nil || len(workout.Data.Details.Points) == 0 {
		return nil
	}

	records := make([]*mesgdef.Record, 0, len(workout.Data.Details.Points))
	for i, p := range workout.Data.Details.Points {
		ts := p.Time
		if ts.IsZero() {
			ts = fallbackStart.Add(p.TotalDuration)
			if ts.IsZero() {
				ts = fallbackStart.Add(time.Duration(i) * time.Second)
			}
		}

		rec := mesgdef.NewRecord(nil).SetTimestamp(ts)
		if !math.IsNaN(p.Lat) && !math.IsNaN(p.Lng) && (p.Lat != 0 || p.Lng != 0) {
			rec.SetPositionLatDegrees(p.Lat).SetPositionLongDegrees(p.Lng)
		}

		elevation := p.EnhancedElevation()
		if !math.IsNaN(elevation) {
			rec.SetEnhancedAltitudeScaled(elevation)
		}

		if p.TotalDistance > 0 {
			rec.SetDistanceScaled(p.TotalDistance)
		}

		speed := p.AverageSpeed()
		if speed > 0 {
			rec.SetSpeedScaled(speed)
		}

		if cadence, ok := p.ExtraMetrics["cadence"]; ok && cadence > 0 {
			rec.SetCadence(clampUint8(math.Round(cadence)))
		}

		if power, ok := p.ExtraMetrics["power"]; ok && power > 0 {
			rec.SetPower(clampUint16(math.Round(power)))
		}

		records = append(records, rec)
	}

	return records
}

func buildWorkoutLaps(workout *model.Workout, start, end time.Time) []*mesgdef.Lap {
	if workout == nil || workout.Data == nil || len(workout.Data.Laps) == 0 {
		return []*mesgdef.Lap{mesgdef.NewLap(nil).
			SetStartTime(start).
			SetTimestamp(end).
			SetTotalDistanceScaled(workout.TotalDistance()).
			SetTotalElapsedTimeScaled(workout.TotalDuration().Seconds()).
			SetTotalTimerTimeScaled(max(workout.TotalDuration()-workout.PauseDuration(), 0).Seconds()).
			SetAvgSpeedScaled(workout.AverageSpeed())}
	}

	laps := make([]*mesgdef.Lap, 0, len(workout.Data.Laps))
	for _, lap := range workout.Data.Laps {
		lapStart := lap.Start
		if lapStart.IsZero() {
			lapStart = start
		}

		lapEnd := lap.Stop
		if lapEnd.IsZero() {
			lapEnd = lapStart.Add(max(lap.TotalDuration, time.Second))
		}

		moving := max(lap.TotalDuration-lap.PauseDuration, 0)
		l := mesgdef.NewLap(nil).
			SetStartTime(lapStart).
			SetTimestamp(lapEnd).
			SetTotalDistanceScaled(lap.TotalDistance).
			SetTotalElapsedTimeScaled(lap.TotalDuration.Seconds()).
			SetTotalTimerTimeScaled(moving.Seconds()).
			SetAvgSpeedScaled(lap.AverageSpeed).
			SetMaxSpeedScaled(lap.MaxSpeed).
			SetTotalAscent(clampUint16(math.Round(lap.TotalUp))).
			SetTotalDescent(clampUint16(math.Round(lap.TotalDown)))

		if lap.AverageCadence > 0 {
			l.SetAvgCadence(clampUint8(math.Round(lap.AverageCadence)))
		}

		if lap.MaxCadence > 0 {
			l.SetMaxCadence(clampUint8(math.Round(lap.MaxCadence)))
		}

		if lap.AveragePower > 0 {
			l.SetAvgPower(clampUint16(math.Round(lap.AveragePower)))
		}

		if lap.MaxPower > 0 {
			l.SetMaxPower(clampUint16(math.Round(lap.MaxPower)))
		}

		laps = append(laps, l)
	}

	return laps
}

func fitSportForWorkout(workout *model.Workout) typedef.Sport {
	if workout == nil {
		return typedef.SportGeneric
	}

	sport := typedef.SportFromString(string(workout.Type))
	if sport != typedef.SportInvalid {
		return sport
	}

	if workout.Data != nil {
		sport = typedef.SportFromString(workout.Data.Type)
		if sport != typedef.SportInvalid {
			return sport
		}
	}

	return typedef.SportGeneric
}

func fitSubSportForWorkout(workout *model.Workout) typedef.SubSport {
	if workout == nil || workout.Data == nil {
		return typedef.SubSportGeneric
	}

	s := typedef.SubSportFromString(workout.Data.SubType)
	if s == typedef.SubSportInvalid {
		return typedef.SubSportGeneric
	}

	return s
}

func sportProfileName(workout *model.Workout) string {
	if workout == nil {
		return "Workout"
	}

	name := strings.ReplaceAll(string(workout.Type), "-", " ")
	if name == "" {
		name = "Workout"
	}

	return name
}

func clampUint16(v float64) uint16 {
	if math.IsNaN(v) || v <= 0 {
		return 0
	}

	if v >= math.MaxUint16 {
		return math.MaxUint16
	}

	return uint16(v)
}

func clampUint8(v float64) uint8 {
	if math.IsNaN(v) || v <= 0 {
		return 0
	}

	if v >= math.MaxUint8 {
		return math.MaxUint8
	}

	return uint8(v)
}
