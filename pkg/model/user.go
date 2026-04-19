package model

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"time"

	gorand "github.com/cat-dealer/go-rand/v2"
	"github.com/invopop/ctxi18n"
	"github.com/invopop/ctxi18n/i18n"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	PasswordMinimumLength = 4
	PasswordMaximumLength = 128
)

var (
	ErrPasswordInvalidLength = errors.New("password has invalid length")
	ErrEmailInvalid          = errors.New("email is not valid")
	ErrNoUser                = errors.New("no user attached")
)

type UserSecrets struct {
	Email    string `gorm:"uniqueIndex;not null;type:varchar" json:"email"` // The user's username
	Password string `gorm:"type:varchar(128);not null"`                     // The user's password as bcrypt hash
	Salt     string `gorm:"type:varchar(16);not null"`                      // The salt used to hash the user's password
	APIKey   string `gorm:"type:varchar(32)"`                               // The user's API key
}

type UserData struct {
	Model
	LastVersion string `gorm:"last_version" json:"lastVersion"` // Which version of the app the user has last seen and acknowledged

	PreferredUnits UserPreferredUnits `gorm:"serializer:json" json:"preferredUnits"` // The user's preferred units

	Language                 string            `json:"language"`                   // The user's preferred language
	Theme                    string            `json:"theme"`                      // The user's preferred color scheme
	TotalsShow               WorkoutType       `json:"totals_show"`                // What workout type of totals to show
	TZ                       string            `json:"timezone"`                   // The user's preferred timezone
	AutoImportDirectory      string            `json:"auto_import_directory"`      // The user's preferred directory for auto-import
	DefaultWorkoutVisibility WorkoutVisibility `json:"default_workout_visibility"` // Default visibility for newly created workouts
	PreferFullDate           bool              `json:"prefer_full_date"`           // Whether to show full dates in the workout details
	APIActive                bool              `json:"api_active"`                 // Whether the user's API key is active

	ActivityPub bool `json:"activity_pub"` // Whether the user has enabled ActivityPub federation
	Active      bool `json:"active"`       // Whether the user is active
	Admin       bool `json:"admin"`        // Whether the user is an admin
}

type User struct {
	db      *gorm.DB
	context context.Context

	UserData
	UserSecrets `swaggerignore:"true"`

	Measurements []Measurement `gorm:"constraint:OnDelete:CASCADE" json:"-"` // The user's measurements

	Profile Profile `gorm:"constraint:OnDelete:CASCADE" json:"profile"` // The user's profile settings

	anonymous bool // Whether we have an actual user or not
}

func (u *User) GetContext() context.Context {
	return u.context
}

func (u *User) SetContext(ctx context.Context) {
	u.context = ctx
}

func (u *User) I18n(message string, vars ...any) string {
	return u.GetTranslator().T(message, vars...)
}

func (u *User) GetTranslator() *i18n.Locale {
	return ctxi18n.Locale(u.context)
}

func AnonymousUser() *User {
	return &User{anonymous: true}
}

func (u *User) ActivityPubEnabled() bool {
	if u == nil {
		return false
	}

	return u.Active && u.ActivityPub
}

func (u *User) IsAnonymous() bool {
	if u == nil {
		return true
	}

	return u.anonymous
}

func (u *User) ShowFullDate() bool {
	if u == nil {
		return false
	}

	return u.PreferFullDate
}

func (u *User) Timezone() *time.Location {
	if u == nil || u.TZ == "" {
		return time.UTC
	}

	loc, err := time.LoadLocation(u.TZ)
	if err != nil {
		return time.UTC
	}

	return loc
}

func (u *User) BeforeSave(_ *gorm.DB) error {
	u.GenerateAPIKey(false)
	u.GenerateSalt()

	return u.IsValid()
}

func (u *User) SetDB(db *gorm.DB) {
	u.db = db
}

func (u *User) IsActive() bool {
	if u == nil {
		return false
	}

	if !u.Active || u.Password == "" || u.Email == "" {
		return false
	}

	return true
}

func (u *User) ValidLogin(password string) bool {
	if !u.IsActive() {
		return false
	}

	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(u.AddSalt(password))) == nil
}

func (u *User) AddSalt(password string) string {
	return u.Salt + password
}

func (u *User) IsValid() error {
	if u.Password == "" {
		return ErrPasswordInvalidLength
	}

	if _, err := mail.ParseAddress(u.Email); err != nil {
		return ErrEmailInvalid
	}

	return nil
}

