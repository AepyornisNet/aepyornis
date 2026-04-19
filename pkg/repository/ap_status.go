package repository

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/templatehelpers"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type APStatus interface {
	ResolveWorkoutIDByObjectOrActivityID(userID uint64, objectOrActivityID string) (uint64, error)
	ResolveWorkoutIDByObjectIRI(objectIRI string) (uint64, error)
	ReplyByActorIRI(workoutID uint64, objectIRI, actorIRI, actorName, content string) error
	UpdateReplyByObjectIRI(workoutID uint64, objectIRI, actorIRI, actorName, content string) error
	DeleteReplyByObjectIRI(workoutID uint64, objectIRI string) error
	UpsertRemoteWorkoutStatus(actorIRI, actorName, activityID, objectID, content string, activityJSON, payloadJSON []byte) error
}

type apStatusRepository struct {
	db *gorm.DB
}

func NewAPStatus(db *gorm.DB) APStatus {
	return &apStatusRepository{db: db}
}

func (r *apStatusRepository) ResolveWorkoutIDByObjectOrActivityID(userID uint64, objectOrActivityID string) (uint64, error) {
	type row struct {
		WorkoutID uint64
		ParentID  *string
	}

	if objectOrActivityID == "" {
		return 0, errors.New("object or activity id is required")
	}

	found := &row{}
	q := r.db.Table("ap_statuses").
		Select("ap_outbox_workout.workout_id AS workout_id, ap_statuses.in_reply_to_object_id AS parent_id").
		Joins("LEFT JOIN ap_outbox_workout ON ap_outbox_workout.id = ap_statuses.ap_status_workout_id").
		Where("ap_statuses.object_id = ? OR ap_statuses.activity_id = ?", objectOrActivityID, objectOrActivityID)

	if userID != 0 {
		q = q.Where("user_id = ?", userID)
	}

	if err := q.Take(found).Error; err == nil {
		if found.WorkoutID != 0 {
			return found.WorkoutID, nil
		}

		if found.ParentID != nil && *found.ParentID != "" {
			return r.ResolveWorkoutIDByObjectOrActivityID(userID, *found.ParentID)
		}
	}

	return 0, gorm.ErrRecordNotFound
}

func (r *apStatusRepository) ResolveWorkoutIDByObjectIRI(objectIRI string) (uint64, error) {
	if objectIRI == "" {
		return 0, errors.New("object iri is required")
	}

	type row struct {
		WorkoutID uint64
		ParentID  *string
	}

	found := &row{}
	if err := r.db.Model(&model.APStatus{}).
		Select("in_reply_to_object_id AS parent_id").
		Where("object_id = ?", objectIRI).
		Take(found).Error; err == nil {
		if found.ParentID != nil && *found.ParentID != "" {
			return r.ResolveWorkoutIDByObjectOrActivityID(0, *found.ParentID)
		}
	}

	return 0, gorm.ErrRecordNotFound
}

func (r *apStatusRepository) ReplyByActorIRI(workoutID uint64, objectIRI, actorIRI, actorName, content string) error {
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

	sanitized := templatehelpers.SanitizeReplyHTML(content)
	now := time.Now().UTC()
	parentObjectID, parentErr := r.parentObjectIDForWorkout(workoutID)
	if parentErr != nil {
		return parentErr
	}

	status := &model.APStatus{
		StatusType:        model.APStatusTypeReply,
		Origin:            "remote",
		ActivityID:        objectIRI,
		ObjectID:          objectIRI,
		InReplyToObjectID: &parentObjectID,
		Activity:          []byte("{}"),
		Content:           sanitized,
		ActorIRI:          &actorIRI,
		PublishedAt:       &now,
	}

	if actorName != "" {
		status.ActorName = &actorName
	}

	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "object_id"}},
		DoNothing: true,
	}).Create(status).Error
}

func (r *apStatusRepository) UpdateReplyByObjectIRI(workoutID uint64, objectIRI, actorIRI, actorName, content string) error {
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

	updates := map[string]any{
		"content": templatehelpers.SanitizeReplyHTML(content),
	}
	if actorName != "" {
		updates["actor_name"] = actorName
	}

	result := r.db.Model(&model.APStatus{}).
		Where("status_type = ?", model.APStatusTypeReply).
		Where("object_id = ? AND actor_iri = ?", objectIRI, actorIRI).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *apStatusRepository) DeleteReplyByObjectIRI(workoutID uint64, objectIRI string) error {
	if workoutID == 0 {
		return errors.New("workout id is required")
	}
	if objectIRI == "" {
		return errors.New("object iri is required")
	}

	return r.db.Model(&model.APStatus{}).
		Where("status_type = ?", model.APStatusTypeReply).
		Where("object_id = ?", objectIRI).
		Delete(&model.APStatus{}).Error
}

func (r *apStatusRepository) UpsertRemoteWorkoutStatus(actorIRI, actorName, activityID, objectID, content string, activityJSON, payloadJSON []byte) error {
	if actorIRI == "" {
		return errors.New("actor iri is required")
	}
	if activityID == "" || objectID == "" {
		return errors.New("activity and object ids are required")
	}
	if len(activityJSON) == 0 || !json.Valid(activityJSON) {
		return errors.New("activity payload is invalid")
	}
	if len(payloadJSON) > 0 && !json.Valid(payloadJSON) {
		return errors.New("object payload is invalid")
	}

	now := time.Now().UTC()
	status := &model.APStatus{
		StatusType:  model.APStatusTypeWorkout,
		Origin:      "remote",
		ActivityID:  activityID,
		ObjectID:    objectID,
		Activity:    datatypes.JSON(activityJSON),
		Payload:     datatypes.JSON(payloadJSON),
		Content:     content,
		ActorIRI:    &actorIRI,
		PublishedAt: &now,
	}

	if actorName != "" {
		status.ActorName = &actorName
	}

	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "object_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"activity_id": activityID,
			"activity":    datatypes.JSON(activityJSON),
			"payload":     datatypes.JSON(payloadJSON),
			"content":     content,
			"actor_name":  status.ActorName,
			"updated_at":  time.Now().UTC(),
		}),
	}).Create(status).Error
}

func (r *apStatusRepository) parentObjectIDForWorkout(workoutID uint64) (string, error) {
	if workoutID == 0 {
		return "", errors.New("workout id is required")
	}

	type row struct {
		ObjectID string
	}

	found := &row{}
	if err := r.db.Table("ap_statuses").
		Select("ap_statuses.object_id").
		Joins("JOIN ap_outbox_workout ON ap_outbox_workout.id = ap_statuses.ap_status_workout_id").
		Where("ap_outbox_workout.workout_id = ?", workoutID).
		Where("ap_statuses.status_type = ?", model.APStatusTypeWorkout).
		Take(found).Error; err == nil {
		return found.ObjectID, nil
	}

	return "", gorm.ErrRecordNotFound
}
