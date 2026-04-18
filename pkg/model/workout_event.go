package model

import (
	"time"

	"gorm.io/datatypes"
)

// WorkoutEvent stores normalized activity event markers (e.g. FIT timer start/stop events).
type WorkoutEvent struct {
	WorkoutID uint64 `gorm:"not null;primaryKey;index:idx_workout_events_parent_order,unique" json:"-"`
	SortOrder int    `gorm:"not null;primaryKey;index:idx_workout_events_parent_order,unique" json:"-"`

	Timestamp      time.Time      `gorm:"index" json:"timestamp"`
	StartTimestamp time.Time      `json:"startTimestamp"`
	Event          string         `gorm:"size:64;index" json:"event"`
	EventType      string         `gorm:"size:64;index" json:"eventType"`
	EventGroup     uint8          `json:"eventGroup"`
	Payload        datatypes.JSON `gorm:"type:jsonb" json:"payload,omitempty"`
}
