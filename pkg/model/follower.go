package model

import (
	"time"
)

type FollowerDirection string

const (
	FollowerDirectionIncoming FollowerDirection = "incoming"
	FollowerDirectionOutgoing FollowerDirection = "outgoing"
)

type Follower struct {
	Model

	UserID uint64 `gorm:"index:idx_followers_user_id;uniqueIndex:idx_followers_user_actor_direction;not null" json:"user_id"`
	User   *User  `json:"-"`

	ActorIRI   string            `gorm:"type:text;uniqueIndex:idx_followers_user_actor_direction;not null" json:"actor_iri"`
	ActorInbox string            `gorm:"type:text" json:"actor_inbox"`
	Direction  FollowerDirection `gorm:"type:varchar(16);uniqueIndex:idx_followers_user_actor_direction;not null;default:incoming;index" json:"direction"`
	Approved   bool              `gorm:"default:false;index" json:"approved"`
	ApprovedAt *time.Time        `json:"approved_at,omitempty"`
	RejectedAt *time.Time        `json:"rejected_at,omitempty"`
}
