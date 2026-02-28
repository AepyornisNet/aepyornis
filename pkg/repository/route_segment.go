package repository

import (
	"fmt"
	"sort"

	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"gorm.io/gorm"
)

type RouteSegment interface {
	GetByID(id uint64) (*model.RouteSegment, error)
	Count() (int64, error)
	List(limit int, offset int) ([]*model.RouteSegment, error)
	CreateFromContent(notes string, filename string, content []byte) (*model.RouteSegment, error)
	Save(routeSegment *model.RouteSegment) error
	Delete(routeSegment *model.RouteSegment) error
}

type routeSegmentRepository struct {
	db *gorm.DB
}

func NewRouteSegment(db *gorm.DB) RouteSegment {
	return &routeSegmentRepository{db: db}
}

func (r *routeSegmentRepository) GetByID(id uint64) (*model.RouteSegment, error) {
	var routeSegment model.RouteSegment
	if err := r.db.Preload("RouteSegmentMatches.Workout.User").First(&routeSegment, id).Error; err != nil {
		return nil, err
	}

	sort.Slice(routeSegment.RouteSegmentMatches, func(i, j int) bool {
		return routeSegment.RouteSegmentMatches[i].Workout.GetDate().Before(routeSegment.RouteSegmentMatches[j].Workout.GetDate())
	})

	return &routeSegment, nil
}

func (r *routeSegmentRepository) Count() (int64, error) {
	var total int64
	if err := r.db.Model(&model.RouteSegment{}).Count(&total).Error; err != nil {
		return 0, err
	}

	return total, nil
}

func (r *routeSegmentRepository) List(limit int, offset int) ([]*model.RouteSegment, error) {
	var routeSegments []*model.RouteSegment
	q := r.db.Preload("RouteSegmentMatches").Order("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if offset > 0 {
		q = q.Offset(offset)
	}

	if err := q.Find(&routeSegments).Error; err != nil {
		return nil, err
	}

	return routeSegments, nil
}

func (r *routeSegmentRepository) CreateFromContent(notes string, filename string, content []byte) (*model.RouteSegment, error) {
	routeSegment, err := model.NewRouteSegment(notes, filename, content)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", model.ErrInvalidData, err)
	}

	if err := routeSegment.Create(r.db); err != nil {
		return nil, err
	}

	return routeSegment, nil
}

func (r *routeSegmentRepository) Save(routeSegment *model.RouteSegment) error {
	return routeSegment.Save(r.db)
}

func (r *routeSegmentRepository) Delete(routeSegment *model.RouteSegment) error {
	return routeSegment.Delete(r.db)
}
