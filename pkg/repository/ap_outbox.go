package repository

import (
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"gorm.io/gorm"
)

type APOutbox interface {
	CreateWorkout(outboxWorkout *model.APOutboxWorkout) error
	CreateEntry(entry *model.APOutboxEntry) error
	CountEntriesByUser(userID uint64) (int64, error)
	GetEntriesByUser(userID uint64, limit int, offset int) ([]model.APOutboxEntry, error)
	GetEntryByUUIDAndUser(userID uint64, outboxID uuid.UUID) (*model.APOutboxEntry, error)
	GetEntryForWorkout(userID uint64, workoutID uint64) (*model.APOutboxEntry, error)
	DeleteEntryForWorkout(userID uint64, workoutID uint64) error
	PublishedMap(userID uint64, workoutIDs []uint64) (map[uint64]bool, error)
}

type apOutboxRepository struct {
	db *gorm.DB
}

func NewAPOutbox(db *gorm.DB) APOutbox {
	return &apOutboxRepository{db: db}
}

func (r *apOutboxRepository) CreateWorkout(outboxWorkout *model.APOutboxWorkout) error {
	if outboxWorkout == nil {
		return errors.New("outbox workout is nil")
	}

	if outboxWorkout.UserID == 0 || outboxWorkout.WorkoutID == 0 {
		return errors.New("outbox workout user_id and workout_id are required")
	}

	if len(outboxWorkout.FitContent) == 0 {
		return errors.New("outbox workout fit content is required")
	}

	return r.db.Create(outboxWorkout).Error
}

func (r *apOutboxRepository) CreateEntry(entry *model.APOutboxEntry) error {
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

	return r.db.Create(entry).Error
}

func (r *apOutboxRepository) CountEntriesByUser(userID uint64) (int64, error) {
	var total int64
	if err := r.db.Model(&model.APOutboxEntry{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return 0, err
	}

	return total, nil
}

func (r *apOutboxRepository) GetEntriesByUser(userID uint64, limit int, offset int) ([]model.APOutboxEntry, error) {
	entries := make([]model.APOutboxEntry, 0)
	if limit <= 0 {
		limit = 20
	}

	err := r.db.
		Where("user_id = ?", userID).
		Order("published_at DESC").
		Order("id DESC").
		Limit(limit).
		Offset(offset).
		Find(&entries).
		Error

	return entries, err
}

func (r *apOutboxRepository) GetEntryByUUIDAndUser(userID uint64, outboxID uuid.UUID) (*model.APOutboxEntry, error) {
	entry := &model.APOutboxEntry{}
	if err := r.db.
		Preload("APOutboxWorkout").
		Where("user_id = ? AND public_uuid = ?", userID, outboxID).
		First(entry).
		Error; err != nil {
		return nil, err
	}

	return entry, nil
}

func (r *apOutboxRepository) GetEntryForWorkout(userID uint64, workoutID uint64) (*model.APOutboxEntry, error) {
	entry := &model.APOutboxEntry{}
	if err := r.db.Model(&model.APOutboxEntry{}).
		Joins("JOIN ap_outbox_workout ON ap_outbox_workout.id = ap_outbox.ap_outbox_workout_id").
		Where("ap_outbox.user_id = ?", userID).
		Where("ap_outbox_workout.workout_id = ?", workoutID).
		First(entry).Error; err != nil {
		return nil, err
	}

	return entry, nil
}

func (r *apOutboxRepository) DeleteEntryForWorkout(userID uint64, workoutID uint64) error {
	outboxWorkout := &model.APOutboxWorkout{}
	if err := r.db.Where("user_id = ? AND workout_id = ?", userID, workoutID).First(outboxWorkout).Error; err != nil {
		return err
	}

	if err := r.db.Delete(outboxWorkout).Error; err != nil {
		return err
	}

	return nil
}

func (r *apOutboxRepository) PublishedMap(userID uint64, workoutIDs []uint64) (map[uint64]bool, error) {
	published := map[uint64]bool{}
	if len(workoutIDs) == 0 {
		return published, nil
	}

	type row struct {
		WorkoutID uint64
	}

	rows := make([]row, 0, len(workoutIDs))
	if err := r.db.Model(&model.APOutboxEntry{}).
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
