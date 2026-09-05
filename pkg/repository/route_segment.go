package repository

import (
	"fmt"
	"sort"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/samber/do/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RouteSegmentStats struct {
	TotalEfforts   int64
	UniqueAthletes int64
	CourseRecord   *model.RouteSegmentMatch
	AvgDuration    float64
	AvgSpeed       float64
}

type RouteSegment interface {
	GetByID(id uint64) (*model.RouteSegment, error)
	Count(viewer *model.User) (int64, error)
	List(viewer *model.User, limit int, offset int) ([]*model.RouteSegment, error)
	CreateFromContent(notes string, filename string, content []byte) (*model.RouteSegment, error)
	Save(routeSegment *model.RouteSegment) error
	Delete(routeSegment *model.RouteSegment) error
	Like(routeSegmentID uint64, profileID uint64) error
	Unlike(routeSegmentID uint64, profileID uint64) error
	HasLiked(routeSegmentID uint64, profileID uint64) (bool, error)
	CountLikes(routeSegmentID uint64) (int64, error)
	GetLikers(routeSegmentID uint64) ([]*model.Profile, error)
	GetLikes(routeSegmentID uint64) ([]model.APStatusLike, error)
	GetMatches(routeSegmentID uint64, viewer *model.User, sort string, limit int, offset int) ([]*model.RouteSegmentMatch, int64, error)
	GetStats(routeSegmentID uint64, viewer *model.User) (*RouteSegmentStats, error)
}

type routeSegmentRepository struct {
	db *gorm.DB
}

func NewRouteSegment(injector do.Injector) (RouteSegment, error) {
	return &routeSegmentRepository{db: do.MustInvoke[*gorm.DB](injector)}, nil
}

func (r *routeSegmentRepository) GetByID(id uint64) (*model.RouteSegment, error) {
	var routeSegment model.RouteSegment
	if err := r.db.Preload("Profile").Preload("RouteSegmentMatches.Workout.Profile").First(&routeSegment, id).Error; err != nil {
		return nil, err
	}

	sort.Slice(routeSegment.RouteSegmentMatches, func(i, j int) bool {
		if routeSegment.RouteSegmentMatches[i].Workout == nil || routeSegment.RouteSegmentMatches[j].Workout == nil {
			return false
		}
		return routeSegment.RouteSegmentMatches[i].Workout.GetDate().Before(routeSegment.RouteSegmentMatches[j].Workout.GetDate())
	})

	return &routeSegment, nil
}

func (r *routeSegmentRepository) Count(viewer *model.User) (int64, error) {
	var total int64
	q := r.db.Model(&model.RouteSegment{})
	q = model.ScopeVisibleRouteSegments(q, viewer)
	if err := q.Count(&total).Error; err != nil {
		return 0, err
	}

	return total, nil
}

func (r *routeSegmentRepository) List(viewer *model.User, limit int, offset int) ([]*model.RouteSegment, error) {
	var routeSegments []*model.RouteSegment
	q := r.db.Model(&model.RouteSegment{})
	q = model.ScopeVisibleRouteSegments(q, viewer)
	q = q.Preload("Profile").
		Preload("RouteSegmentMatches", func(db *gorm.DB) *gorm.DB {
			return model.ScopeVisibleJoinedWorkouts(db.Select("route_segment_matches.*").Joins("JOIN workouts ON workouts.id = route_segment_matches.workout_id"), viewer)
		}).
		Order("route_segments.created_at DESC")
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

func (r *routeSegmentRepository) routeSegmentStatusID(routeSegmentID uint64) (uint64, error) {
	var rs model.RouteSegment
	if err := r.db.First(&rs, routeSegmentID).Error; err != nil {
		return 0, err
	}

	status := &model.APStatus{
		RouteSegmentID: &routeSegmentID,
		StatusType:     model.APStatusTypeRouteSegment,
		Origin:         "local",
		ActivityID:     fmt.Sprintf("local:route_segment:%d:activity", routeSegmentID),
		ObjectID:       fmt.Sprintf("local:route_segment:%d:object", routeSegmentID),
		Activity:       []byte("{}"),
		Content:        "",
	}

	if err := r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(status).Error; err != nil {
		return 0, err
	}

	if status.ID == 0 {
		if err := r.db.Where("route_segment_id = ? AND status_type = ?", routeSegmentID, model.APStatusTypeRouteSegment).Take(status).Error; err != nil {
			return 0, err
		}
	}

	return status.ID, nil
}

func (r *routeSegmentRepository) Like(routeSegmentID uint64, profileID uint64) error {
	statusID, err := r.routeSegmentStatusID(routeSegmentID)
	if err != nil {
		return err
	}
	like := &model.APStatusLike{StatusID: statusID, ProfileID: &profileID}
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(like).Error
}

func (r *routeSegmentRepository) Unlike(routeSegmentID uint64, profileID uint64) error {
	statusID, err := r.routeSegmentStatusID(routeSegmentID)
	if err != nil {
		return err
	}
	return r.db.Where("status_id = ? AND profile_id = ?", statusID, profileID).Delete(&model.APStatusLike{}).Error
}

func (r *routeSegmentRepository) HasLiked(routeSegmentID uint64, profileID uint64) (bool, error) {
	if profileID == 0 {
		return false, nil
	}
	var count int64
	err := r.db.Table("ap_status_likes").
		Joins("JOIN ap_statuses ON ap_statuses.id = ap_status_likes.status_id").
		Where("ap_statuses.route_segment_id = ? AND ap_status_likes.profile_id = ?", routeSegmentID, profileID).
		Count(&count).Error
	return count > 0, err
}

func (r *routeSegmentRepository) CountLikes(routeSegmentID uint64) (int64, error) {
	var count int64
	err := r.db.Table("ap_status_likes").
		Joins("JOIN ap_statuses ON ap_statuses.id = ap_status_likes.status_id").
		Where("ap_statuses.route_segment_id = ?", routeSegmentID).
		Count(&count).Error
	return count, err
}

func (r *routeSegmentRepository) GetLikers(routeSegmentID uint64) ([]*model.Profile, error) {
	var likes []model.APStatusLike
	err := r.db.Preload("Profile").
		Joins("JOIN ap_statuses ON ap_statuses.id = ap_status_likes.status_id").
		Where("ap_statuses.route_segment_id = ?", routeSegmentID).
		Order("ap_status_likes.created_at DESC").
		Find(&likes).Error
	if err != nil {
		return nil, err
	}

	profiles := make([]*model.Profile, 0, len(likes))
	for _, l := range likes {
		if l.Profile != nil {
			profiles = append(profiles, l.Profile)
		}
	}
	return profiles, nil
}

func (r *routeSegmentRepository) GetLikes(routeSegmentID uint64) ([]model.APStatusLike, error) {
	var likes []model.APStatusLike
	err := r.db.Preload("Profile").Preload("Profile.User").
		Joins("JOIN ap_statuses ON ap_statuses.id = ap_status_likes.status_id").
		Where("ap_statuses.route_segment_id = ?", routeSegmentID).
		Order("ap_status_likes.created_at DESC, ap_status_likes.id DESC").
		Find(&likes).Error
	if err != nil {
		return nil, err
	}
	return likes, nil
}

func (r *routeSegmentRepository) GetMatches(routeSegmentID uint64, viewer *model.User, sort string, limit int, offset int) ([]*model.RouteSegmentMatch, int64, error) {
	var matches []*model.RouteSegmentMatch
	var total int64

	base := r.db.Model(&model.RouteSegmentMatch{}).
		Joins("JOIN workouts ON workouts.id = route_segment_matches.workout_id").
		Where("route_segment_matches.route_segment_id = ?", routeSegmentID)
	base = model.ScopeVisibleJoinedWorkouts(base, viewer)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	q := r.db.Preload("Workout.Profile").
		Joins("JOIN workouts ON workouts.id = route_segment_matches.workout_id").
		Where("route_segment_matches.route_segment_id = ?", routeSegmentID)
	q = model.ScopeVisibleJoinedWorkouts(q, viewer)

	switch sort {
	case "newest", "recent":
		q = q.Order("workouts.date DESC, route_segment_matches.duration ASC")
	case "oldest":
		q = q.Order("workouts.date ASC, route_segment_matches.duration ASC")
	case "best", "fastest":
		fallthrough
	default:
		q = q.Order("route_segment_matches.duration ASC, workouts.date DESC")
	}

	if limit > 0 {
		q = q.Limit(limit)
	}
	if offset > 0 {
		q = q.Offset(offset)
	}

	if err := q.Find(&matches).Error; err != nil {
		return nil, 0, err
	}

	return matches, total, nil
}

func (r *routeSegmentRepository) GetStats(routeSegmentID uint64, viewer *model.User) (*RouteSegmentStats, error) {
	var totalEfforts int64
	base := r.db.Model(&model.RouteSegmentMatch{}).
		Joins("JOIN workouts ON workouts.id = route_segment_matches.workout_id").
		Where("route_segment_matches.route_segment_id = ?", routeSegmentID)
	base = model.ScopeVisibleJoinedWorkouts(base, viewer)
	if err := base.Count(&totalEfforts).Error; err != nil {
		return nil, err
	}

	var uniqueAthletes int64
	qAthletes := r.db.Model(&model.RouteSegmentMatch{}).
		Joins("JOIN workouts ON workouts.id = route_segment_matches.workout_id").
		Where("route_segment_matches.route_segment_id = ?", routeSegmentID)
	qAthletes = model.ScopeVisibleJoinedWorkouts(qAthletes, viewer)
	if err := qAthletes.Select("COUNT(DISTINCT workouts.profile_id)").Scan(&uniqueAthletes).Error; err != nil {
		return nil, err
	}

	var bestMatch model.RouteSegmentMatch
	qBest := r.db.Preload("Workout.Profile").
		Joins("JOIN workouts ON workouts.id = route_segment_matches.workout_id").
		Where("route_segment_matches.route_segment_id = ?", routeSegmentID)
	qBest = model.ScopeVisibleJoinedWorkouts(qBest, viewer)
	err := qBest.Order("route_segment_matches.duration ASC").First(&bestMatch).Error
	var courseRecord *model.RouteSegmentMatch
	if err == nil {
		courseRecord = &bestMatch
	}

	var avgDuration float64
	var avgSpeed float64
	if totalEfforts > 0 {
		var result struct {
			AvgDuration float64
			AvgDistance float64
		}
		qAvg := r.db.Model(&model.RouteSegmentMatch{}).
			Joins("JOIN workouts ON workouts.id = route_segment_matches.workout_id").
			Where("route_segment_matches.route_segment_id = ?", routeSegmentID)
		qAvg = model.ScopeVisibleJoinedWorkouts(qAvg, viewer)
		_ = qAvg.Select("AVG(route_segment_matches.duration) as avg_duration, AVG(route_segment_matches.distance) as avg_distance").
			Scan(&result).Error

		if result.AvgDuration > 0 {
			durationSecs := result.AvgDuration / float64(time.Second)
			avgDuration = durationSecs
			if durationSecs > 0 {
				avgSpeed = result.AvgDistance / durationSecs
			}
		}
	}

	return &RouteSegmentStats{
		TotalEfforts:   totalEfforts,
		UniqueAthletes: uniqueAthletes,
		CourseRecord:   courseRecord,
		AvgDuration:    avgDuration,
		AvgSpeed:       avgSpeed,
	}, nil
}
