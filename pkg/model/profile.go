package model

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"
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

	Username        string          `gorm:"not null;type:varchar(128);uniqueIndex:idx_profiles_username_domain,priority:1" json:"username"` // The user's username
	Domain          *string         `gorm:"type:varchar(255);uniqueIndex:idx_profiles_username_domain,priority:2" json:"domain,omitempty"`  // The profile domain for remote accounts
	DisplayName     string          `gorm:"type:varchar(64);not null" json:"display_name"`                                                  // The user's name
	Birthdate       *datatypes.Date `json:"birthdate,omitempty"`                                                                            // The user's birthdate
	URL             *string         `gorm:"type:text;uniqueIndex" json:"url,omitempty"`
	InboxURL        *string         `gorm:"type:text" json:"inbox_url,omitempty"`
	OutboxURL       *string         `gorm:"type:text" json:"outbox_url,omitempty"`
	FollowersURL    *string         `gorm:"type:text" json:"followers_url,omitempty"`
	AvatarRemoteURL *string         `gorm:"type:text" json:"avatar_remote_url,omitempty"`

	Workouts  []Workout   `gorm:"constraint:OnDelete:CASCADE" json:"-"` // The profiles's workouts
	Equipment []Equipment `gorm:"constraint:OnDelete:CASCADE" json:"-"` // The profiles's equipment

	PublicKey  string `gorm:"type:text"` // The user's public key for ActivityPub federation
	PrivateKey string `gorm:"type:text"` // The user's private key for ActivityPub federation
}

func (p *Profile) Save(db *gorm.DB) error {
	return db.Save(p).Error
}

func (p *Profile) BeforeSave(_ *gorm.DB) error {
	p.normalizeLocality()
	p.normalizeDomain()
	p.normalizeRemoteFields()
	return nil
}

func (p *Profile) normalizeLocality() {
	p.Local = p.isLocalProfile()
}

func (p *Profile) isLocalProfile() bool {
	if p.UserID != nil || p.User != nil {
		return true
	}

	return p.Domain == nil && p.URL == nil && p.InboxURL == nil && p.OutboxURL == nil && p.FollowersURL == nil
}

func (p *Profile) normalizeDomain() {
	// Local profiles belong to a local user record and must not carry remote addressing.
	if p.isLocalProfile() {
		p.Domain = nil
		return
	}

	if p.Domain == nil && p.URL != nil {
		if parsed, err := url.Parse(strings.TrimSpace(*p.URL)); err == nil && parsed.Host != "" {
			host := parsed.Host
			p.Domain = &host
		}
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

func (p *Profile) normalizeRemoteFields() {
	if p.isLocalProfile() {
		p.URL = nil
		p.InboxURL = nil
		p.OutboxURL = nil
		p.FollowersURL = nil
		p.AvatarRemoteURL = nil
		return
	}

	p.URL = normalizeOptionalString(p.URL)
	p.InboxURL = normalizeOptionalString(p.InboxURL)
	p.OutboxURL = normalizeOptionalString(p.OutboxURL)
	p.FollowersURL = normalizeOptionalString(p.FollowersURL)
	p.AvatarRemoteURL = normalizeOptionalString(p.AvatarRemoteURL)

	if p.URL == nil && p.Domain != nil && strings.TrimSpace(p.Username) != "" {
		u := fmt.Sprintf("https://%s/ap/users/%s", strings.TrimSpace(*p.Domain), strings.TrimSpace(p.Username))
		p.URL = &u
	}

	if p.URL != nil {
		base := strings.TrimRight(*p.URL, "/")
		if p.InboxURL == nil {
			inbox := base + "/inbox"
			p.InboxURL = &inbox
		}
		if p.OutboxURL == nil {
			outbox := base + "/outbox"
			p.OutboxURL = &outbox
		}
		if p.FollowersURL == nil {
			followers := base + "/followers"
			p.FollowersURL = &followers
		}
	}
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func (p *Profile) ActorURL() string {
	if p == nil {
		return ""
	}

	if p.URL != nil && strings.TrimSpace(*p.URL) != "" {
		return strings.TrimSpace(*p.URL)
	}

	if p.Domain != nil && strings.TrimSpace(*p.Domain) != "" && strings.TrimSpace(p.Username) != "" {
		return fmt.Sprintf("https://%s/ap/users/%s", strings.TrimSpace(*p.Domain), strings.TrimSpace(p.Username))
	}

	return ""
}

func (p *Profile) UpsertRemote(db *gorm.DB) (*Profile, error) {
	if p != nil && (p.ID != 0 || p.isLocalProfile()) {
		return p, nil
	}
	if err := validateRemoteUpsert(p, db); err != nil {
		return nil, err
	}

	ensureRemoteDisplayName(p)
	existing, err := findExistingRemoteProfile(db, p)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := db.Create(p).Error; err != nil {
			return nil, err
		}
		return p, nil
	}
	if err != nil {
		return nil, err
	}

	mergeRemoteProfile(&existing, p)

	if err := db.Save(&existing).Error; err != nil {
		return nil, err
	}

	return &existing, nil
}

func validateRemoteUpsert(p *Profile, db *gorm.DB) error {
	if p == nil {
		return errors.New("profile is nil")
	}
	if db == nil {
		return errors.New("db is nil")
	}
	if p.URL != nil && strings.TrimSpace(*p.URL) != "" {
		return nil
	}
	if p.Domain != nil && strings.TrimSpace(*p.Domain) != "" && strings.TrimSpace(p.Username) != "" {
		return nil
	}

	return errors.New("remote profile requires url or username/domain")
}

func ensureRemoteDisplayName(p *Profile) {
	if p != nil && p.DisplayName == "" {
		p.DisplayName = p.Username
	}
}

func findExistingRemoteProfile(db *gorm.DB, p *Profile) (Profile, error) {
	var existing Profile
	if p.URL != nil && strings.TrimSpace(*p.URL) != "" {
		return existing, db.Where("url = ?", strings.TrimSpace(*p.URL)).First(&existing).Error
	}

	return existing, db.Where(
		"username = ? AND domain = ?",
		strings.TrimSpace(p.Username),
		strings.TrimSpace(*p.Domain),
	).First(&existing).Error
}

func mergeRemoteProfile(existing, incoming *Profile) {
	existing.Local = false
	if strings.TrimSpace(incoming.Username) != "" {
		existing.Username = strings.TrimSpace(incoming.Username)
	}
	if strings.TrimSpace(incoming.DisplayName) != "" {
		existing.DisplayName = strings.TrimSpace(incoming.DisplayName)
	}
	mergeOptionalString(&existing.Domain, incoming.Domain)
	mergeOptionalString(&existing.URL, incoming.URL)
	mergeOptionalString(&existing.InboxURL, incoming.InboxURL)
	mergeOptionalString(&existing.OutboxURL, incoming.OutboxURL)
	mergeOptionalString(&existing.FollowersURL, incoming.FollowersURL)
	mergeOptionalString(&existing.AvatarRemoteURL, incoming.AvatarRemoteURL)
}

func mergeOptionalString(dst **string, src *string) {
	if src == nil {
		return
	}

	normalized := normalizeOptionalString(src)
	if normalized == nil {
		return
	}

	*dst = normalized
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
