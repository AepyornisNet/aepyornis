package controller

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"path"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/model/dto"
	"github.com/AepyornisNet/aepyornis/pkg/repository"
	"github.com/AepyornisNet/aepyornis/pkg/service"
	"github.com/labstack/echo/v5"
	"github.com/samber/do/v2"
	"github.com/spf13/cast"
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
	db               *gorm.DB
	routeSegmentRepo repository.RouteSegment
	routeSegmentSvc  service.RouteSegmentService
}

func NewRouteSegmentController(injector do.Injector) RouteSegmentController {
	return &routeSegmentController{
		db:               do.MustInvoke[*gorm.DB](injector),
		routeSegmentRepo: do.MustInvoke[repository.RouteSegment](injector),
		routeSegmentSvc:  do.MustInvoke[service.RouteSegmentService](injector),
	}
}

func (rc *routeSegmentController) getRouteSegment(c *echo.Context) (*model.RouteSegment, error) {
	id, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return nil, err
	}

	user := currentUser(c)
	rs, _, _, _, err := rc.routeSegmentSvc.GetRouteSegment(c.Request().Context(), user, id)
	return rs, err
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
// @Router       /route-segments [get]
func (rc *routeSegmentController) GetRouteSegments(c *echo.Context) error {
	var pagination dto.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	pagination.SetDefaults()

	user := currentUser(c)
	segments, totalCount, err := rc.routeSegmentSvc.GetRouteSegments(
		c.Request().Context(),
		user,
		pagination.Page,
		pagination.PerPage,
	)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	results := make([]dto.RouteSegmentResponse, 0, len(segments))
	var currentProfileID uint64
	var isAdmin bool
	if user != nil {
		currentProfileID = user.Profile.ID
		isAdmin = user.Admin
	}
	for _, seg := range segments {
		res := dto.NewRouteSegmentResponse(seg)
		likeCount, _ := rc.routeSegmentRepo.CountLikes(seg.ID)
		res.LikeCount = likeCount
		if user != nil && user.Profile.ID > 0 {
			hasLiked, _ := rc.routeSegmentRepo.HasLiked(seg.ID, user.Profile.ID)
			res.HasLiked = hasLiked
		}
		res.CanEdit = isAdmin || (res.ProfileID != 0 && res.ProfileID == currentProfileID)
		res.CanDelete = isAdmin || (res.ProfileID != 0 && res.ProfileID == currentProfileID)
		results = append(results, res)
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

// GetRouteSegment returns details for a single route segment
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
	id, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	user := currentUser(c)
	rs, statsData, likeCount, hasLiked, err := rc.routeSegmentSvc.GetRouteSegment(c.Request().Context(), user, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return renderApiError(c, http.StatusNotFound, err)
		}
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	canEdit := rc.routeSegmentSvc.CanEditRouteSegment(user, rs)
	detailResp := dto.NewRouteSegmentDetailResponse(rs)
	detailResp.CanEdit = canEdit
	detailResp.CanDelete = canEdit
	detailResp.LikeCount = likeCount
	detailResp.HasLiked = hasLiked

	if statsData != nil {
		detailResp.Stats = buildRouteSegmentStatsResponse(statsData)
	}

	resp := dto.Response[dto.RouteSegmentDetailResponse]{
		Results: detailResp,
	}

	return c.JSON(http.StatusOK, resp)
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
	if len(files) == 0 {
		return renderApiError(c, http.StatusBadRequest, errors.New("no files uploaded"))
	}

	user := currentUser(c)
	var profileID uint64
	if user != nil {
		profileID = user.Profile.ID
	}

	names := form.Value["name"]
	if len(names) == 0 {
		names = form.Value["names"]
	}

	params := service.RouteSegmentUploadParams{
		ProfileID:     profileID,
		Notes:         multipartFormValue(form, c, "notes"),
		Category:      multipartFormValue(form, c, "category"),
		Visibility:    multipartFormValue(form, c, "visibility"),
		Difficulty:    multipartFormValue(form, c, "difficulty"),
		Description:   multipartFormValue(form, c, "description"),
		Bidirectional: multipartFormValue(form, c, "bidirectional"),
		Circular:      multipartFormValue(form, c, "circular"),
		Names:         names,
	}

	uploadedFiles := make([]service.UploadedWorkoutFile, 0, len(files))
	for _, file := range files {
		content, parseErr := uploadedRouteSegmentFile(file)
		if parseErr != nil {
			return renderApiError(c, http.StatusBadRequest, parseErr)
		}
		uploadedFiles = append(uploadedFiles, service.UploadedWorkoutFile{
			Filename: file.Filename,
			Content:  content,
		})
	}

	created, errMsgs := rc.routeSegmentSvc.CreateRouteSegments(c.Request().Context(), user, uploadedFiles, params)

	segments := make([]*dto.RouteSegmentResponse, len(created))
	for i, seg := range created {
		resp := dto.NewRouteSegmentResponse(seg)
		segments[i] = &resp
	}

	resp := dto.Response[dto.RouteSegmentsDetailResponse]{
		Results: segments,
		Errors:  errMsgs,
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

	var params dto.RouteSegmentFromWorkoutRequest
	if err := c.Bind(&params); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	if err := c.Validate(&params); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	user := currentUser(c)
	rs, err := rc.routeSegmentSvc.CreateRouteSegmentFromWorkout(c.Request().Context(), user, workoutID, &params)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return renderApiError(c, http.StatusNotFound, err)
		}
		return renderApiError(c, http.StatusInternalServerError, err)
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
	if err := rc.routeSegmentSvc.DeleteRouteSegment(c.Request().Context(), user, rs); err != nil {
		if err.Error() == "not authorized" {
			return renderApiError(c, http.StatusForbidden, err)
		}
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[map[string]string]{
		Results: map[string]string{"message": "Route segment deleted successfully"},
	}

	return c.JSON(http.StatusOK, resp)
}

// RefreshRouteSegment marks a route segment for match reprocessing
// @Summary      Refresh route segment
// @Tags         route-segments
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "Route segment ID"
// @Produce      json
// @Success      200  {object}  dto.Response[map[string]string]
// @Failure      404  {object}  dto.Response[string]
// @Failure      403  {object}  dto.Response[string]
// @Router       /route-segments/{id}/refresh [post]
func (rc *routeSegmentController) RefreshRouteSegment(c *echo.Context) error {
	rs, err := rc.getRouteSegment(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	user := currentUser(c)
	if err := rc.routeSegmentSvc.RefreshRouteSegment(c.Request().Context(), user, rs); err != nil {
		if errors.Is(err, service.ErrForbidden) {
			return renderApiError(c, http.StatusForbidden, errors.New("forbidden"))
		}
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[map[string]string]{
		Results: map[string]string{"message": "Route segment refreshed successfully"},
	}

	return c.JSON(http.StatusOK, resp)
}

// UpdateRouteSegment updates segment metadata
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
// @Failure      403  {object}  dto.Response[string]
// @Router       /route-segments/{id} [put]
func (rc *routeSegmentController) UpdateRouteSegment(c *echo.Context) error {
	rs, err := rc.getRouteSegment(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	var req dto.RouteSegmentUpdateRequest
	if err := c.Bind(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	user := currentUser(c)
	updated, err := rc.routeSegmentSvc.UpdateRouteSegment(c.Request().Context(), user, rs, &req)
	if err != nil {
		if errors.Is(err, service.ErrForbidden) {
			return renderApiError(c, http.StatusForbidden, errors.New("forbidden"))
		}
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[dto.RouteSegmentDetailResponse]{
		Results: dto.NewRouteSegmentDetailResponse(updated),
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
	if err := rc.routeSegmentSvc.FindRouteSegmentMatches(c.Request().Context(), user, rs); err != nil {
		if errors.Is(err, service.ErrForbidden) {
			return renderApiError(c, http.StatusForbidden, errors.New("forbidden"))
		}
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

	matches, totalCount, err := rc.routeSegmentSvc.GetRouteSegmentMatches(c.Request().Context(), user, rs.ID, query.Sort, query.PerPage, query.GetOffset())
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

	count, err := rc.routeSegmentSvc.LikeRouteSegment(c.Request().Context(), user, rs)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

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

	count, err := rc.routeSegmentSvc.UnlikeRouteSegment(c.Request().Context(), user, rs)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

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
