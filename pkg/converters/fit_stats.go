package converters

import (
	"math"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/muktihari/fit/profile/filedef"
	"github.com/muktihari/fit/profile/mesgdef"
)

// parseWorkoutStats extracts session stats and applies swim fallbacks if available.
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

	stats.AverageSpeed = extractSessionAvgSpeed(session)
	if session.MaxSpeed != math.MaxUint16 {
		stats.MaxSpeed = session.MaxSpeedScaled()
	}

	stats.MinElevation = extractSessionMinElevation(session)
	stats.MaxElevation = extractSessionMaxElevation(session)

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
		enrichStatsFromSwimLengths(&stats, session, lengths)
	}

	return stats
}

func extractSessionAvgSpeed(session *mesgdef.Session) float64 {
	if session.EnhancedAvgSpeed != math.MaxUint32 {
		return session.EnhancedAvgSpeedScaled()
	} else if session.AvgSpeed != math.MaxUint16 {
		return session.AvgSpeedScaled()
	}
	return 0
}

func extractSessionMinElevation(session *mesgdef.Session) float64 {
	if session.EnhancedMinAltitude != math.MaxUint32 {
		return session.EnhancedMinAltitudeScaled()
	} else if session.MinAltitude != math.MaxUint16 {
		return session.MinAltitudeScaled()
	}
	return 0
}

func extractSessionMaxElevation(session *mesgdef.Session) float64 {
	if session.EnhancedMaxAltitude != math.MaxUint32 {
		return session.EnhancedMaxAltitudeScaled()
	} else if session.MaxAltitude != math.MaxUint16 {
		return session.MaxAltitudeScaled()
	}
	return 0
}

func enrichStatsFromSwimLengths(stats *model.WorkoutStats, session *mesgdef.Session, lengths []fitSwimLength) {
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
