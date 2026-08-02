package model

import "time"

// UserWebpushSubscription represents a WebPush notification endpoint registered for a user device.
type UserWebpushSubscription struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	UserID    uint64    `gorm:"not null;index" json:"user_id"`
	Endpoint  string    `gorm:"type:text;not null" json:"endpoint"`
	P256dh    string    `gorm:"type:text;not null" json:"-"` // Restrict raw p256dh key from JSON output
	Auth      string    `gorm:"type:text;not null" json:"-"` // Restrict raw auth key from JSON output
	UserAgent string    `gorm:"type:text" json:"user_agent,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
