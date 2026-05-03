package model

import (
	"database/sql"
	"slices"
	"time"

	"github.com/tkrajina/gpxgo/gpx"
)

func (w *Workout) ProcessRawRecords() {
	w.Data = GetGeoMeta(w)
	w.Records = slices.DeleteFunc(w.Records, func(r WorkoutRecord) bool {
		return r.Lat == 0 && r.Lng == 0
	})

	w.fixMissingDataRecords()
	w.markPauseRecordsFromEvents()
	w.UpdateAverages()
	w.UpdateExtraMetrics()
}

func (w *Workout) fixMissingDataRecords() {
	for i := range w.Records {
		if i+1 >= len(w.Records) {
			continue
		}

		r := &w.Records[i]
		rNext := w.Records[i+1]
		rPrev := WorkoutRecord{}
		if i > 0 {
			rPrev = w.Records[i-1]
		}

		if r.Distance2D == 0 && r.Distance != 0 {
			r.Distance2D = gpx.Distance2D(r.Lat, r.Lng, rNext.Lat, rNext.Lng, false)
			r.TotalDistance2D = rPrev.TotalDistance2D + r.Distance2D
		}
	}
}

func (w *Workout) markPauseRecordsFromEvents() {
	rIdx := 0
	var pauseStart time.Time
	var pauseStop time.Time
	for _, ev := range w.Events {
		if ev.Event != "timer" {
			continue
		}

		if ev.EventType == "stop_all" {
			pauseStart = ev.Timestamp
			continue
		}

		if ev.EventType == "start" {
			pauseStop = ev.Timestamp
		}

		for ; rIdx < len(w.Records); rIdx++ {
			r := &w.Records[rIdx]
			if (r.Time.After(pauseStart) && r.Time.Before(pauseStop)) || r.Time.Equal(pauseStop) || r.Time.Equal(pauseStart) {
				r.Pause = sql.NullBool{Valid: true, Bool: true}
			} else {
				r.Pause = sql.NullBool{Valid: true, Bool: false}
			}

			if r.Time.After(pauseStop) {
				break
			}
		}
	}
}

func GetGeoMeta(workout *Workout) *WorkoutGeoMeta {
	if len(workout.Records) == 0 {
		return nil
	}

	lat, lng := 0.0, 0.0
	validPoints := 0
	for _, r := range workout.Records {
		if r.Lat == 0 && r.Lng == 0 {
			continue
		}

		lat += r.Lat
		lng += r.Lng
		validPoints++
	}

	if validPoints == 0 {
		return nil
	}

	mc := MapCenter{
		Lat: lat / float64(validPoints),
		Lng: lng / float64(validPoints),
	}

	mc.UpdateTimezone()
	return &WorkoutGeoMeta{
		Center: mc,
	}
}
