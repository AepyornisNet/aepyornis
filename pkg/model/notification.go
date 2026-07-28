package model

import (
	"time"

	"gorm.io/datatypes"
)

type Notification struct {
	Model

	UserID uint64 `gorm:"not null" json:"-"`          // The ID of the user the notification is sent to
	User   *User  `gorm:"foreignKey:UserID" json:"-"` // The user this notification is sent to

	Type   string          `json:"type"`    // The type of notification
	Meta   *datatypes.JSON `json:"meta"`    // Metainfo of the notification
	ReadAt *time.Time      `json:"read_at"` // The time the notification was read at

	Subject string `json:"subject"` // The notification subject
	Msg     string `json:"msg"`     // The notification message
}
