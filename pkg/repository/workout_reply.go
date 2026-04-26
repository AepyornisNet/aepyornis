package repository

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/templatehelpers"
	"github.com/samber/do/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WorkoutReply interface {
	ReplyByActorIRI(workoutID uint64, objectIRI, actorIRI, actorName, content string) error
	UpdateReplyByObjectIRI(workoutID uint64, objectIRI, actorIRI, actorName, content string) error
	CreateLocalReply(workoutID, profileID uint64, content string) (*model.APStatus, error)
	DeleteReplyByObjectIRI(workoutID uint64, objectIRI string) error
	ResolveWorkoutIDByObjectIRI(objectIRI string) (uint64, error)
	CountMapByWorkoutIDs(workoutIDs []uint64) (map[uint64]int64, error)
	CountByWorkoutID(workoutID uint64) (int64, error)
	ListByWorkoutID(workoutID uint64, limit int, offset int) ([]model.APStatus, error)
}

type workoutReplyRepository struct {
	db *gorm.DB
}

func NewWorkoutReply(injector do.Injector) (WorkoutReply, error) {
	return &workoutReplyRepository{db: do.MustInvoke[*gorm.DB](injector)}, nil
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
	parentObjectID, err := r.parentObjectIDForWorkout(workoutID)
	if err != nil {
		return err
	}

	profileURL := strings.TrimSpace(actorIRI)
	profile, err := (&model.Profile{
		DisplayName: strings.TrimSpace(actorName),
		URL:         &profileURL,
	}).UpsertRemote(r.db)
	if err != nil {
		return err
	}

	reply := &model.APStatus{
		ObjectID:          objectIRI,
		ActivityID:        objectIRI,
		ProfileID:         &profile.ID,
		Activity:          []byte("{}"),
		Content:           content,
		StatusType:        model.APStatusTypeReply,
		Origin:            "remote",
		InReplyToObjectID: &parentObjectID,
	}

	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "object_id"}},
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
	parentObjectID, err := r.parentObjectIDForWorkout(workoutID)
	if err != nil {
		return err
	}

	profileURL := strings.TrimSpace(actorIRI)
	profile, err := (&model.Profile{
		DisplayName: strings.TrimSpace(actorName),
		URL:         &profileURL,
	}).UpsertRemote(r.db)
	if err != nil {
		return err
	}

	updates := map[string]any{"content": content}

	result := r.db.Model(&model.APStatus{}).
		Where("status_type = ?", model.APStatusTypeReply).
		Where("object_id = ? AND profile_id = ? AND in_reply_to_object_id = ?", objectIRI, profile.ID, parentObjectID).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *workoutReplyRepository) CreateLocalReply(workoutID, profileID uint64, content string) (*model.APStatus, error) {
	if workoutID == 0 {
		return nil, errors.New("workout id is required")
	}
	if profileID == 0 {
		return nil, errors.New("profile id is required")
	}
	if content == "" {
		return nil, errors.New("content is required")
	}

	content = templatehelpers.SanitizeReplyHTML(content)
	parentObjectID, err := r.parentObjectIDForWorkout(workoutID)
	if err != nil {
		return nil, err
	}

	objectIRI := "local:" + strconv.FormatUint(workoutID, 10) + ":" + strconv.FormatUint(profileID, 10) + ":" + strconv.FormatInt(time.Now().UnixNano(), 10)

	reply := &model.APStatus{
		ObjectID:          objectIRI,
		ActivityID:        objectIRI,
		ProfileID:         &profileID,
		Activity:          []byte("{}"),
		Content:           content,
		StatusType:        model.APStatusTypeReply,
		Origin:            "local",
		InReplyToObjectID: &parentObjectID,
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

	parentObjectID, err := r.parentObjectIDForWorkout(workoutID)
	if err != nil {
		return err
	}

	return r.db.Where("status_type = ? AND object_id = ? AND in_reply_to_object_id = ?", model.APStatusTypeReply, objectIRI, parentObjectID).Delete(&model.APStatus{}).Error
}

func (r *workoutReplyRepository) ResolveWorkoutIDByObjectIRI(objectIRI string) (uint64, error) {
	if objectIRI == "" {
		return 0, errors.New("object iri is required")
	}

	type row struct{ WorkoutID uint64 }
	found := &row{}
	if err := r.db.Table("ap_statuses AS reply").
		Select("ap_outbox_workout.workout_id AS workout_id").
		Joins("JOIN ap_statuses AS parent ON parent.object_id = reply.in_reply_to_object_id").
		Joins("JOIN ap_outbox_workout ON ap_outbox_workout.id = parent.ap_status_workout_id").
		Where("reply.status_type = ?", model.APStatusTypeReply).
		Where("reply.object_id = ?", objectIRI).
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
	if err := r.db.Table("ap_statuses AS reply").
		Select("ap_outbox_workout.workout_id AS workout_id, COUNT(reply.id) as total").
		Joins("JOIN ap_statuses AS parent ON parent.object_id = reply.in_reply_to_object_id").
		Joins("JOIN ap_outbox_workout ON ap_outbox_workout.id = parent.ap_status_workout_id").
		Where("reply.status_type = ?", model.APStatusTypeReply).
		Where("ap_outbox_workout.workout_id IN ?", workoutIDs).
		Group("ap_outbox_workout.workout_id").
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
	if err := r.db.Table("ap_statuses AS reply").
		Joins("JOIN ap_statuses AS parent ON parent.object_id = reply.in_reply_to_object_id").
		Joins("JOIN ap_outbox_workout ON ap_outbox_workout.id = parent.ap_status_workout_id").
		Where("reply.status_type = ?", model.APStatusTypeReply).
		Where("ap_outbox_workout.workout_id = ?", workoutID).
		Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *workoutReplyRepository) ListByWorkoutID(workoutID uint64, limit int, offset int) ([]model.APStatus, error) {
	if workoutID == 0 {
		return nil, errors.New("workout id is required")
	}
	if limit <= 0 {
		limit = 20
	}

	replies := make([]model.APStatus, 0)
	if err := r.db.Model(&model.APStatus{}).
		Preload("Profile").
		Preload("Profile.User").
		Table("ap_statuses AS reply").
		Select("reply.*").
		Joins("JOIN ap_statuses AS parent ON parent.object_id = reply.in_reply_to_object_id").
		Joins("JOIN ap_outbox_workout ON ap_outbox_workout.id = parent.ap_status_workout_id").
		Where("reply.status_type = ?", model.APStatusTypeReply).
		Where("ap_outbox_workout.workout_id = ?", workoutID).
		Order("COALESCE(reply.published_at, reply.created_at) DESC, reply.id DESC").
		Limit(limit).
		Offset(offset).
		Find(&replies).Error; err != nil {
		return nil, err
	}

	return replies, nil
}

func (r *workoutReplyRepository) parentObjectIDForWorkout(workoutID uint64) (string, error) {
	type row struct{ ObjectID string }
	found := &row{}
	if err := r.db.Table("ap_statuses").
		Select("ap_statuses.object_id").
		Joins("JOIN ap_outbox_workout ON ap_outbox_workout.id = ap_statuses.ap_status_workout_id").
		Where("ap_statuses.status_type = ?", model.APStatusTypeWorkout).
		Where("ap_outbox_workout.workout_id = ?", workoutID).
		Take(found).Error; err != nil {
		return "", err
	}

	return found.ObjectID, nil
}
