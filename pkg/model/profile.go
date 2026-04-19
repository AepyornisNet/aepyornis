package model

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	DefaultProfileLanguage = "browser"
	DefaultProfileTheme    = "browser"
)

type Profile struct {
	Model

	UserID *uint64 `json:"user_id,omitempty"`          // The ID of the user who owns this profile
	User   *User   `gorm:"foreignKey:UserID" json:"-"` // The user who owns this profile

	Local bool `json:"local"` // Whether the profile belongs to a user from this instance

	Username    string          `gorm:"not null;type:varchar(128);uniqueIndex:idx_profiles_username_domain,priority:1" json:"username"` // The user's username
	Domain      *string         `gorm:"type:varchar(255);uniqueIndex:idx_profiles_username_domain,priority:2" json:"domain,omitempty"`  // The profile domain for remote accounts
	DisplayName string          `gorm:"type:varchar(64);not null" json:"display_name"`                                                  // The user's name
	Birthdate   *datatypes.Date `json:"birthdate,omitempty"`                                                                            // The user's birthdate

	Workouts  []Workout   `gorm:"constraint:OnDelete:CASCADE" json:"-"` // The profiles's workouts
	Equipment []Equipment `gorm:"constraint:OnDelete:CASCADE" json:"-"` // The profiles's equipment

	PublicKey  string `gorm:"type:text"` // The user's public key for ActivityPub federation
	PrivateKey string `gorm:"type:text"` // The user's private key for ActivityPub federation
}

func (p *Profile) Save(db *gorm.DB) error {
	return db.Save(p).Error
}

func (p *Profile) BeforeSave(_ *gorm.DB) error {
	p.normalizeDomain()
	return nil
}

func (p *Profile) normalizeDomain() {
	// Local profiles belong to a local user record and must not carry a domain.
	if p.UserID != nil || p.User != nil {
		p.Domain = nil
		return
	}

	if p.Domain == nil {
		return
	}

	d := strings.TrimSpace(*p.Domain)
	if d == "" {
		p.Domain = nil
		return
	}

	p.Domain = &d
}

func (p *Profile) MarkWorkoutsDirty(db *gorm.DB) error {
	if p == nil || p.ID == 0 {
		return ErrNoUser
	}

	return db.Model(&Workout{}).Where(&Workout{ProfileID: p.ID}).Update("dirty", true).Error
}

func (p *Profile) GetAllEquipment(db *gorm.DB) ([]*Equipment, error) {
	if p == nil || p.ID == 0 {
		return nil, ErrNoUser
	}

	var equipment []*Equipment
	if err := db.Preload("Workouts").Where(&Equipment{ProfileID: p.ID}).Order("name DESC").Find(&equipment).Error; err != nil {
		return nil, err
	}

	return equipment, nil
}

func (p *Profile) MaxHeartRateAt(d time.Time) float64 {
	val := p.measurementAt("max_heart_rate", d)
	if val == 0 {
		if p == nil || p.Birthdate == nil {
			return 200
		}

		return calculateMaxHeartRate(time.Time(*p.Birthdate), d)
	}

	return val
}

func (p *Profile) measurementAt(key string, d time.Time) float64 {
	if p == nil || p.User == nil || p.User.db == nil {
		return 0
	}

	return p.User.measurementAt(key, d)
}

func (p *Profile) GenerateActivityPubKeys(force bool) error {
	if !force && p.PublicKey != "" && p.PrivateKey != "" {
		return nil
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return err
	}

	privatePEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return err
	}

	publicPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	p.PrivateKey = string(privatePEM)
	p.PublicKey = string(publicPEM)

	return nil
}

func (p *Profile) AddWorkout(db *gorm.DB, workoutType WorkoutType, notes string, filename string, content []byte) ([]*Workout, []error) {
	if p == nil {
		return nil, []error{ErrNoUser}
	}

	ws, err := NewWorkout(p, workoutType, notes, filename, content)
	if err != nil {
		return nil, []error{fmt.Errorf("%w: %s", ErrInvalidData, err)}
	}

	errs := []error{}
	defaultVisibility := WorkoutVisibilityPrivate
	if p.User != nil {
		defaultVisibility = p.User.EffectiveDefaultWorkoutVisibility()
	}

	for _, w := range ws {
		w.Visibility = defaultVisibility

		if err := w.Create(db); err != nil {
			errs = append(errs, err)
			continue
		}

		var equipment []*Equipment

		for i, e := range p.Equipment {
			if e.ValidFor(&w.Type) {
				equipment = append(equipment, &p.Equipment[i])
			}
		}

		if err := db.Model(&w).Association("Equipment").Replace(equipment); err != nil {
			errs = append(errs, err)
		}
	}

	return ws, errs
}
