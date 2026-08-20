package converters

import (
	"math"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/muktihari/fit/profile/filedef"
	"github.com/muktihari/fit/profile/mesgdef"
)

// parseLaps extracts workout laps from FIT activity lap messages.
func parseLaps(act *filedef.Activity) []model.WorkoutLap {
	if act == nil || len(act.Laps) == 0 {
		return nil
	}

	laps := make([]model.WorkoutLap, 0, len(act.Laps))
	for _, lap := range act.Laps {
		if lap == nil {
			continue
		}

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
		movingDuration := elapsed - pause
		avgSpeed := extractLapAvgSpeed(lap)
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
				MinElevation:        extractLapMinElevation(lap),
				MaxElevation:        extractLapMaxElevation(lap),
				TotalUp:             extractLapTotalUp(lap),
				TotalDown:           extractLapTotalDown(lap),
				AverageSpeed:        avgSpeed,
				AverageSpeedNoPause: avgSpeedNoPause,
				MaxSpeed:            extractLapMaxSpeed(lap),
				AverageCadence:      extractLapAvgCadence(lap),
				MaxCadence:          extractLapMaxCadence(lap),
				AverageHeartRate:    extractLapAvgHeartRate(lap),
				MaxHeartRate:        extractLapMaxHeartRate(lap),
				AveragePower:        extractLapAvgPower(lap),
				MaxPower:            extractLapMaxPower(lap),
			},
		})
	}

	return laps
}

func extractLapMinElevation(lap *mesgdef.Lap) float64 {
	if lap.EnhancedMinAltitude != math.MaxUint32 {
		return lap.EnhancedMinAltitudeScaled()
	} else if lap.MinAltitude != math.MaxUint16 {
		return lap.MinAltitudeScaled()
	}
	return 0
}

func extractLapMaxElevation(lap *mesgdef.Lap) float64 {
	if lap.EnhancedMaxAltitude != math.MaxUint32 {
		return lap.EnhancedMaxAltitudeScaled()
	} else if lap.MaxAltitude != math.MaxUint16 {
		return lap.MaxAltitudeScaled()
	}
	return 0
}

func extractLapAvgSpeed(lap *mesgdef.Lap) float64 {
	if lap.EnhancedAvgSpeed != math.MaxUint32 {
		return lap.EnhancedAvgSpeedScaled()
	} else if lap.AvgSpeed != math.MaxUint16 {
		return lap.AvgSpeedScaled()
	}
	return 0
}

func extractLapMaxSpeed(lap *mesgdef.Lap) float64 {
	if lap.EnhancedMaxSpeed != math.MaxUint32 {
		return lap.EnhancedMaxSpeedScaled()
	} else if lap.MaxSpeed != math.MaxUint16 {
		return lap.MaxSpeedScaled()
	}
	return 0
}

func extractLapAvgCadence(lap *mesgdef.Lap) float64 {
	if lap.AvgCadence != math.MaxUint8 {
		return float64(lap.AvgCadence)
	}
	return 0
}

func extractLapMaxCadence(lap *mesgdef.Lap) float64 {
	if lap.MaxCadence != math.MaxUint8 {
		return float64(lap.MaxCadence)
	}
	return 0
}

func extractLapAvgHeartRate(lap *mesgdef.Lap) float64 {
	if lap.AvgHeartRate != math.MaxUint8 {
		return float64(lap.AvgHeartRate)
	}
	return 0
}

func extractLapMaxHeartRate(lap *mesgdef.Lap) float64 {
	if lap.MaxHeartRate != math.MaxUint8 {
		return float64(lap.MaxHeartRate)
	}
	return 0
}

func extractLapAvgPower(lap *mesgdef.Lap) float64 {
	if lap.AvgPower != math.MaxUint16 {
		return float64(lap.AvgPower)
	}
	return 0
}

func extractLapMaxPower(lap *mesgdef.Lap) float64 {
	if lap.MaxPower != math.MaxUint16 {
		return float64(lap.MaxPower)
	}
	return 0
}

func extractLapTotalUp(lap *mesgdef.Lap) float64 {
	if lap.TotalAscent != math.MaxUint16 {
		return float64(lap.TotalAscent)
	}
	return 0
}

func extractLapTotalDown(lap *mesgdef.Lap) float64 {
	if lap.TotalDescent != math.MaxUint16 {
		return float64(lap.TotalDescent)
	}
	return 0
}
