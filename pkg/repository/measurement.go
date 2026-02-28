package repository

import (
	"errors"
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Measurement interface {
	CountByUserID(userID uint64) (int64, error)
	ListByUserID(userID uint64, limit int, offset int) ([]*model.Measurement, error)
	GetByUserIDForDateOrNew(userID uint64, date time.Time) (*model.Measurement, error)
	Save(measurement *model.Measurement) error
	Delete(measurement *model.Measurement) error
}

type measurementRepository struct {
	db *gorm.DB
}

func NewMeasurement(db *gorm.DB) Measurement {
	return &measurementRepository{db: db}
}

func (r *measurementRepository) CountByUserID(userID uint64) (int64, error) {
	var total int64
	if err := r.db.Model(&model.Measurement{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return 0, err
	}

	return total, nil
}

func (r *measurementRepository) ListByUserID(userID uint64, limit int, offset int) ([]*model.Measurement, error) {
	measurements := make([]*model.Measurement, 0)
	q := r.db.Where("user_id = ?", userID).Order("date DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if offset > 0 {
		q = q.Offset(offset)
	}

	if err := q.Find(&measurements).Error; err != nil {
		return nil, err
	}

	return measurements, nil
}

func (r *measurementRepository) GetByUserIDForDateOrNew(userID uint64, date time.Time) (*model.Measurement, error) {
	var measurement model.Measurement

	if err := r.db.Where(&model.Measurement{UserID: userID}).Where("date = ?", datatypes.Date(date.UTC())).First(&measurement).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &model.Measurement{
				UserID: userID,
				Date:   datatypes.Date(date),
			}, nil
		}

		return nil, err
	}

	return &measurement, nil
}

func (r *measurementRepository) Save(measurement *model.Measurement) error {
	return measurement.Save(r.db)
}

func (r *measurementRepository) Delete(measurement *model.Measurement) error {
	return measurement.Delete(r.db)
}
