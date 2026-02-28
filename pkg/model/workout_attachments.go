package model

import (
	"crypto/sha256"
	"errors"

	"gorm.io/gorm"
)

const WorkoutAttachmentKindRouteImage = "route_image"

type WorkoutAttachment struct {
	Model
	WorkoutID   uint64   `gorm:"not null;index:idx_workout_attachments_workout_sort" json:"workout_id"`
	Workout     *Workout `gorm:"constraint:OnDelete:CASCADE" json:"-"`
	Kind        string   `gorm:"type:varchar(64);not null;default:image;index" json:"kind"`
	Filename    string   `gorm:"type:varchar(255);not null" json:"filename"`
	Content     []byte   `gorm:"type:bytes;not null" json:"-"`
	Checksum    []byte   `gorm:"type:bytes;not null" json:"-"`
	ContentType string   `gorm:"type:varchar(128);not null;default:image/png" json:"content_type"`
	SortOrder   int      `gorm:"column:sort_order;not null;default:0;index:idx_workout_attachments_workout_sort" json:"sort_order"`
}

func (WorkoutAttachment) TableName() string {
	return "workout_attachments"
}

func (a *WorkoutAttachment) BeforeSave(_ *gorm.DB) error {
	if len(a.Content) > 0 {
		h := sha256.Sum256(a.Content)
		a.Checksum = h[:]
	}

	if a.Kind == "" {
		a.Kind = "image"
	}

	if a.ContentType == "" {
		a.ContentType = RouteImageMIMEType
	}

	return nil
}

func GetRouteImageAttachment(db *gorm.DB, workoutID uint64) (*WorkoutAttachment, error) {
	var attachment WorkoutAttachment
	if err := db.Where("workout_id = ? AND kind = ?", workoutID, WorkoutAttachmentKindRouteImage).
		Order("sort_order ASC").
		Order("id ASC").
		First(&attachment).Error; err != nil {
		return nil, err
	}

	return &attachment, nil
}

func UpsertRouteImageAttachment(db *gorm.DB, workoutID uint64, filename string, contentType string, content []byte) (*WorkoutAttachment, error) {
	var attachment WorkoutAttachment
	err := db.Where("workout_id = ? AND kind = ?", workoutID, WorkoutAttachmentKindRouteImage).
		Order("sort_order ASC").
		Order("id ASC").
		First(&attachment).Error

	if err == nil {
		attachment.Filename = filename
		attachment.ContentType = contentType
		attachment.Content = content

		if err := db.Save(&attachment).Error; err != nil {
			return nil, err
		}

		return &attachment, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	attachment = WorkoutAttachment{
		WorkoutID:   workoutID,
		Kind:        WorkoutAttachmentKindRouteImage,
		Filename:    filename,
		ContentType: contentType,
		Content:     content,
		SortOrder:   0,
	}

	if err := db.Create(&attachment).Error; err != nil {
		return nil, err
	}

	return &attachment, nil
}
