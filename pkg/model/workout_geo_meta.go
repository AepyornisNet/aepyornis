package model

import (
	"slices"
	"strings"

	"github.com/codingsince1985/geo-golang"
	"github.com/jovandeginste/workout-tracker/v2/pkg/templatehelpers"
	"gorm.io/gorm"
)

type WorkoutGeoMeta struct {
	Model
	Address *geo.Address `gorm:"serializer:json" json:"address"` // The address of the workout

	Workout       *Workout  `gorm:"foreignKey:WorkoutID" json:"-"`         // The workout this geo meta belongs to
	AddressString string    `json:"addressString"`                         // The generic location of the workout
	Center        MapCenter `gorm:"serializer:json" json:"center"`         // The center of the workout (in coordinates)
	WorkoutID     uint64    `gorm:"not null;uniqueIndex" json:"workoutID"` // The workout this data belongs to
}

func (WorkoutGeoMeta) TableName() string {
	return "workout_geo_meta"
}

func (m *WorkoutGeoMeta) UpdateExtraMetrics(points []WorkoutRecord) []string {
	if m == nil ||
		len(points) == 0 {
		return nil
	}

	metrics := []string{}
	found := map[string]bool{}

	for _, d := range points {
		for k := range d.ExtraMetrics {
			if found[k] {
				continue
			}

			metrics = append(metrics, k)
			found[k] = true
		}
	}

	slices.Sort(metrics)

	return metrics
}

func addressIsUnset(a *geo.Address) bool {
	if a == nil {
		return true
	}

	if a.Country == "" {
		return true
	}

	return false
}

func (m *WorkoutGeoMeta) UpdateAddress() {
	if addressIsUnset(m.Address) && !m.Center.IsZero() {
		m.Address = m.Center.Address()
	}

	if addressIsUnset(m.Address) && m.hasAddressString() {
		return
	}

	m.AddressString = m.addressString()
}

func (m *WorkoutGeoMeta) hasAddressString() bool {
	switch m.AddressString {
	case "", UnknownLocation:
		return false
	default:
		return true
	}
}

func (m *WorkoutGeoMeta) addressString() string {
	if addressIsUnset(m.Address) {
		return ""
	}

	r := ""
	if m.Address.CountryCode != "" {
		r += templatehelpers.CountryToFlag(m.Address.CountryCode) + " "
	}

	switch {
	case m.Address.City != "":
		r += m.Address.City
	case m.Address.Street != "":
		r += m.Address.Street
	default:
		return r + m.Address.FormattedAddress
	}

	if shouldAddState(m.Address) {
		r += ", " + m.Address.State
	}

	return r
}

func (m *WorkoutGeoMeta) shouldPersist() bool {
	if m == nil {
		return false
	}

	return !m.Center.IsZero() || !addressIsUnset(m.Address) || m.AddressString != ""
}

func shouldAddState(address *geo.Address) bool {
	return address.CountryCode == "US"
}

func (m *WorkoutGeoMeta) Save(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if strings.TrimSpace(m.AddressString) == UnknownLocation {
			m.AddressString = ""
		}

		if !m.shouldPersist() {
			if m.WorkoutID != 0 {
				if err := tx.Where("workout_id = ?", m.WorkoutID).Delete(&WorkoutGeoMeta{}).Error; err != nil {
					return err
				}
			}

			m.ID = 0
			return nil
		}

		if err := tx.Save(m).Error; err != nil {
			return err
		}

		return nil
	})
}
