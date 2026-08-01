package repository

import (
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/samber/do/v2"
	"gorm.io/gorm"
)

type Workout interface {
	GetByProfileID(userID uint64, id uint64) (*model.Workout, error)
	ListByProfileID(userID uint64) ([]*model.Workout, error)
	ListByProfileIDWithDetails(userID uint64) ([]*model.Workout, error)
	CountByProfileAndFilters(userID uint64, filters *model.WorkoutFilters) (int64, error)
	ListByProfileAndFilters(userID uint64, filters *model.WorkoutFilters, limit int, offset int) ([]*model.Workout, error)
	GetByIDForRead(id uint64, withRouteSegmentMatches bool) (*model.Workout, error)
	GetDetailsByID(id uint64) (*model.Workout, error)
}

type workoutRepository struct {
	db *gorm.DB
}

func NewWorkout(injector do.Injector) (Workout, error) {
	return &workoutRepository{db: do.MustInvoke[*gorm.DB](injector)}, nil
}

func (r *workoutRepository) GetByProfileID(profileID uint64, id uint64) (*model.Workout, error) {
	var workout model.Workout

	q := model.PreloadWorkoutDetails(r.db).Preload("File").Preload("Equipment")
	if err := q.Where(&model.Workout{ProfileID: profileID}).First(&workout, id).Error; err != nil {
		return nil, err
	}

	return &workout, nil
}

func (r *workoutRepository) ListByProfileID(profileID uint64) ([]*model.Workout, error) {
	var workouts []*model.Workout

	if err := model.PreloadWorkoutData(r.db).Where(&model.Workout{ProfileID: profileID}).Order("date DESC").Find(&workouts).Error; err != nil {
		return nil, err
	}

	return workouts, nil
}

func (r *workoutRepository) ListByProfileIDWithDetails(profileID uint64) ([]*model.Workout, error) {
	var workouts []*model.Workout

	if err := model.PreloadWorkoutDetails(r.db).Where(&model.Workout{ProfileID: profileID}).Order("date DESC").Find(&workouts).Error; err != nil {
		return nil, err
	}

	return workouts, nil
}

func (r *workoutRepository) CountByProfileAndFilters(profileID uint64, filters *model.WorkoutFilters) (int64, error) {
	var totalCount int64

	q := r.db.Model(&model.Workout{}).Where("profile_id = ?", profileID)
	if filters != nil {
		q = filters.ToQuery(q)
	}

	if err := q.Select("COUNT(workouts.id)").Count(&totalCount).Error; err != nil {
		return 0, err
	}

	return totalCount, nil
}

func (r *workoutRepository) ListByProfileAndFilters(profileID uint64, filters *model.WorkoutFilters, limit int, offset int) ([]*model.Workout, error) {
	var workouts []*model.Workout

	q := r.db.Model(&model.Workout{})
	if filters != nil {
		q = filters.ToQuery(q)
	}

	q = model.PreloadWorkoutData(q).
		Preload("File").
		Where("profile_id = ?", profileID).
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
		Preload("File").
		Preload("Equipment").
		Preload("Profile").
		Preload("Profile.User")

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

	if err := model.PreloadWorkoutDetails(r.db).Preload("File").Preload("Profile").First(&workout, id).Error; err != nil {
		return nil, err
	}

	return &workout, nil
}
