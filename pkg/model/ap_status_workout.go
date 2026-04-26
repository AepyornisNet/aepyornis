package model

import (
	"crypto/sha256"

	"gorm.io/gorm"
)

type APStatusWorkout struct {
	Model

	ProfileID *uint64  `gorm:"index:idx_ap_outbox_workout_profile_workout;uniqueIndex:idx_ap_outbox_workout_profile_workout" json:"profile_id,omitempty"`
	Profile   *Profile `gorm:"foreignKey:ProfileID;constraint:OnDelete:CASCADE" json:"-"`

	WorkoutID uint64   `gorm:"uniqueIndex:idx_ap_outbox_workout_profile_workout;not null" json:"workout_id"`
	Workout   *Workout `gorm:"constraint:OnDelete:CASCADE" json:"-"`

	FitFilename    string `gorm:"type:varchar(255);not null" json:"fit_filename"`
	FitContent     []byte `gorm:"type:bytes;not null" json:"-"`
	FitChecksum    []byte `gorm:"type:bytes;not null" json:"-"`
	FitContentType string `gorm:"type:varchar(128);not null;default:application/vnd.ant.fit" json:"fit_content_type"`
}

func (APStatusWorkout) TableName() string {
	return "ap_outbox_workout"
}

func (w *APStatusWorkout) BeforeCreate(_ *gorm.DB) error {
	if len(w.FitContent) > 0 {
		h := sha256.Sum256(w.FitContent)
		w.FitChecksum = h[:]
	}

	if w.FitContentType == "" {
		w.FitContentType = "application/vnd.ant.fit"
	}

	return nil
}
