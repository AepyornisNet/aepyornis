package controller

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"path"
	"strings"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/model/dto"
	"github.com/AepyornisNet/aepyornis/pkg/repository"
	"github.com/AepyornisNet/aepyornis/pkg/worker"
	"github.com/labstack/echo/v5"
	"github.com/samber/do/v2"
	"github.com/spf13/cast"
	"github.com/vgarvardt/gue/v6"
	"gorm.io/gorm"
)

type RouteSegmentController interface {
	GetRouteSegments(c *echo.Context) error
	GetRouteSegment(c *echo.Context) error
	CreateRouteSegment(c *echo.Context) error
	CreateRouteSegmentFromWorkout(c *echo.Context) error
	DeleteRouteSegment(c *echo.Context) error
	RefreshRouteSegment(c *echo.Context) error
	UpdateRouteSegment(c *echo.Context) error
	DownloadRouteSegment(c *echo.Context) error
	FindRouteSegmentMatches(c *echo.Context) error
	GetRouteSegmentMatches(c *echo.Context) error
	LikeRouteSegment(c *echo.Context) error
	UnlikeRouteSegment(c *echo.Context) error
	GetRouteSegmentLikers(c *echo.Context) error
}

type routeSegmentController struct {
	client           *gue.Client
	db               *gorm.DB
	logger           *slog.Logger
	routeSegmentRepo repository.RouteSegment
	workoutRepo      repository.Workout
}

func NewRouteSegmentController(injector do.Injector) RouteSegmentController {
	return &routeSegmentController{
		client:           do.MustInvoke[*gue.Client](injector),
		db:               do.MustInvoke[*gorm.DB](injector),
		logger:           do.MustInvoke[*slog.Logger](injector),
		routeSegmentRepo: do.MustInvoke[repository.RouteSegment](injector),
		workoutRepo:      do.MustInvoke[repository.Workout](injector),
	}
}

func canEditRouteSegment(user *model.User, rs *model.RouteSegment) bool {
	if user == nil || rs == nil {
		return false
	}
	if user.Admin {
		return true
	}
	return rs.ProfileID != 0 && rs.ProfileID == user.Profile.ID
}

func (rc *routeSegmentController) getRouteSegment(c *echo.Context) (*model.RouteSegment, error) {
	id, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return nil, err
	}

	rs, err := rc.routeSegmentRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	user := currentUser(c)
	allowed, err := model.CanReadRouteSegment(rc.db, user, rs)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, gorm.ErrRecordNotFound
	}

	return rs, nil
}

// GetRouteSegments returns a paginated list of route segments
// @Summary      List route segments
// @Tags         route-segments
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Param        page      query  int false "Page"
// @Param        per_page  query  int false "Items per page"
// @Success      200  {object}  dto.PaginatedResponse[dto.RouteSegmentResponse]
// @Failure      400  {object}  dto.Response[string]
// @Failure      500  {object}  dto.Response[string]
// @Router       /route-segments [get]
func (rc *routeSegmentController) GetRouteSegments(c *echo.Context) error {
	user := currentUser(c)

	var pagination dto.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	pagination.SetDefaults()

	totalCount, err := rc.routeSegmentRepo.Count(user)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	routeSegments, err := rc.routeSegmentRepo.List(user, pagination.PerPage, pagination.GetOffset())
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	results := dto.NewRouteSegmentsResponse(routeSegments)

	var currentProfileID uint64
	var isAdmin bool
	if user != nil {
		currentProfileID = user.Profile.ID
		isAdmin = user.Admin
	}
	for i := range results {
		results[i].CanEdit = isAdmin || (results[i].ProfileID != 0 && results[i].ProfileID == currentProfileID)
		results[i].CanDelete = isAdmin || (results[i].ProfileID != 0 && results[i].ProfileID == currentProfileID)
	}

	resp := dto.PaginatedResponse[dto.RouteSegmentResponse]{
		Results:    results,
		Page:       pagination.Page,
		PerPage:    pagination.PerPage,
		TotalPages: pagination.CalculateTotalPages(totalCount),
		TotalCount: totalCount,
	}

	return c.JSON(http.StatusOK, resp)
}

