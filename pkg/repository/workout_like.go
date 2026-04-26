package repository

import (
	"errors"
	"fmt"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/samber/do/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WorkoutLike interface {
	LikeByUser(workoutID uint64, userID uint64) error
	LikeByActorIRI(workoutID uint64, actorIRI string) error
	UnlikeByActorIRI(workoutID uint64, actorIRI string) error
	ListByWorkoutID(workoutID uint64) ([]model.APStatusLike, error)
	CountMapByWorkoutIDs(workoutIDs []uint64) (map[uint64]int64, error)
	LikedMapByUser(workoutIDs []uint64, userID uint64) (map[uint64]bool, error)
}

type workoutLikeRepository struct {
	db *gorm.DB
}

func NewWorkoutLike(injector do.Injector) (WorkoutLike, error) {
	return &workoutLikeRepository{db: do.MustInvoke[*gorm.DB](injector)}, nil
}

func (r *workoutLikeRepository) LikeByUser(workoutID uint64, userID uint64) error {
	if workoutID == 0 || userID == 0 {
		return errors.New("workout and user IDs are required")
	}

	statusID, err := r.workoutStatusID(workoutID)
	if err != nil {
		return err
	}

	like := &model.APStatusLike{StatusID: statusID, UserID: &userID}
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(like).Error
}

func (r *workoutLikeRepository) LikeByActorIRI(workoutID uint64, actorIRI string) error {
	if workoutID == 0 || actorIRI == "" {
		return errors.New("workout id and actor IRI are required")
	}

	statusID, err := r.workoutStatusID(workoutID)
	if err != nil {
		return err
	}

	like := &model.APStatusLike{StatusID: statusID, ActorIRI: &actorIRI}
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(like).Error
}

func (r *workoutLikeRepository) UnlikeByActorIRI(workoutID uint64, actorIRI string) error {
	if workoutID == 0 || actorIRI == "" {
		return errors.New("workout id and actor IRI are required")
	}

	statusID, err := r.workoutStatusID(workoutID)
	if err != nil {
		return err
	}

	return r.db.Where("status_id = ? AND actor_iri = ?", statusID, actorIRI).Delete(&model.APStatusLike{}).Error
}

func (r *workoutLikeRepository) ListByWorkoutID(workoutID uint64) ([]model.APStatusLike, error) {
	if workoutID == 0 {
		return nil, errors.New("workout id is required")
	}

	likes := make([]model.APStatusLike, 0)
	if err := r.db.Preload("User").Preload("User.Profile").
		Joins("JOIN ap_statuses ON ap_statuses.id = ap_status_likes.status_id").
		Joins("JOIN ap_outbox_workout ON ap_outbox_workout.id = ap_statuses.ap_status_workout_id").
		Where("ap_outbox_workout.workout_id = ?", workoutID).
		Order("created_at DESC, id DESC").
		Find(&likes).Error; err != nil {
		return nil, err
	}

	return likes, nil
}

func (r *workoutLikeRepository) CountMapByWorkoutIDs(workoutIDs []uint64) (map[uint64]int64, error) {
	counts := map[uint64]int64{}
	if len(workoutIDs) == 0 {
		return counts, nil
	}

	type row struct {
		WorkoutID uint64
		Total     int64
	}

	rows := []row{}
	if err := r.db.Table("ap_status_likes").
		Select("ap_outbox_workout.workout_id, COUNT(ap_status_likes.id) as total").
		Joins("JOIN ap_statuses ON ap_statuses.id = ap_status_likes.status_id").
		Joins("JOIN ap_outbox_workout ON ap_outbox_workout.id = ap_statuses.ap_status_workout_id").
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

func (r *workoutLikeRepository) LikedMapByUser(workoutIDs []uint64, userID uint64) (map[uint64]bool, error) {
	liked := map[uint64]bool{}
	if len(workoutIDs) == 0 || userID == 0 {
		return liked, nil
	}

	type row struct {
		WorkoutID uint64
	}

	rows := []row{}
	if err := r.db.Table("ap_status_likes").
		Select("ap_outbox_workout.workout_id").
		Joins("JOIN ap_statuses ON ap_statuses.id = ap_status_likes.status_id").
		Joins("JOIN ap_outbox_workout ON ap_outbox_workout.id = ap_statuses.ap_status_workout_id").
		Where("ap_outbox_workout.workout_id IN ?", workoutIDs).
		Where("ap_status_likes.user_id = ?", userID).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		liked[row.WorkoutID] = true
	}

	return liked, nil
}

func (r *workoutLikeRepository) workoutStatusID(workoutID uint64) (uint64, error) {
	type row struct {
		StatusID uint64
	}

	found := &row{}
	if err := r.db.Table("ap_statuses").
		Select("ap_statuses.id AS status_id").
		Joins("JOIN ap_outbox_workout ON ap_outbox_workout.id = ap_statuses.ap_status_workout_id").
		Where("ap_statuses.status_type = ?", model.APStatusTypeWorkout).
		Where("ap_outbox_workout.workout_id = ?", workoutID).
		Take(found).Error; err == nil {
		return found.StatusID, nil
	}

	type workoutOwnerRow struct {
		WorkoutID uint64
		UserID    uint64
	}

	workout := &workoutOwnerRow{}
	if err := r.db.Table("workouts").
		Select("workouts.id AS workout_id, profiles.user_id AS user_id").
		Joins("JOIN profiles ON profiles.id = workouts.profile_id").
		Where("workouts.id = ?", workoutID).
		Take(workout).Error; err != nil {
		return 0, err
	}

	outboxWorkout := &model.APStatusWorkout{
		UserID:      workout.UserID,
		WorkoutID:   workoutID,
		FitFilename: fmt.Sprintf("workout-%d.fit", workoutID),
		FitContent:  []byte("placeholder"),
	}
	if err := r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(outboxWorkout).Error; err != nil {
		return 0, err
	}

	if outboxWorkout.ID == 0 {
		if err := r.db.Where("workout_id = ?", workoutID).Take(outboxWorkout).Error; err != nil {
			return 0, err
		}
	}

	status := &model.APStatus{
		UserID:            &workout.UserID,
		APStatusWorkoutID: &outboxWorkout.ID,
		StatusType:        model.APStatusTypeWorkout,
		Origin:            "local",
		ActivityID:        fmt.Sprintf("local:workout:%d:activity", workoutID),
		ObjectID:          fmt.Sprintf("local:workout:%d:object", workoutID),
		Activity:          []byte("{}"),
		Content:           "",
	}

	if err := r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(status).Error; err != nil {
		return 0, err
	}

	if status.ID == 0 {
		if err := r.db.Where("ap_status_workout_id = ? AND status_type = ?", outboxWorkout.ID, model.APStatusTypeWorkout).Take(status).Error; err != nil {
			return 0, err
		}
	}

	return status.ID, nil
}
