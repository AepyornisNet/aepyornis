package model

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const APOutboxWorkoutKind = "workout"

type APOutboxEntry struct {
	Model

	PublicUUID uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"public_uuid"`

	UserID uint64 `gorm:"index:idx_ap_outbox_user_published;not null" json:"user_id"`
	User   *User  `json:"-"`

	APOutboxWorkoutID *uint64          `gorm:"index:idx_ap_outbox_workout" json:"ap_outbox_workout_id,omitempty"`
	APOutboxWorkout   *APOutboxWorkout `gorm:"constraint:OnDelete:CASCADE" json:"-"`

	Kind string `gorm:"type:varchar(64);index;not null" json:"kind"`

	ActivityID string         `gorm:"type:text;uniqueIndex;not null" json:"activity_id"`
	ObjectID   string         `gorm:"type:text;uniqueIndex;not null" json:"object_id"`
	Activity   datatypes.JSON `gorm:"type:json;not null" json:"activity"`
	Payload    datatypes.JSON `gorm:"type:json" json:"payload"`
	NoteText   string         `gorm:"type:text" json:"note_text"`

	PublishedAt time.Time `gorm:"index:idx_ap_outbox_user_published;not null" json:"published_at"`
}

type APOutboxWorkout struct {
	Model

	UserID uint64 `gorm:"index:idx_ap_outbox_workout_user_workout;not null" json:"user_id"`
	User   *User  `json:"-"`

	WorkoutID uint64   `gorm:"uniqueIndex:idx_ap_outbox_workout_user_workout;not null" json:"workout_id"`
	Workout   *Workout `gorm:"constraint:OnDelete:CASCADE" json:"-"`

	FitFilename    string `gorm:"type:varchar(255);not null" json:"fit_filename"`
	FitContent     []byte `gorm:"type:bytes;not null" json:"-"`
	FitChecksum    []byte `gorm:"type:bytes;not null" json:"-"`
	FitContentType string `gorm:"type:varchar(128);not null;default:application/vnd.ant.fit" json:"fit_content_type"`

	RouteImageFilename    string `gorm:"type:varchar(255)" json:"route_image_filename,omitempty"`
	RouteImageContent     []byte `gorm:"type:bytes" json:"-"`
	RouteImageChecksum    []byte `gorm:"type:bytes" json:"-"`
	RouteImageContentType string `gorm:"type:varchar(128);default:image/png" json:"route_image_content_type,omitempty"`
}

func (APOutboxEntry) TableName() string {
	return "ap_outbox"
}

func (APOutboxWorkout) TableName() string {
	return "ap_outbox_workout"
}

func (e *APOutboxEntry) BeforeCreate(_ *gorm.DB) error {
	if e.PublicUUID == uuid.Nil {
		e.PublicUUID = uuid.New()
	}

	if e.PublishedAt.IsZero() {
		e.PublishedAt = time.Now().UTC()
	}

	return nil
}

func (w *APOutboxWorkout) BeforeCreate(_ *gorm.DB) error {
	if len(w.FitContent) > 0 {
		h := sha256.Sum256(w.FitContent)
		w.FitChecksum = h[:]
	}

	if len(w.RouteImageContent) > 0 {
		h := sha256.Sum256(w.RouteImageContent)
		w.RouteImageChecksum = h[:]
	}

	if w.FitContentType == "" {
		w.FitContentType = "application/vnd.ant.fit"
	}

	if len(w.RouteImageContent) > 0 && w.RouteImageContentType == "" {
		w.RouteImageContentType = "image/png"
	}

	return nil
}

func CreateAPOutboxWorkout(db *gorm.DB, outboxWorkout *APOutboxWorkout) error {
	if outboxWorkout == nil {
		return errors.New("outbox workout is nil")
	}

	if outboxWorkout.UserID == 0 || outboxWorkout.WorkoutID == 0 {
		return errors.New("outbox workout user_id and workout_id are required")
	}

	if len(outboxWorkout.FitContent) == 0 {
		return errors.New("outbox workout fit content is required")
	}

	return db.Create(outboxWorkout).Error
}

func CreateAPOutboxEntry(db *gorm.DB, entry *APOutboxEntry) error {
	if entry == nil {
		return errors.New("outbox entry is nil")
	}

	if entry.ActivityID == "" || entry.ObjectID == "" {
		return errors.New("outbox entry IDs are required")
	}

	if !json.Valid(entry.Activity) {
		return errors.New("outbox activity payload is invalid JSON")
	}

	if len(entry.Payload) > 0 && !json.Valid(entry.Payload) {
		return errors.New("outbox object payload is invalid JSON")
	}

	return db.Create(entry).Error
}

func CountAPOutboxEntriesByUser(db *gorm.DB, userID uint64) (int64, error) {
	var total int64
	if err := db.Model(&APOutboxEntry{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return 0, err
	}

	return total, nil
}

func GetAPOutboxEntriesByUser(db *gorm.DB, userID uint64, limit int, offset int) ([]APOutboxEntry, error) {
	entries := make([]APOutboxEntry, 0)
	if limit <= 0 {
		limit = 20
	}

	err := db.
		Where("user_id = ?", userID).
		Order("published_at DESC").
		Order("id DESC").
		Limit(limit).
		Offset(offset).
		Find(&entries).
		Error

	return entries, err
}

func GetAPOutboxEntryByUUIDAndUser(db *gorm.DB, userID uint64, outboxID uuid.UUID) (*APOutboxEntry, error) {
	entry := &APOutboxEntry{}
	if err := db.
		Preload("APOutboxWorkout").
		Where("user_id = ? AND public_uuid = ?", userID, outboxID).
		First(entry).
		Error; err != nil {
		return nil, err
	}

	return entry, nil
}

func GetAPOutboxEntryForWorkout(db *gorm.DB, userID uint64, workoutID uint64) (*APOutboxEntry, error) {
	entry := &APOutboxEntry{}
	if err := db.Model(&APOutboxEntry{}).
		Joins("JOIN ap_outbox_workout ON ap_outbox_workout.id = ap_outbox.ap_outbox_workout_id").
		Where("ap_outbox.user_id = ?", userID).
		Where("ap_outbox_workout.workout_id = ?", workoutID).
		First(entry).Error; err != nil {
		return nil, err
	}

	return entry, nil
}

func DeleteAPOutboxEntryForWorkout(db *gorm.DB, userID uint64, workoutID uint64) error {
	outboxWorkout := &APOutboxWorkout{}
	if err := db.Where("user_id = ? AND workout_id = ?", userID, workoutID).First(outboxWorkout).Error; err != nil {
		return err
	}

	if err := db.Delete(outboxWorkout).Error; err != nil {
		return err
	}

	return nil
}

func APOutboxPublishedMap(db *gorm.DB, userID uint64, workoutIDs []uint64) (map[uint64]bool, error) {
	published := map[uint64]bool{}
	if len(workoutIDs) == 0 {
		return published, nil
	}

	type row struct {
		WorkoutID uint64
	}

	rows := make([]row, 0, len(workoutIDs))
	if err := db.Model(&APOutboxEntry{}).
		Select("ap_outbox_workout.workout_id").
		Joins("JOIN ap_outbox_workout ON ap_outbox_workout.id = ap_outbox.ap_outbox_workout_id").
		Where("ap_outbox.user_id = ?", userID).
		Where("ap_outbox_workout.workout_id IN ?", workoutIDs).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	for _, r := range rows {
		published[r.WorkoutID] = true
	}

	return published, nil
}
