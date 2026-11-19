package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/model/dto"
	"github.com/AepyornisNet/aepyornis/pkg/repository"
	"github.com/samber/do/v2"
	"github.com/spf13/cast"
	"github.com/vgarvardt/gue/v6"
	"gorm.io/gorm"
)

var ErrForbidden = errors.New("forbidden")

type RouteSegmentUploadParams struct {
	ProfileID     uint64
	Notes         string
	Category      string
	Visibility    string
	Difficulty    string
	Description   string
	Bidirectional string
	Circular      string
	Names         []string
}

type RouteSegmentService interface {
	GetRouteSegments(ctx context.Context, viewer *model.User, page, perPage int) ([]*model.RouteSegment, int64, error)
	GetRouteSegment(ctx context.Context, viewer *model.User, id uint64) (*model.RouteSegment, *repository.RouteSegmentStats, int64, bool, error)
	CreateRouteSegments(ctx context.Context, viewer *model.User, files []UploadedWorkoutFile, params RouteSegmentUploadParams) ([]*model.RouteSegment, []string)
	CreateRouteSegmentFromWorkout(ctx context.Context, viewer *model.User, workoutID uint64, params *dto.RouteSegmentFromWorkoutRequest) (*model.RouteSegment, error)
	DeleteRouteSegment(ctx context.Context, viewer *model.User, rs *model.RouteSegment) error
	RefreshRouteSegment(ctx context.Context, viewer *model.User, rs *model.RouteSegment) error
	UpdateRouteSegment(ctx context.Context, viewer *model.User, rs *model.RouteSegment, req *dto.RouteSegmentUpdateRequest) (*model.RouteSegment, error)
	FindRouteSegmentMatches(ctx context.Context, viewer *model.User, rs *model.RouteSegment) error
	LikeRouteSegment(ctx context.Context, viewer *model.User, rs *model.RouteSegment) (int64, error)
	UnlikeRouteSegment(ctx context.Context, viewer *model.User, rs *model.RouteSegment) (int64, error)
	GetRouteSegmentLikers(ctx context.Context, rs *model.RouteSegment) ([]*model.Profile, error)
	GetRouteSegmentMatches(ctx context.Context, viewer *model.User, segmentID uint64, sort string, perPage, offset int) ([]*model.RouteSegmentMatch, int64, error)
	CanReadRouteSegment(viewer *model.User, rs *model.RouteSegment) (bool, error)
	CanEditRouteSegment(viewer *model.User, rs *model.RouteSegment) bool
}

type routeSegmentService struct {
	client           *gue.Client
	db               *gorm.DB
	logger           *slog.Logger
	routeSegmentRepo repository.RouteSegment
	workoutRepo      repository.Workout
}

func NewRouteSegmentService(injector do.Injector) (RouteSegmentService, error) {
	return &routeSegmentService{
		client:           do.MustInvoke[*gue.Client](injector),
		db:               do.MustInvoke[*gorm.DB](injector),
		logger:           do.MustInvoke[*slog.Logger](injector),
		routeSegmentRepo: do.MustInvoke[repository.RouteSegment](injector),
		workoutRepo:      do.MustInvoke[repository.Workout](injector),
	}, nil
}

func (s *routeSegmentService) CanEditRouteSegment(user *model.User, rs *model.RouteSegment) bool {
	if user == nil || rs == nil {
		return false
	}
	if user.Admin {
		return true
	}
	return rs.ProfileID != 0 && rs.ProfileID == user.Profile.ID
}

func (s *routeSegmentService) CanReadRouteSegment(viewer *model.User, rs *model.RouteSegment) (bool, error) {
	if rs == nil {
		return false, gorm.ErrRecordNotFound
	}
	return model.CanReadRouteSegment(s.db, viewer, rs)
}

