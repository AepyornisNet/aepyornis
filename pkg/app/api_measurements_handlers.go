package app

import (
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
	"github.com/jovandeginste/workout-tracker/v2/pkg/templatehelpers"
)

type Measurement struct {
	Date       string  `form:"date" json:"date"`               // The date of the measurement
	Steps      float64 `form:"steps" json:"steps"`             // The number of steps taken
	WeightUnit string  `form:"weight_unit" json:"weight_unit"` // The unit of the weight (or the user's preferred unit)
	HeightUnit string  `form:"height_unit" json:"height_unit"` // The unit of the height (or the user's preferred unit)

	Weight           float64 `form:"weight" json:"weight"`                         // The weight of the user, in kilograms
	Height           float64 `form:"height" json:"height"`                         // The height of the user, in centimeter
	FTP              float64 `form:"ftp" json:"ftp"`                               // Functional Threshold Power, in watts
	RestingHeartRate float64 `form:"resting_heart_rate" json:"resting_heart_rate"` // Resting heart rate, in bpm
	MaxHeartRate     float64 `form:"max_heart_rate" json:"max_heart_rate"`         // Maximum heart rate, in bpm

	units *database.UserPreferredUnits
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
		m.HeightUnit = m.units.Height()
	}

	d := templatehelpers.HeightToDatabase(m.Height, m.HeightUnit)

	return &d
}

func (m *Measurement) ToWeight() *float64 {
	if m.Weight == 0 {
		return nil
	}

	if m.WeightUnit == "" {
		m.WeightUnit = m.units.Weight()
	}

	d := templatehelpers.WeightToDatabase(m.Weight, m.WeightUnit)

	return &d
}

func (m *Measurement) Update(measurement *database.Measurement) {
	setIfNotNil(&measurement.Weight, m.ToWeight())
	setIfNotNil(&measurement.Height, m.ToHeight())
	setIfNotNil(&measurement.Steps, m.ToSteps())
	setIfNotNil(&measurement.FTP, m.ToFTP())
	setIfNotNil(&measurement.RestingHeartRate, m.ToRestingHeartRate())
	setIfNotNil(&measurement.MaxHeartRate, m.ToMaxHeartRate())
}