// GetRouteSegment returns a single route segment by ID with full details
// @Summary      Get route segment
// @Tags         route-segments
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "Route segment ID"
// @Produce      json
// @Success      200  {object}  dto.Response[dto.RouteSegmentDetailResponse]
// @Failure      404  {object}  dto.Response[string]
// @Router       /route-segments/{id} [get]
func (rc *routeSegmentController) filterVisibleMatches(user *model.User, matches []*model.RouteSegmentMatch) []*model.RouteSegmentMatch {
	visible := make([]*model.RouteSegmentMatch, 0, len(matches))
	for _, m := range matches {
		if m != nil && m.Workout != nil {
			canRead, _ := model.CanReadWorkout(rc.db, user, m.Workout)
			if canRead {
				visible = append(visible, m)
			}
		}
	}
	return visible
}

func buildRouteSegmentStatsResponse(statsData *repository.RouteSegmentStats) *dto.RouteSegmentStatsResponse {
	if statsData == nil {
		return nil
	}
	statsResp := &dto.RouteSegmentStatsResponse{
		TotalEfforts:   statsData.TotalEfforts,
		UniqueAthletes: statsData.UniqueAthletes,
		AvgDuration:    statsData.AvgDuration,
		AvgSpeed:       statsData.AvgSpeed,
	}
	if cr := statsData.CourseRecord; cr != nil && cr.Workout != nil {
		statsResp.CourseRecord = &dto.CourseRecordInfo{
			WorkoutID:   cr.WorkoutID,
			WorkoutName: cr.Workout.Name,
			Duration:    int(cr.Duration.Seconds()),
			Speed:       cr.AverageSpeed(),
		}
		if cr.Workout.Profile != nil {
			statsResp.CourseRecord.ProfileID = cr.Workout.Profile.ID
			statsResp.CourseRecord.ProfileName = cr.Workout.Profile.DisplayName
		}
	}
	return statsResp
}