func (u *User) SetPassword(password string) error {
	if len(password) < PasswordMinimumLength || len(password) > PasswordMaximumLength {
		return ErrPasswordInvalidLength
	}

	u.GenerateSalt()

	cryptedPassword, err := bcrypt.GenerateFromPassword([]byte(u.AddSalt(password)), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	u.Password = string(cryptedPassword)

	return nil
}

func (u *User) Create(db *gorm.DB) error {
	return db.Create(u).Error
}

func (u *User) GenerateAPIKey(force bool) {
	if !force && u.APIKey != "" {
		return
	}

	u.APIKey = gorand.String(32, gorand.GetAlphaNumericPool())
}

func (u *User) GenerateSalt() {
	if u.Salt != "" {
		return
	}

	u.Salt = gorand.String(8, gorand.GetAlphaNumericPool())
}

func (u *User) ResetDefaults() {
	u.Language = DefaultProfileLanguage
	u.Theme = DefaultProfileTheme
	u.TotalsShow = WorkoutTypeRunning
	u.PreferredUnits.SpeedRaw = "km/h"
	u.PreferredUnits.DistanceRaw = "km"
	u.PreferredUnits.ElevationRaw = "m"
	u.PreferredUnits.WeightRaw = "kg"
	u.PreferredUnits.HeightRaw = "cm"
	u.DefaultWorkoutVisibility = WorkoutVisibilityPrivate
}

func (u *User) EffectiveDefaultWorkoutVisibility() WorkoutVisibility {
	if u == nil || !u.DefaultWorkoutVisibility.IsValid() {
		return WorkoutVisibilityPrivate
	}

	return u.DefaultWorkoutVisibility
}

func (u *User) CanImportFromDirectory() (bool, error) {
	if u == nil {
		return false, nil
	}

	if u.AutoImportDirectory == "" {
		return false, nil
	}

	info, err := os.Stat(u.AutoImportDirectory)
	if err != nil {
		return false, err
	}

	if !info.IsDir() {
		return false, fmt.Errorf("%v is not a directory", u.AutoImportDirectory)
	}

	return true, nil
}

func (u *User) Save(db *gorm.DB) error {
	return db.Save(u).Error
}

func (u *User) Delete(db *gorm.DB) error {
	return db.Select(clause.Associations).Delete(u).Error
}

func (u *User) HeightAt(d time.Time) float64 {
	w := u.measurementAt("height", d)
	if w == 0 {
		return 165
	}

	return w
}

func (u *User) WeightAt(d time.Time) float64 {
	w := u.measurementAt("weight", d)
	if w == 0 {
		return 70
	}

	return w
}

func (u *User) FTPAt(d time.Time) float64 {
	val := u.measurementAt("ftp", d)
	if val == 0 {
		return 200
	}

	return val
}

func (u *User) RestingHeartRateAt(d time.Time) float64 {
	val := u.measurementAt("resting_heart_rate", d)
	if val == 0 {
		return 60
	}

	return val
}

func calculateMaxHeartRate(birthdate time.Time, at time.Time) float64 {
	age := at.Year() - birthdate.Year()
	if at.Month() < birthdate.Month() || (at.Month() == birthdate.Month() && at.Day() < birthdate.Day()) {
		age--
	}

	if age < 0 {
		age = 0
	}

	maxHR := 220 - age
	if maxHR <= 0 {
		return 200
	}

	return float64(maxHR)
}

func (u *User) measurementAt(key string, d time.Time) float64 {
	var w float64

	q := u.db.
		Model(&Measurement{}).
		Where(&Measurement{UserID: u.ID}).
		Where("date <= ?", datatypes.Date(d)).
		Where(key+" > ?", 0).
		Order("date DESC").
		Pluck(key, &w)

	if err := q.Error; err != nil {
		return 0
	}

	return w
}

type UserPreferredUnits struct {
	SpeedRaw     string `form:"speed" json:"speed"`         // The user's preferred speed unit
	DistanceRaw  string `form:"distance" json:"distance"`   // The user's preferred distance unit
	ElevationRaw string `form:"elevation" json:"elevation"` // The user's preferred elevation unit
	WeightRaw    string `form:"weight" json:"weight"`       // The user's preferred weight unit
	HeightRaw    string `form:"height" json:"height"`       // The user's preferred height unit
}

func (u UserPreferredUnits) Tempo() string {
	return "min/" + u.Distance()
}

func (u UserPreferredUnits) HeartRate() string {
	return "bpm"
}

func (u UserPreferredUnits) Height() string {
	switch u.HeightRaw {
	case "in":
		return "in"
	default:
		return "cm"
	}
}

func (u UserPreferredUnits) Temperature() string {
	return "°C"
}

func (u UserPreferredUnits) Cadence() string {
	return "spm"
}

func (u UserPreferredUnits) Elevation() string {
	switch u.ElevationRaw {
	case "ft":
		return "ft"
	default:
		return "m"
	}
}

func (u UserPreferredUnits) Weight() string {
	switch u.WeightRaw {
	case "lbs":
		return "lbs"
	default:
		return "kg"
	}
}

func (u UserPreferredUnits) Distance() string {
	switch u.DistanceRaw {
	case "mi":
		return "mi"
	default:
		return "km"
	}
}

func (u UserPreferredUnits) Speed() string {
	switch u.SpeedRaw {
	case "mph":
		return "mph"
	default:
		return "km/h"
	}
}
