package repository

import (
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"gorm.io/gorm"
)

type Equipment interface {
	GetByUserID(userID uint64, id uint64) (*model.Equipment, error)
	GetByUserIDs(userID uint64, ids []uint64) ([]*model.Equipment, error)
	CountByUserID(userID uint64) (int64, error)
	ListByUserID(userID uint64, limit int, offset int) ([]*model.Equipment, error)
	Save(e *model.Equipment) error
	Delete(e *model.Equipment) error
}

type equipmentRepository struct {
	db *gorm.DB
}

func NewEquipment(db *gorm.DB) Equipment {
	return &equipmentRepository{db: db}
}

func (r *equipmentRepository) GetByUserID(userID uint64, id uint64) (*model.Equipment, error) {
	var equipment model.Equipment
	if err := r.db.Preload("Workouts").Preload("Workouts.Data").Where(&model.Equipment{UserID: userID}).First(&equipment, id).Error; err != nil {
		return nil, err
	}

	return &equipment, nil
}

func (r *equipmentRepository) GetByUserIDs(userID uint64, ids []uint64) ([]*model.Equipment, error) {
	var equipment []*model.Equipment
	if len(ids) == 0 {
		return equipment, nil
	}

	if err := r.db.Where("user_id = ?", userID).Find(&equipment, ids).Error; err != nil {
		return nil, err
	}

	return equipment, nil
}

func (r *equipmentRepository) CountByUserID(userID uint64) (int64, error) {
	var total int64
	if err := r.db.Model(&model.Equipment{}).Where(&model.Equipment{UserID: userID}).Count(&total).Error; err != nil {
		return 0, err
	}

	return total, nil
}

func (r *equipmentRepository) ListByUserID(userID uint64, limit int, offset int) ([]*model.Equipment, error) {
	var equipment []*model.Equipment

	q := r.db.Where(&model.Equipment{UserID: userID}).Order("name DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if offset > 0 {
		q = q.Offset(offset)
	}

	if err := q.Find(&equipment).Error; err != nil {
		return nil, err
	}

	return equipment, nil
}

func (r *equipmentRepository) Save(e *model.Equipment) error {
	return e.Save(r.db)
}

func (r *equipmentRepository) Delete(e *model.Equipment) error {
	return e.Delete(r.db)
}
