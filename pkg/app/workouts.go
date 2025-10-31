package app

import (
	"io"
	"mime/multipart"
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
	"github.com/jovandeginste/workout-tracker/v2/pkg/geocoder"
	"github.com/jovandeginste/workout-tracker/v2/pkg/templatehelpers"
)

const (
	htmlDateFormat = "2006-01-02T15:04"
)

func uploadedFile(file *multipart.FileHeader) ([]byte, error) {
	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	// Read all from r into a bytes slice
	content, err := io.ReadAll(src)
	if err != nil {
		return nil, err
	}

	return content, nil
}

type ManualWorkout struct {
	Name            *string               `form:"name" json:"name"`
	Date            *string               `form:"date" json:"date"`
	Timezone        *string               `form:"timezone" json:"timezone"`
	Location        *string               `form:"location" json:"location"`
	DurationHours   *int                  `form:"duration_hours" json:"duration_hours"`
	DurationMinutes *int                  `form:"duration_minutes" json:"duration_minutes"`
	DurationSeconds *int                  `form:"duration_seconds" json:"duration_seconds"`
	Distance        *float64              `form:"distance" json:"distance"`
	Repetitions     *int                  `form:"repetitions" json:"repetitions"`
	Weight          *float64              `form:"weight" json:"weight"`
	Notes           *string               `form:"notes" json:"notes"`
	Type            *database.WorkoutType `form:"type" json:"type"`
	CustomType      *string               `form:"custom_type" json:"custom_type"`
	EquipmentIDs    []uint64              `form:"equipment_ids" json:"equipment_ids"`

	units *database.UserPreferredUnits
}

func (m *ManualWorkout) ToDate() *time.Time {
	if m.Date == nil {
		return nil
	}

	d, err := time.Parse(htmlDateFormat, *m.Date)
	if err != nil {
		return nil
	}

	if m.Timezone == nil {
		return &d
	}

	// Handle timezone offset
	tzLoc, err := time.LoadLocation(*m.Timezone)
	if err == nil {
		d = d.In(tzLoc)
	}

	_, zoneOffset := d.Zone()
	d = d.Add(-time.Duration(zoneOffset) * time.Second)

	// handle DST transitions
	if d.IsDST() {
		d = d.Add(1 * time.Hour)
	}

	return &d
}

func (m *ManualWorkout) ToWeight() *float64 {
	if m.Weight == nil || *m.Weight == 0 {
		return nil
	}

	d := templatehelpers.WeightToDatabase(*m.Weight, m.units.Weight())

	return &d
}

func (m *ManualWorkout) ToDistance() *float64 {
	if m.Distance == nil || *m.Distance == 0 {
		return nil
	}

	d := templatehelpers.DistanceToDatabase(*m.Distance, m.units.Distance())

	return &d
}

func (m *ManualWorkout) ToDuration() *time.Duration {
	var totalDuration time.Duration

	if m.DurationHours != nil {
		totalDuration += time.Duration(*m.DurationHours) * time.Hour
	}

	if m.DurationMinutes != nil {
		totalDuration += time.Duration(*m.DurationMinutes) * time.Minute
	}

	if m.DurationSeconds != nil {
		totalDuration += time.Duration(*m.DurationSeconds) * time.Second
	}

	if totalDuration == 0 {
		return nil
	}

	return &totalDuration
}

func setIfNotNil[T any](dst *T, src *T) {
	if src == nil {
		return
	}

	*dst = *src
}

func (m *ManualWorkout) Update(w *database.Workout) {
	if w.Data == nil {
		w.Data = &database.MapData{}
	}

	dDate := m.ToDate()

	setIfNotNil(&w.Name, m.Name)
	setIfNotNil(&w.Notes, m.Notes)
	setIfNotNil(&w.Date, dDate)
	setIfNotNil(&w.Type, m.Type)
	setIfNotNil(&w.CustomType, m.CustomType)

	setIfNotNil(&w.Data.AddressString, m.Location)
	setIfNotNil(&w.Data.TotalDistance, m.ToDistance())
	setIfNotNil(&w.Data.TotalDuration, m.ToDuration())
	setIfNotNil(&w.Data.TotalRepetitions, m.Repetitions)
	setIfNotNil(&w.Data.TotalWeight, m.ToWeight())

	if m.Location != nil && w.FullAddress() != *m.Location {
		a, err := geocoder.Find(*m.Location)
		if err != nil {
			w.Data.Address = nil
			return
		}

		w.Data.Address = a
		w.Data.UpdateAddress()
	}

	w.Data.UpdateExtraMetrics()
}