func (s *routeSegmentService) GetRouteSegments(_ context.Context, viewer *model.User, page, perPage int) ([]*model.RouteSegment, int64, error) {
	totalCount, err := s.routeSegmentRepo.Count(viewer)
	if err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * perPage
	segments, err := s.routeSegmentRepo.List(viewer, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	return segments, totalCount, nil
}

func (s *routeSegmentService) GetRouteSegment(_ context.Context, viewer *model.User, id uint64) (*model.RouteSegment, *repository.RouteSegmentStats, int64, bool, error) {
	rs, err := s.routeSegmentRepo.GetByID(id)
	if err != nil {
		return nil, nil, 0, false, err
	}

	canRead, err := s.CanReadRouteSegment(viewer, rs)
	if err != nil {
		return nil, nil, 0, false, err
	}
	if !canRead {
		return nil, nil, 0, false, gorm.ErrRecordNotFound
	}

	rs.RouteSegmentMatches = filterVisibleMatches(s.db, viewer, rs.RouteSegmentMatches)

	likeCount, _ := s.routeSegmentRepo.CountLikes(rs.ID)
	hasLiked := false
	if viewer != nil && viewer.Profile.ID > 0 {
		hasLiked, _ = s.routeSegmentRepo.HasLiked(rs.ID, viewer.Profile.ID)
	}

	statsData, _ := s.routeSegmentRepo.GetStats(rs.ID, viewer)
	return rs, statsData, likeCount, hasLiked, nil
}

func (s *routeSegmentService) CreateRouteSegments(
	ctx context.Context,
	viewer *model.User,
	files []UploadedWorkoutFile,
	params RouteSegmentUploadParams,
) ([]*model.RouteSegment, []string) {
	var errMsgs []string
	created := make([]*model.RouteSegment, 0, len(files))

	for i, file := range files {
		w, addErr := s.routeSegmentRepo.CreateFromContent(params.Notes, file.Filename, file.Content)
		if addErr != nil {
			errMsgs = append(errMsgs, addErr.Error())
			continue
		}

		customName := resolveCustomName(params.Names, i, len(files))
		applyUploadParams(w, params, customName)
		_ = w.Save(s.db)

		created = append(created, w)

		if err := s.enqueueRouteSegmentJob(ctx, "update_route_segment", w.ID); err != nil {
			s.logger.Error("Failed to enqueue route segment update", "route_segment_id", w.ID, "error", err)
		}
	}

	return created, errMsgs
}

func (s *routeSegmentService) CreateRouteSegmentFromWorkout(
	ctx context.Context,
	viewer *model.User,
	workoutID uint64,
	params *dto.RouteSegmentFromWorkoutRequest,
) (*model.RouteSegment, error) {
	workout, err := s.workoutRepo.GetDetailsByID(workoutID)
	if err != nil {
		return nil, err
	}

	canRead, err := model.CanReadWorkout(s.db, viewer, workout)
	if err != nil {
		return nil, err
	}
	if !canRead {
		return nil, gorm.ErrRecordNotFound
	}

	if !isValidRouteSegmentCategory(params.Category) {
		return nil, fmt.Errorf("invalid route segment category: %s", params.Category)
	}

	content, err := model.RouteSegmentFromPoints(workout, params.Start, params.End)
	if err != nil {
		return nil, err
	}

	rs, err := s.routeSegmentRepo.CreateFromContent("", params.Filename(), content)
	if err != nil {
		return nil, err
	}

	applyCreationParams(rs, params, workout, viewer)
	if err := rs.Save(s.db); err != nil {
		return nil, err
	}

	if err := s.enqueueRouteSegmentJob(ctx, "update_route_segment", rs.ID); err != nil {
		s.logger.Error("Failed to enqueue route segment update", "route_segment_id", rs.ID, "error", err)
	}

	return rs, nil
}

func (s *routeSegmentService) DeleteRouteSegment(_ context.Context, viewer *model.User, rs *model.RouteSegment) error {
	if !s.CanEditRouteSegment(viewer, rs) {
		return ErrForbidden
	}
	return s.routeSegmentRepo.Delete(rs)
}

func (s *routeSegmentService) RefreshRouteSegment(ctx context.Context, viewer *model.User, rs *model.RouteSegment) error {
	if !s.CanEditRouteSegment(viewer, rs) {
		return ErrForbidden
	}
	if err := rs.UpdateFromContent(); err != nil {
		return err
	}
	return s.routeSegmentRepo.Save(rs)
}

func (s *routeSegmentService) UpdateRouteSegment(
	ctx context.Context,
	viewer *model.User,
	rs *model.RouteSegment,
	req *dto.RouteSegmentUpdateRequest,
) (*model.RouteSegment, error) {
	if !s.CanEditRouteSegment(viewer, rs) {
		return nil, ErrForbidden
	}

	if !isValidRouteSegmentCategory(req.Category) {
		return nil, fmt.Errorf("invalid route segment category: %s", req.Category)
	}

	rs.Name = req.Name
	rs.Notes = req.Notes
	rs.Category = req.Category
	if req.Visibility != "" {
		rs.Visibility = req.Visibility
	}
	rs.Description = req.Description
	rs.Difficulty = req.Difficulty
	rs.Bidirectional = req.Bidirectional
	rs.Circular = req.Circular
	rs.Dirty = true

	if err := rs.Save(s.db); err != nil {
		return nil, err
	}

	if err := s.enqueueRouteSegmentJob(ctx, "update_route_segment", rs.ID); err != nil {
		return nil, err
	}

	return rs, nil
}

func (s *routeSegmentService) FindRouteSegmentMatches(ctx context.Context, viewer *model.User, rs *model.RouteSegment) error {
	if !s.CanEditRouteSegment(viewer, rs) {
		return ErrForbidden
	}
	rs.Dirty = true
	if err := rs.Save(s.db); err != nil {
		return err
	}
	return s.enqueueRouteSegmentJob(ctx, "update_route_segment", rs.ID)
}

func (s *routeSegmentService) LikeRouteSegment(_ context.Context, viewer *model.User, rs *model.RouteSegment) (int64, error) {
	if err := s.routeSegmentRepo.Like(rs.ID, viewer.Profile.ID); err != nil {
		return 0, err
	}
	count, _ := s.routeSegmentRepo.CountLikes(rs.ID)
	return count, nil
}

func (s *routeSegmentService) UnlikeRouteSegment(_ context.Context, viewer *model.User, rs *model.RouteSegment) (int64, error) {
	if err := s.routeSegmentRepo.Unlike(rs.ID, viewer.Profile.ID); err != nil {
		return 0, err
	}
	count, _ := s.routeSegmentRepo.CountLikes(rs.ID)
	return count, nil
}

func (s *routeSegmentService) GetRouteSegmentLikers(_ context.Context, rs *model.RouteSegment) ([]*model.Profile, error) {
	return s.routeSegmentRepo.GetLikers(rs.ID)
}

func (s *routeSegmentService) GetRouteSegmentMatches(_ context.Context, viewer *model.User, segmentID uint64, sort string, perPage, offset int) ([]*model.RouteSegmentMatch, int64, error) {
	return s.routeSegmentRepo.GetMatches(segmentID, viewer, sort, perPage, offset)
}

func (s *routeSegmentService) enqueueRouteSegmentJob(ctx context.Context, jobType string, id uint64) error {
	if s.client == nil {
		return nil
	}
	raw, err := json.Marshal(struct {
		ID uint64 `json:"id"`
	}{ID: id})
	if err != nil {
		return err
	}
	return s.client.Enqueue(ctx, &gue.Job{
		Type: jobType,
		Args: raw,
	})
}

func filterVisibleMatches(db *gorm.DB, user *model.User, matches []*model.RouteSegmentMatch) []*model.RouteSegmentMatch {
	if len(matches) == 0 {
		return matches
	}
	visible := make([]*model.RouteSegmentMatch, 0, len(matches))
	for _, m := range matches {
		if m != nil && m.Workout != nil {
			canRead, _ := model.CanReadWorkout(db, user, m.Workout)
			if canRead {
				visible = append(visible, m)
			}
		}
	}
	return visible
}

func resolveCustomName(names []string, fileIndex int, totalFiles int) string {
	if fileIndex < len(names) {
		if name := strings.TrimSpace(names[fileIndex]); name != "" {
			return name
		}
	}
	if len(names) == 1 && totalFiles == 1 {
		return strings.TrimSpace(names[0])
	}
	return ""
}

func applyUploadParams(w *model.RouteSegment, p RouteSegmentUploadParams, customName string) {
	if p.ProfileID != 0 {
		w.ProfileID = p.ProfileID
	}
	if customName != "" {
		w.Name = customName
	}
	if p.Category != "" && isValidRouteSegmentCategory(p.Category) {
		w.Category = p.Category
	}
	if vis := model.WorkoutVisibility(p.Visibility); vis.IsValid() && vis != "" {
		w.Visibility = vis
	}
	if diff := model.RouteSegmentDifficulty(p.Difficulty); diff.IsValid() {
		w.Difficulty = diff
	}
	if p.Description != "" {
		w.Description = p.Description
	}
	if p.Bidirectional != "" {
		w.Bidirectional = cast.ToBool(p.Bidirectional)
	}
	if p.Circular != "" {
		w.Circular = cast.ToBool(p.Circular)
	}
}

func applyCreationParams(rs *model.RouteSegment, params *dto.RouteSegmentFromWorkoutRequest, workout *model.Workout, user *model.User) {
	rs.Category = resolveRouteSegmentCategory(params.Category, workout.Type.String())
	rs.Bidirectional = params.Bidirectional
	rs.Circular = params.Circular
	if params.Difficulty.IsValid() {
		rs.Difficulty = params.Difficulty
	}
	if params.Visibility.IsValid() && params.Visibility != "" {
		rs.Visibility = params.Visibility
	}
	if params.Description != "" {
		rs.Description = params.Description
	}
	if params.Notes != "" {
		rs.Notes = params.Notes
	}
	if user != nil {
		rs.ProfileID = user.Profile.ID
	} else if workout.ProfileID != 0 {
		rs.ProfileID = workout.ProfileID
	}
}

func resolveRouteSegmentCategory(category string, workoutType string) string {
	if category != "" && isValidRouteSegmentCategory(category) {
		return category
	}
	if workoutType != "" && isValidRouteSegmentCategory(workoutType) {
		return workoutType
	}
	return ""
}

func isValidRouteSegmentCategory(cat string) bool {
	switch cat {
	case "running", "trail_running", "walking", "hiking",
		"cycling", "road_biking", "gravel_biking", "mountain_biking":
		return true
	}
	return false
}
