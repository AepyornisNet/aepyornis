package model

import "gorm.io/gorm"

type WorkoutFile struct {
	Model
	Filename  string `json:"filename"`                              // The filename of the file
	Content   []byte `gorm:"type:bytes" json:"content"`             // The file content
	Checksum  []byte `gorm:"not null;uniqueIndex" json:"checksum"`  // The checksum of the content
	WorkoutID uint64 `gorm:"not null;uniqueIndex" json:"workoutID"` // The ID of the workout
}

func (d *WorkoutFile) Save(db *gorm.DB) error {
	if d.Content == nil {
		return ErrInvalidData
	}

	return db.Save(d).Error
}
