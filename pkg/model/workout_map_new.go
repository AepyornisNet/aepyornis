package model

import "slices"

func (w *Workout) ProcessRawRecords() {
	w.Data = GetGeoMeta(w)
	w.Records = slices.DeleteFunc(w.Records, func(r WorkoutRecord) bool {
		return r.Lat == 0 && r.Lng == 0
	})

	w.UpdateAverages()
	w.UpdateExtraMetrics()
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

	mc.updateTimezone()
	return &WorkoutGeoMeta{
		Center: mc,
	}
}
