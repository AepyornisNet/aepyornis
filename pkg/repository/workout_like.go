package repository

import (
	"errors"

	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WorkoutLike interface {
	LikeByUser(workoutID uint64, userID uint64) error
	LikeByActorIRI(workoutID uint64, actorIRI string) error
	UnlikeByActorIRI(workoutID uint64, actorIRI string) error
	ListByWorkoutID(workoutID uint64) ([]model.WorkoutLike, error)
	CountMapByWorkoutIDs(workoutIDs []uint64) (map[uint64]int64, error)
	LikedMapByUser(workoutIDs []uint64, userID uint64) (map[uint64]bool, error)
}

type workoutLikeRepository struct {
	db *gorm.DB
}

func NewWorkoutLike(db *gorm.DB) WorkoutLike {
	return &workoutLikeRepository{db: db}
}

func (r *workoutLikeRepository) LikeByUser(workoutID uint64, userID uint64) error {
	if workoutID == 0 || userID == 0 {
		return errors.New("workout and user IDs are required")
	}

	like := &model.WorkoutLike{WorkoutID: workoutID, UserID: &userID}
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(like).Error
}

func (r *workoutLikeRepository) LikeByActorIRI(workoutID uint64, actorIRI string) error {
	if workoutID == 0 || actorIRI == "" {
		return errors.New("workout id and actor IRI are required")
	}

	like := &model.WorkoutLike{WorkoutID: workoutID, ActorIRI: &actorIRI}
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(like).Error
}

func (r *workoutLikeRepository) UnlikeByActorIRI(workoutID uint64, actorIRI string) error {
	if workoutID == 0 || actorIRI == "" {
		return errors.New("workout id and actor IRI are required")
	}

	return r.db.Where("workout_id = ? AND actor_iri = ?", workoutID, actorIRI).Delete(&model.WorkoutLike{}).Error
}

func (r *workoutLikeRepository) ListByWorkoutID(workoutID uint64) ([]model.WorkoutLike, error) {
	if workoutID == 0 {
		return nil, errors.New("workout id is required")
	}

	likes := make([]model.WorkoutLike, 0)
	if err := r.db.Preload("User").
		Where("workout_id = ?", workoutID).
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
	if err := r.db.Model(&model.WorkoutLike{}).
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

func (r *workoutLikeRepository) LikedMapByUser(workoutIDs []uint64, userID uint64) (map[uint64]bool, error) {
	liked := map[uint64]bool{}
	if len(workoutIDs) == 0 || userID == 0 {
		return liked, nil
	}

	type row struct {
		WorkoutID uint64
	}

	rows := []row{}
	if err := r.db.Model(&model.WorkoutLike{}).
		Select("workout_id").
		Where("workout_id IN ?", workoutIDs).
		Where("user_id = ?", userID).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		liked[row.WorkoutID] = true
	}

	return liked, nil
}
