package model

import (
	"crypto/sha256"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const APOutboxWorkoutKind = "workout"

type APOutboxEntry struct {
	Model

	PublicUUID uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"public_uuid"`

	UserID uint64 `gorm:"index:idx_ap_outbox_user_published;not null" json:"user_id"`
	User   *User  `json:"-"`

	APOutboxWorkoutID *uint64          `gorm:"index:idx_ap_outbox_workout" json:"ap_outbox_workout_id,omitempty"`
	APOutboxWorkout   *APOutboxWorkout `gorm:"constraint:OnDelete:CASCADE" json:"-"`

	Kind string `gorm:"type:varchar(64);index;not null" json:"kind"`

	ActivityID string         `gorm:"type:text;uniqueIndex;not null" json:"activity_id"`
	ObjectID   string         `gorm:"type:text;uniqueIndex;not null" json:"object_id"`
	Activity   datatypes.JSON `gorm:"type:json;not null" json:"activity"`
	Payload    datatypes.JSON `gorm:"type:json" json:"payload"`
	NoteText   string         `gorm:"type:text" json:"note_text"`

	PublishedAt time.Time `gorm:"index:idx_ap_outbox_user_published;not null" json:"published_at"`
}

type APOutboxWorkout struct {
	Model

	UserID uint64 `gorm:"index:idx_ap_outbox_workout_user_workout;not null" json:"user_id"`
	User   *User  `json:"-"`

	WorkoutID uint64   `gorm:"uniqueIndex:idx_ap_outbox_workout_user_workout;not null" json:"workout_id"`
	Workout   *Workout `gorm:"constraint:OnDelete:CASCADE" json:"-"`

	FitFilename    string `gorm:"type:varchar(255);not null" json:"fit_filename"`
	FitContent     []byte `gorm:"type:bytes;not null" json:"-"`
	FitChecksum    []byte `gorm:"type:bytes;not null" json:"-"`
	FitContentType string `gorm:"type:varchar(128);not null;default:application/vnd.ant.fit" json:"fit_content_type"`

	RouteImageFilename    string `gorm:"type:varchar(255)" json:"route_image_filename,omitempty"`
	RouteImageContent     []byte `gorm:"type:bytes" json:"-"`
	RouteImageChecksum    []byte `gorm:"type:bytes" json:"-"`
	RouteImageContentType string `gorm:"type:varchar(128);default:image/png" json:"route_image_content_type,omitempty"`
}

func (APOutboxEntry) TableName() string {
	return "ap_outbox"
}

func (APOutboxWorkout) TableName() string {
	return "ap_outbox_workout"
}

func (e *APOutboxEntry) BeforeCreate(_ *gorm.DB) error {
	if e.PublicUUID == uuid.Nil {
		e.PublicUUID = uuid.New()
	}

	if e.PublishedAt.IsZero() {
		e.PublishedAt = time.Now().UTC()
	}

	return nil
}

func (w *APOutboxWorkout) BeforeCreate(_ *gorm.DB) error {
	if len(w.FitContent) > 0 {
		h := sha256.Sum256(w.FitContent)
		w.FitChecksum = h[:]
	}

	if len(w.RouteImageContent) > 0 {
		h := sha256.Sum256(w.RouteImageContent)
		w.RouteImageChecksum = h[:]
	}

	if w.FitContentType == "" {
		w.FitContentType = "application/vnd.ant.fit"
	}

	if len(w.RouteImageContent) > 0 && w.RouteImageContentType == "" {
		w.RouteImageContentType = "image/png"
	}

	return nil
}
