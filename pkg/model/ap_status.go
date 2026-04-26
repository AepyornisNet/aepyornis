package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	APStatusTypeWorkout = "workout"
	APStatusTypeReply   = "reply"
)

type APStatus struct {
	Model

	PublicUUID uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"public_uuid"`

	ProfileID *uint64  `gorm:"index:idx_ap_statuses_profile_published" json:"profile_id,omitempty"`
	Profile   *Profile `gorm:"foreignKey:ProfileID;constraint:OnDelete:CASCADE" json:"-"`

	// Link to local outbox workout attachment row for local workout posts.
	APStatusWorkoutID *uint64          `gorm:"index" json:"ap_status_workout_id,omitempty"`
	APStatusWorkout   *APStatusWorkout `gorm:"constraint:OnDelete:SET NULL" json:"-"`

	StatusType string `gorm:"type:varchar(64);index;not null" json:"status_type"`
	Origin     string `gorm:"type:varchar(16);index;not null;default:local" json:"origin"`

	ActivityID string `gorm:"type:text;uniqueIndex;not null" json:"activity_id"`
	ObjectID   string `gorm:"type:text;uniqueIndex;not null" json:"object_id"`

	InReplyToObjectID *string `gorm:"type:text;index" json:"in_reply_to_object_id,omitempty"`

	Activity datatypes.JSON `gorm:"type:json;not null" json:"activity"`
	Payload  datatypes.JSON `gorm:"type:json" json:"payload"`

	Content string  `gorm:"type:text" json:"content"`
	Summary *string `gorm:"type:text" json:"summary,omitempty"`

	To  datatypes.JSON `gorm:"type:json" json:"to,omitempty"`
	CC  datatypes.JSON `gorm:"type:json" json:"cc,omitempty"`
	BCC datatypes.JSON `gorm:"type:json" json:"bcc,omitempty"`

	PublishedAt *time.Time `gorm:"index:idx_ap_statuses_profile_published" json:"published_at,omitempty"`
}

func (APStatus) TableName() string {
	return "ap_statuses"
}

func (s *APStatus) BeforeCreate(_ *gorm.DB) error {
	if s.PublicUUID == uuid.Nil {
		s.PublicUUID = uuid.New()
	}

	if s.PublishedAt == nil {
		now := time.Now().UTC()
		s.PublishedAt = &now
	}

	if s.Origin == "" {
		s.Origin = "local"
	}

	return nil
}