// GetRouteSegment returns a single route segment by ID with full details
// @Summary      Get route segment
// @Tags         route-segments
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "Route segment ID"
// @Produce      json
// @Success      200  {object}  dto.Response[dto.RouteSegmentDetailResponse]
// @Failure      404  {object}  dto.Response[string]
// @Router       /route-segments/{id} [get]
func (rc *routeSegmentController) GetRouteSegment(c *echo.Context) error {
	rs, err := rc.getRouteSegment(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	var currentProfileID uint64
	var isAdmin bool
	user := currentUser(c)
	if user != nil {
		currentProfileID = user.Profile.ID
		isAdmin = user.Admin
	}

	rs.RouteSegmentMatches = rc.filterVisibleMatches(user, rs.RouteSegmentMatches)

	detailResp := dto.NewRouteSegmentDetailResponse(rs)
	detailResp.CanEdit = isAdmin || (rs.ProfileID != 0 && rs.ProfileID == currentProfileID)
	detailResp.CanDelete = isAdmin || (rs.ProfileID != 0 && rs.ProfileID == currentProfileID)

	likeCount, _ := rc.routeSegmentRepo.CountLikes(rs.ID)
	detailResp.LikeCount = likeCount
	if currentProfileID > 0 {
		hasLiked, _ := rc.routeSegmentRepo.HasLiked(rs.ID, currentProfileID)
		detailResp.HasLiked = hasLiked
	}

	statsData, err := rc.routeSegmentRepo.GetStats(rs.ID, user)
	if err == nil {
		detailResp.Stats = buildRouteSegmentStatsResponse(statsData)
	}

	resp := dto.Response[dto.RouteSegmentDetailResponse]{
		Results: detailResp,
	}

	return c.JSON(http.StatusOK, resp)
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

func applyRouteSegmentCreationParams(rs *model.RouteSegment, params *dto.RouteSegmentFromWorkoutRequest, workout *model.Workout, user *model.User) {
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

// CreateRouteSegment uploads one or more route segment files
// @Summary      Create route segment
// @Tags         route-segments
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Accept       multipart/form-data
// @Produce      json
// @Param        file   formData  file   true  "GPX file"
// @Param        notes  formData  string false "Notes"
// @Success      201  {object}  dto.Response[dto.RouteSegmentsDetailResponse]
// @Failure      400  {object}  dto.Response[string]
// @Failure      500  {object}  dto.Response[string]
// @Router       /route-segments [post]
func (rc *routeSegmentController) CreateRouteSegment(c *echo.Context) error {
	form, err := c.MultipartForm()
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	files := form.File["file"]
	errMsg := []string{}

	var profileID uint64
	if user := currentUser(c); user != nil {
		profileID = user.Profile.ID
	}

	notes := multipartFormValue(form, c, "notes")
	category := multipartFormValue(form, c, "category")
	visibility := multipartFormValue(form, c, "visibility")
	difficulty := multipartFormValue(form, c, "difficulty")
	description := multipartFormValue(form, c, "description")
	bidirectional := multipartFormValue(form, c, "bidirectional")
	circular := multipartFormValue(form, c, "circular")

	names := form.Value["name"]
	if len(names) == 0 {
		names = form.Value["names"]
	}

	segments := []*dto.RouteSegmentResponse{}
	for i, file := range files {
		content, parseErr := uploadedRouteSegmentFile(file)
		if parseErr != nil {
			errMsg = append(errMsg, parseErr.Error())
			continue
		}

		var customName string
		if i < len(names) {
			customName = strings.TrimSpace(names[i])
		}
		if customName == "" && len(names) == 1 && len(files) == 1 {
			customName = strings.TrimSpace(names[0])
		}

		w, addErr := rc.routeSegmentRepo.CreateFromContent(notes, file.Filename, content)
		if addErr != nil {
			errMsg = append(errMsg, addErr.Error())
			continue
		}

		if profileID != 0 {
			w.ProfileID = profileID
		}
		if customName != "" {
			w.Name = customName
		}
		if category != "" && isValidRouteSegmentCategory(category) {
			w.Category = category
		}
		if vis := model.WorkoutVisibility(visibility); vis.IsValid() && vis != "" {
			w.Visibility = vis
		}
		if diff := model.RouteSegmentDifficulty(difficulty); diff.IsValid() {
			w.Difficulty = diff
		}
		if description != "" {
			w.Description = description
		}
		if bidirectional != "" {
			w.Bidirectional = cast.ToBool(bidirectional)
		}
		if circular != "" {
			w.Circular = cast.ToBool(circular)
		}
		_ = w.Save(rc.db)

		resp := dto.NewRouteSegmentResponse(w)
		segments = append(segments, &resp)

		if err := worker.EnqueueRouteSegmentUpdate(c.Request().Context(), rc.client, w.ID); err != nil {
			rc.logger.Error("Failed to enqueue route segment update", "route_segment_id", w.ID, "error", err)
		}
	}

	resp := dto.Response[dto.RouteSegmentsDetailResponse]{
		Results: segments,
		Errors:  errMsg,
	}

	return c.JSON(http.StatusCreated, resp)
}

// CreateRouteSegmentFromWorkout creates a route segment from a workout
// @Summary      Create route segment from workout
// @Tags         route-segments
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "Workout ID"
// @Accept       json
// @Produce      json
// @Success      201  {object}  dto.Response[dto.RouteSegmentDetailResponse]
// @Failure      400  {object}  dto.Response[string]
// @Failure      404  {object}  dto.Response[string]
// @Router       /workouts/{id}/route-segment [post]
func (rc *routeSegmentController) CreateRouteSegmentFromWorkout(c *echo.Context) error {
	workoutID, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	workout, err := rc.workoutRepo.GetDetailsByID(workoutID)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	user := currentUser(c)
	canRead, err := model.CanReadWorkout(rc.db, user, workout)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}
	if !canRead {
		return renderApiError(c, http.StatusNotFound, gorm.ErrRecordNotFound)
	}

	var params dto.RouteSegmentFromWorkoutRequest
	if err := c.Bind(&params); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	if err := c.Validate(&params); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	if !isValidRouteSegmentCategory(params.Category) {
		return renderApiError(c, http.StatusBadRequest, fmt.Errorf("invalid route segment category: %s", params.Category))
	}

	content, err := model.RouteSegmentFromPoints(workout, params.Start, params.End)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	rs, err := rc.routeSegmentRepo.CreateFromContent("", params.Filename(), content)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	applyRouteSegmentCreationParams(rs, &params, workout, user)
	if err := rs.Save(rc.db); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if err := worker.EnqueueRouteSegmentUpdate(c.Request().Context(), rc.client, rs.ID); err != nil {
		rc.logger.Error("Failed to enqueue route segment update", "route_segment_id", rs.ID, "error", err)
	}

	resp := dto.Response[dto.RouteSegmentDetailResponse]{
		Results: dto.NewRouteSegmentDetailResponse(rs),
	}

	return c.JSON(http.StatusCreated, resp)
}

// DeleteRouteSegment deletes a route segment
// @Summary      Delete route segment
// @Tags         route-segments
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "Route segment ID"
// @Produce      json
// @Success      200  {object}  dto.Response[map[string]string]
// @Failure      404  {object}  dto.Response[string]
// @Failure      500  {object}  dto.Response[string]
// @Router       /route-segments/{id} [delete]
func (rc *routeSegmentController) DeleteRouteSegment(c *echo.Context) error {
	rs, err := rc.getRouteSegment(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	user := currentUser(c)
	if !canEditRouteSegment(user, rs) {
		return renderApiError(c, http.StatusForbidden, errors.New("forbidden"))
	}

	if err := rc.routeSegmentRepo.Delete(rs); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[map[string]string]{
		Results: map[string]string{"message": "Route segment deleted successfully"},
	}

	return c.JSON(http.StatusOK, resp)
}

// RefreshRouteSegment marks a route segment for refresh
// @Summary      Refresh route segment
// @Tags         route-segments
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "Route segment ID"
// @Produce      json
// @Success      200  {object}  dto.Response[map[string]string]
// @Failure      404  {object}  dto.Response[string]
// @Failure      500  {object}  dto.Response[string]
// @Router       /route-segments/{id}/refresh [post]
func (rc *routeSegmentController) RefreshRouteSegment(c *echo.Context) error {
	rs, err := rc.getRouteSegment(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	user := currentUser(c)
	if !canEditRouteSegment(user, rs) {
		return renderApiError(c, http.StatusForbidden, errors.New("forbidden"))
	}

	if err := rs.UpdateFromContent(); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if err := rc.routeSegmentRepo.Save(rs); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[map[string]string]{
		Results: map[string]string{"message": "Route segment refreshed successfully"},
	}

	return c.JSON(http.StatusOK, resp)
}

// UpdateRouteSegment updates a route segment
// @Summary      Update route segment
// @Tags         route-segments
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "Route segment ID"
// @Accept       json
// @Produce      json
// @Success      200  {object}  dto.Response[dto.RouteSegmentDetailResponse]
// @Failure      400  {object}  dto.Response[string]
// @Failure      404  {object}  dto.Response[string]
// @Failure      500  {object}  dto.Response[string]
// @Router       /route-segments/{id} [put]
func (rc *routeSegmentController) UpdateRouteSegment(c *echo.Context) error {
	rs, err := rc.getRouteSegment(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	user := currentUser(c)
	if !canEditRouteSegment(user, rs) {
		return renderApiError(c, http.StatusForbidden, errors.New("forbidden"))
	}

	var params dto.RouteSegmentUpdateRequest
	if err := c.Bind(&params); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	if err := c.Validate(&params); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	if !isValidRouteSegmentCategory(params.Category) {
		return renderApiError(c, http.StatusBadRequest, fmt.Errorf("invalid route segment category: %s", params.Category))
	}

	rs.Name = params.Name
	rs.Notes = params.Notes
	rs.Category = params.Category
	if params.Visibility != "" {
		rs.Visibility = params.Visibility
	}
	rs.Description = params.Description
	rs.Difficulty = params.Difficulty
	rs.Bidirectional = params.Bidirectional
	rs.Circular = params.Circular
	rs.Dirty = true

	if err := rs.Save(rc.db); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if err := worker.EnqueueRouteSegmentUpdate(c.Request().Context(), rc.client, rs.ID); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[dto.RouteSegmentDetailResponse]{
		Results: dto.NewRouteSegmentDetailResponse(rs),
	}

	return c.JSON(http.StatusOK, resp)
}

// DownloadRouteSegment downloads the original route segment file
// @Summary      Download route segment file
// @Tags         route-segments
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "Route segment ID"
// @Produce      octet-stream
// @Success      200  {string}  string  "binary GPX content"
// @Failure      404  {object}  dto.Response[string]
// @Router       /route-segments/{id}/download [get]
func (rc *routeSegmentController) DownloadRouteSegment(c *echo.Context) error {
	rs, err := rc.getRouteSegment(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	basename := path.Base(rs.Filename)
	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename=\""+basename+"\"")

	return c.Stream(http.StatusOK, "application/binary", bytes.NewReader(rs.Content))
}

// FindRouteSegmentMatches finds matching workouts for a route segment
// @Summary      Find matching workouts
// @Tags         route-segments
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "Route segment ID"
// @Produce      json
// @Success      200  {object}  dto.Response[map[string]string]
// @Failure      404  {object}  dto.Response[string]
// @Failure      500  {object}  dto.Response[string]
// @Router       /route-segments/{id}/matches [post]
func (rc *routeSegmentController) FindRouteSegmentMatches(c *echo.Context) error {
	rs, err := rc.getRouteSegment(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	user := currentUser(c)
	if !canEditRouteSegment(user, rs) {
		return renderApiError(c, http.StatusForbidden, errors.New("forbidden"))
	}

	rs.Dirty = true
	if err := rs.Save(rc.db); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if err := worker.EnqueueRouteSegmentUpdate(c.Request().Context(), rc.client, rs.ID); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[map[string]string]{
		Results: map[string]string{"message": "Finding matches in background"},
	}

	return c.JSON(http.StatusOK, resp)
}

func (rc *routeSegmentController) GetRouteSegmentMatches(c *echo.Context) error {
	rs, err := rc.getRouteSegment(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	user := currentUser(c)
	var query dto.RouteSegmentMatchesQuery
	if err := c.Bind(&query); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	query.SetDefaults()

	matches, totalCount, err := rc.routeSegmentRepo.GetMatches(rs.ID, user, query.Sort, query.PerPage, query.GetOffset())
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	results := make([]dto.RouteSegmentMatch, len(matches))
	for i, m := range matches {
		results[i] = dto.NewRouteSegmentMatchResponse(m)
	}

	resp := dto.PaginatedResponse[dto.RouteSegmentMatch]{
		Results:    results,
		Page:       query.Page,
		PerPage:    query.PerPage,
		TotalPages: query.CalculateTotalPages(totalCount),
		TotalCount: totalCount,
	}

	return c.JSON(http.StatusOK, resp)
}

func (rc *routeSegmentController) LikeRouteSegment(c *echo.Context) error {
	rs, err := rc.getRouteSegment(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	user := currentUser(c)
	if user == nil {
		return renderApiError(c, http.StatusUnauthorized, errors.New("unauthorized"))
	}

	if err := rc.routeSegmentRepo.Like(rs.ID, user.Profile.ID); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	count, _ := rc.routeSegmentRepo.CountLikes(rs.ID)
	return c.JSON(http.StatusOK, dto.Response[map[string]any]{
		Results: map[string]any{"liked": true, "like_count": count},
	})
}

func (rc *routeSegmentController) UnlikeRouteSegment(c *echo.Context) error {
	rs, err := rc.getRouteSegment(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	user := currentUser(c)
	if user == nil {
		return renderApiError(c, http.StatusUnauthorized, errors.New("unauthorized"))
	}

	if err := rc.routeSegmentRepo.Unlike(rs.ID, user.Profile.ID); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	count, _ := rc.routeSegmentRepo.CountLikes(rs.ID)
	return c.JSON(http.StatusOK, dto.Response[map[string]any]{
		Results: map[string]any{"liked": false, "like_count": count},
	})
}

// GetRouteSegmentLikers returns users who liked the route segment
// @Summary      Get route segment likers
// @Tags         route-segments
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path      int  true  "Route Segment ID"
// @Produce      json
// @Success      200  {object}  dto.Response[[]dto.WorkoutLikeResponse]
// @Failure      400  {object}  dto.Response[string]
// @Failure      404  {object}  dto.Response[string]
// @Router       /route-segments/{id}/likes [get]
func (rc *routeSegmentController) GetRouteSegmentLikers(c *echo.Context) error {
	rs, err := rc.getRouteSegment(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	likes, err := rc.routeSegmentRepo.GetLikes(rs.ID)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	results := make([]dto.WorkoutLikeResponse, 0, len(likes))
	for i := range likes {
		likeResponse := dto.NewWorkoutLikeResponse(&likes[i])
		results = append(results, likeResponse)
	}

	return c.JSON(http.StatusOK, dto.Response[[]dto.WorkoutLikeResponse]{
		Results: results,
	})
}

func uploadedRouteSegmentFile(file *multipart.FileHeader) ([]byte, error) {
	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	content, err := io.ReadAll(src)
	if err != nil {
		return nil, err
	}

	return content, nil
}

func isValidRouteSegmentCategory(cat string) bool {
	if cat == "" {
		return true
	}
	wt, valid := model.ParseWorkoutType(cat)
	return valid && wt != model.WorkoutTypeAll && wt != model.WorkoutTypeUnknown
}
