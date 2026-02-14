package dto

import (
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/geocoder"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/jovandeginste/workout-tracker/v2/pkg/templatehelpers"
)

const htmlDateFormat = "2006-01-02T15:04"

type ManualWorkout struct {
	Name            *string            `form:"name" json:"name"`
	Date            *string            `form:"date" json:"date"`
	Timezone        *string            `form:"timezone" json:"timezone"`
	Location        *string            `form:"location" json:"location"`
	DurationHours   *int               `form:"duration_hours" json:"duration_hours"`
	DurationMinutes *int               `form:"duration_minutes" json:"duration_minutes"`
	DurationSeconds *int               `form:"duration_seconds" json:"duration_seconds"`
	Distance        *float64           `form:"distance" json:"distance"`
	Repetitions     *int               `form:"repetitions" json:"repetitions"`
	Weight          *float64           `form:"weight" json:"weight"`
	Notes           *string            `form:"notes" json:"notes"`
	Type            *model.WorkoutType `form:"type" json:"type"`
	CustomType      *string            `form:"custom_type" json:"custom_type"`
	EquipmentIDs    []uint64           `form:"equipment_ids" json:"equipment_ids"`

	Units *model.UserPreferredUnits `json:"-" form:"-"`
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

	tzLoc, err := time.LoadLocation(*m.Timezone)
	if err == nil {
		d = d.In(tzLoc)
	}

	_, zoneOffset := d.Zone()
	d = d.Add(-time.Duration(zoneOffset) * time.Second)

	if d.IsDST() {
		d = d.Add(1 * time.Hour)
	}

	return &d
}

func (m *ManualWorkout) ToWeight() *float64 {
	if m.Weight == nil || *m.Weight == 0 {
		return nil
	}

	unit := "kg"
	if m.Units != nil {
		unit = m.Units.Weight()
	}

	d := templatehelpers.WeightToDatabase(*m.Weight, unit)
	return &d
}

func (m *ManualWorkout) ToDistance() *float64 {
	if m.Distance == nil || *m.Distance == 0 {
		return nil
	}

	unit := "km"
	if m.Units != nil {
		unit = m.Units.Distance()
	}

	d := templatehelpers.DistanceToDatabase(*m.Distance, unit)
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

func (m *ManualWorkout) Update(w *model.Workout) {
	if w.Data == nil {
		w.Data = &model.MapData{}
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
	}

	w.Data.UpdateExtraMetrics()
}

type Measurement struct {
	Date       string  `form:"date" json:"date"`
	Steps      float64 `form:"steps" json:"steps"`
	WeightUnit string  `form:"weight_unit" json:"weight_unit"`
	HeightUnit string  `form:"height_unit" json:"height_unit"`

	Weight           float64 `form:"weight" json:"weight"`
	Height           float64 `form:"height" json:"height"`
	FTP              float64 `form:"ftp" json:"ftp"`
	RestingHeartRate float64 `form:"resting_heart_rate" json:"resting_heart_rate"`
	MaxHeartRate     float64 `form:"max_heart_rate" json:"max_heart_rate"`

	Units *model.UserPreferredUnits `json:"-" form:"-"`
}

func (m *Measurement) Time() time.Time {
	if m.Date == "" {
		return time.Now()
	}

	d, err := time.Parse("2006-01-02", m.Date)
	if err != nil {
		return time.Now()
	}

	return d
}

func (m *Measurement) ToSteps() *float64 {
	if m.Steps == 0 {
		return nil
	}

	d := m.Steps
	return &d
}

func (m *Measurement) ToFTP() *float64 {
	if m.FTP == 0 {
		return nil
	}

	d := m.FTP
	return &d
}

func (m *Measurement) ToRestingHeartRate() *float64 {
	if m.RestingHeartRate == 0 {
		return nil
	}

	d := m.RestingHeartRate
	return &d
}

func (m *Measurement) ToMaxHeartRate() *float64 {
	if m.MaxHeartRate == 0 {
		return nil
	}

	d := m.MaxHeartRate
	return &d
}

func (m *Measurement) ToHeight() *float64 {
	if m.Height == 0 {
		return nil
	}

	if m.HeightUnit == "" {
		if m.Units != nil {
			m.HeightUnit = m.Units.Height()
		} else {
			m.HeightUnit = "cm"
		}
	}

	d := templatehelpers.HeightToDatabase(m.Height, m.HeightUnit)
	return &d
}

func (m *Measurement) ToWeight() *float64 {
	if m.Weight == 0 {
		return nil
	}

	if m.WeightUnit == "" {
		if m.Units != nil {
			m.WeightUnit = m.Units.Weight()
		} else {
			m.WeightUnit = "kg"
		}
	}

	d := templatehelpers.WeightToDatabase(m.Weight, m.WeightUnit)
	return &d
}

func (m *Measurement) Update(measurement *model.Measurement) {
	setIfNotNil(&measurement.Weight, m.ToWeight())
	setIfNotNil(&measurement.Height, m.ToHeight())
	setIfNotNil(&measurement.Steps, m.ToSteps())
	setIfNotNil(&measurement.FTP, m.ToFTP())
	setIfNotNil(&measurement.RestingHeartRate, m.ToRestingHeartRate())
	setIfNotNil(&measurement.MaxHeartRate, m.ToMaxHeartRate())
}

func setIfNotNil[T any](dst *T, src *T) {
	if src == nil {
		return
	}

	*dst = *src
}

type SigninRequest struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
}

type RegisterRequest struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
	Name     string `json:"name" form:"name"`
}

type AdminUserUpdateData struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Admin    bool   `json:"admin"`
	Active   bool   `json:"active"`
	Password string `json:"password,omitempty"`
}

type ProfileUpdateData struct {
	Birthdate           *string                  `json:"birthdate"`
	PreferredUnits      model.UserPreferredUnits `json:"preferred_units"`
	Language            string                   `json:"language"`
	Theme               string                   `json:"theme"`
	TotalsShow          string                   `json:"totals_show"`
	Timezone            string                   `json:"timezone"`
	AutoImportDirectory string                   `json:"auto_import_directory"`
	APIActive           bool                     `json:"api_active"`
	SocialsDisabled     bool                     `json:"socials_disabled"`
	PreferFullDate      bool                     `json:"prefer_full_date"`
}

type CalendarQueryParams struct {
	Start    *string `query:"start"`
	End      *string `query:"end"`
	TimeZone *string `query:"timeZone"`
}
