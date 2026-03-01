package model

import (
	"time"

	"gorm.io/datatypes"
)

// WorkoutReply represents a reply to a workout post
// This can be from a local user or a remote ActivityPub actor
type WorkoutReply struct {
	Model

	WorkoutID uint64   `gorm:"index:idx_workout_reply_workout_actor,unique;not null" json:"workout_id"`
	Workout   *Workout `gorm:"constraint:OnDelete:CASCADE" json:"-"`

	// The IRI of the reply object (from remote actors) or local reference
	ObjectIRI string `gorm:"type:text;index:idx_workout_reply_workout_actor,unique;not null" json:"object_iri"`

	// For local replies, track the user who created it
	UserID *uint64 `gorm:"index" json:"user_id,omitempty"`
	User   *User   `gorm:"constraint:OnDelete:CASCADE" json:"-"`

	// For remote replies, store the actor IRI
	ActorIRI *string `gorm:"type:text;index" json:"actor_iri,omitempty"`

	// Actor name from remote server
	ActorName *string `gorm:"type:varchar(255)" json:"actor_name,omitempty"`

	// The content/text of the reply
	Content string `gorm:"type:text" json:"content"`

	// Summary or display name of the reply if available
	Summary *string `gorm:"type:text" json:"summary,omitempty"`

	// The full ActivityPub object of the reply (for display and re-delivery)
	Object datatypes.JSON `gorm:"type:json" json:"object,omitempty"`

	// When the reply was created on the remote server
	PublishedAt *time.Time `gorm:"index" json:"published_at,omitempty"`

	// Audience: who can read this reply (JSON array of IRIs)
	To  datatypes.JSON `gorm:"type:json" json:"to,omitempty"`
	CC  datatypes.JSON `gorm:"type:json" json:"cc,omitempty"`
	BCC datatypes.JSON `gorm:"type:json" json:"bcc,omitempty"`

	// HTTP signatures and other metadata for federated interactions
	InboxURL *string `gorm:"type:text;index" json:"inbox_url,omitempty"`
}

func (WorkoutReply) TableName() string {
	return "workout_replies"
}
