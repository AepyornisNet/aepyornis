package repository

import (
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"gorm.io/gorm"
)

type Workout interface {
	GetByUserID(userID uint64, id uint64) (*model.Workout, error)
	ListByUserID(userID uint64) ([]*model.Workout, error)
	CountByUserAndFilters(userID uint64, filters *model.WorkoutFilters) (int64, error)
	ListByUserAndFilters(userID uint64, filters *model.WorkoutFilters, limit int, offset int) ([]*model.Workout, error)
	GetByIDForRead(id uint64, withRouteSegmentMatches bool) (*model.Workout, error)
	GetDetailsByID(id uint64) (*model.Workout, error)
}

type workoutRepository struct {
	db *gorm.DB
}

func NewWorkout(db *gorm.DB) Workout {
	return &workoutRepository{db: db}
}

func (r *workoutRepository) GetByUserID(userID uint64, id uint64) (*model.Workout, error) {
	var workout model.Workout

	q := model.PreloadWorkoutDetails(r.db).Preload("GPX").Preload("Equipment")
	if err := q.Where(&model.Workout{UserID: userID}).First(&workout, id).Error; err != nil {
		return nil, err
	}

	return &workout, nil
}

func (r *workoutRepository) ListByUserID(userID uint64) ([]*model.Workout, error) {
	var workouts []*model.Workout

	if err := model.PreloadWorkoutData(r.db).Where(&model.Workout{UserID: userID}).Order("date DESC").Find(&workouts).Error; err != nil {
		return nil, err
	}

	return workouts, nil
}

func (r *workoutRepository) CountByUserAndFilters(userID uint64, filters *model.WorkoutFilters) (int64, error) {
	var totalCount int64

	q := r.db.Model(&model.Workout{}).Where("user_id = ?", userID)
	if filters != nil {
		q = filters.ToQuery(q)
	}

	if err := q.Select("COUNT(workouts.id)").Count(&totalCount).Error; err != nil {
		return 0, err
	}

	return totalCount, nil
}

func (r *workoutRepository) ListByUserAndFilters(userID uint64, filters *model.WorkoutFilters, limit int, offset int) ([]*model.Workout, error) {
	var workouts []*model.Workout

	q := r.db.Model(&model.Workout{})
	if filters != nil {
		q = filters.ToQuery(q)
	}

	q = model.PreloadWorkoutData(q).
		Preload("GPX").
		Where("user_id = ?", userID).
		Order("date DESC")

	if limit > 0 {
		q = q.Limit(limit)
	}
	if offset > 0 {
		q = q.Offset(offset)
	}

	if err := q.Find(&workouts).Error; err != nil {
		return nil, err
	}

	return workouts, nil
}

func (r *workoutRepository) GetByIDForRead(id uint64, withRouteSegmentMatches bool) (*model.Workout, error) {
	q := model.PreloadWorkoutDetails(r.db).
		Preload("GPX").
		Preload("Equipment").
		Preload("User").
		Preload("User.Profile")

	if withRouteSegmentMatches {
		q = q.Preload("RouteSegmentMatches.RouteSegment")
	}

	var workout model.Workout
	if err := q.First(&workout, id).Error; err != nil {
		return nil, err
	}

	return &workout, nil
}

func (r *workoutRepository) GetDetailsByID(id uint64) (*model.Workout, error) {
	var workout model.Workout

	if err := model.PreloadWorkoutDetails(r.db).Preload("GPX").First(&workout, id).Error; err != nil {
		return nil, err
	}

	return &workout, nil
}
