package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	APOutboxWorkoutKind     = "workout"
	APOutboxReplyCreateKind = "reply-create"
)

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

func (APOutboxEntry) TableName() string {
	return "ap_outbox"
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
