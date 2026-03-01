package repository

import (
	"errors"
	"strconv"
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/jovandeginste/workout-tracker/v2/pkg/templatehelpers"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WorkoutReply interface {
	ReplyByActorIRI(workoutID uint64, objectIRI, actorIRI, actorName, content string) error
	UpdateReplyByObjectIRI(workoutID uint64, objectIRI, actorIRI, actorName, content string) error
	CreateLocalReply(workoutID, userID uint64, content string) (*model.WorkoutReply, error)
	DeleteReplyByObjectIRI(workoutID uint64, objectIRI string) error
	ResolveWorkoutIDByObjectIRI(objectIRI string) (uint64, error)
	CountMapByWorkoutIDs(workoutIDs []uint64) (map[uint64]int64, error)
	CountByWorkoutID(workoutID uint64) (int64, error)
	ListByWorkoutID(workoutID uint64, limit int, offset int) ([]model.WorkoutReply, error)
}

type workoutReplyRepository struct {
	db *gorm.DB
}

func NewWorkoutReply(db *gorm.DB) WorkoutReply {
	return &workoutReplyRepository{db: db}
}

func (r *workoutReplyRepository) ReplyByActorIRI(workoutID uint64, objectIRI, actorIRI, actorName, content string) error {
	if workoutID == 0 {
		return errors.New("workout id is required")
	}
	if objectIRI == "" {
		return errors.New("object iri is required")
	}
	if actorIRI == "" {
		return errors.New("actor iri is required")
	}
	if content == "" {
		return errors.New("content is required")
	}

	content = templatehelpers.SanitizeReplyHTML(content)

	reply := &model.WorkoutReply{
		WorkoutID: workoutID,
		ObjectIRI: objectIRI,
		ActorIRI:  &actorIRI,
		Content:   content,
	}

	if actorName != "" {
		reply.ActorName = &actorName
	}

	// Use upsert to handle duplicates (same actor/object combination)
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "workout_id"}, {Name: "object_iri"}},
		DoNothing: true,
	}).Create(reply).Error
}

func (r *workoutReplyRepository) UpdateReplyByObjectIRI(workoutID uint64, objectIRI, actorIRI, actorName, content string) error {
	if workoutID == 0 {
		return errors.New("workout id is required")
	}
	if objectIRI == "" {
		return errors.New("object iri is required")
	}
	if actorIRI == "" {
		return errors.New("actor iri is required")
	}
	if content == "" {
		return errors.New("content is required")
	}

	content = templatehelpers.SanitizeReplyHTML(content)

	updates := map[string]any{
		"content": content,
	}
	if actorName != "" {
		updates["actor_name"] = actorName
	}

	result := r.db.Model(&model.WorkoutReply{}).
		Where("workout_id = ? AND object_iri = ? AND actor_iri = ?", workoutID, objectIRI, actorIRI).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *workoutReplyRepository) CreateLocalReply(workoutID, userID uint64, content string) (*model.WorkoutReply, error) {
	if workoutID == 0 {
		return nil, errors.New("workout id is required")
	}
	if userID == 0 {
		return nil, errors.New("user id is required")
	}
	if content == "" {
		return nil, errors.New("content is required")
	}

	content = templatehelpers.SanitizeReplyHTML(content)

	// Generate a local object IRI for this reply
	objectIRI := "local:" + strconv.FormatUint(workoutID, 10) + ":" + strconv.FormatUint(userID, 10) + ":" + strconv.FormatInt(time.Now().UnixNano(), 10)

	reply := &model.WorkoutReply{
		WorkoutID: workoutID,
		ObjectIRI: objectIRI,
		UserID:    &userID,
		Content:   content,
	}

	if err := r.db.Create(reply).Error; err != nil {
		return nil, err
	}

	return reply, nil
}

func (r *workoutReplyRepository) DeleteReplyByObjectIRI(workoutID uint64, objectIRI string) error {
	if workoutID == 0 {
		return errors.New("workout id is required")
	}
	if objectIRI == "" {
		return errors.New("object iri is required")
	}

	return r.db.Where("workout_id = ? AND object_iri = ?", workoutID, objectIRI).Delete(&model.WorkoutReply{}).Error
}

func (r *workoutReplyRepository) ResolveWorkoutIDByObjectIRI(objectIRI string) (uint64, error) {
	if objectIRI == "" {
		return 0, errors.New("object iri is required")
	}

	type row struct {
		WorkoutID uint64
	}

	found := &row{}
	if err := r.db.Model(&model.WorkoutReply{}).
		Select("workout_id").
		Where("object_iri = ?", objectIRI).
		Take(found).Error; err != nil {
		return 0, err
	}

	return found.WorkoutID, nil
}

func (r *workoutReplyRepository) CountMapByWorkoutIDs(workoutIDs []uint64) (map[uint64]int64, error) {
	counts := map[uint64]int64{}
	if len(workoutIDs) == 0 {
		return counts, nil
	}

	type row struct {
		WorkoutID uint64
		Total     int64
	}

	rows := []row{}
	if err := r.db.Model(&model.WorkoutReply{}).
		Select("workout_id, COUNT(id) as total").
		Where("workout_id IN ?", workoutIDs).
		Group("workout_id").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		counts[row.WorkoutID] = row.Total
	}

	return counts, nil
}

func (r *workoutReplyRepository) CountByWorkoutID(workoutID uint64) (int64, error) {
	if workoutID == 0 {
		return 0, errors.New("workout id is required")
	}

	var count int64
	if err := r.db.Model(&model.WorkoutReply{}).Where("workout_id = ?", workoutID).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *workoutReplyRepository) ListByWorkoutID(workoutID uint64, limit int, offset int) ([]model.WorkoutReply, error) {
	if workoutID == 0 {
		return nil, errors.New("workout id is required")
	}
	if limit <= 0 {
		limit = 20
	}

	replies := make([]model.WorkoutReply, 0)
	if err := r.db.Preload("User").
		Where("workout_id = ?", workoutID).
		Order("COALESCE(published_at, created_at) DESC, id DESC").
		Limit(limit).
		Offset(offset).
		Find(&replies).Error; err != nil {
		return nil, err
	}

	return replies, nil
}
