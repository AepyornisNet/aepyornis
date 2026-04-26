package model

import (
	"time"
)

type Follower struct {
	Model

	ProfileID uint64   `gorm:"index:idx_followers_profile_id;uniqueIndex:idx_followers_profile_following;not null" json:"profile_id"`
	Profile   *Profile `gorm:"foreignKey:ProfileID;constraint:OnDelete:CASCADE" json:"-"`

	FollowingProfileID uint64   `gorm:"index:idx_followers_following_profile_id;uniqueIndex:idx_followers_profile_following;not null" json:"following_profile_id"`
	FollowingProfile   *Profile `gorm:"foreignKey:FollowingProfileID;constraint:OnDelete:CASCADE" json:"following_profile,omitempty"`

	Approved   bool       `gorm:"default:false;index" json:"approved"`
	ApprovedAt *time.Time `json:"approved_at,omitempty"`
	RejectedAt *time.Time `json:"rejected_at,omitempty"`
}
