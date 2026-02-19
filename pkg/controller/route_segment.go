package controller

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"path"

	"github.com/jovandeginste/workout-tracker/v2/pkg/container"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model/dto"
	"github.com/jovandeginste/workout-tracker/v2/pkg/worker"
	"github.com/labstack/echo/v4"
	"github.com/spf13/cast"
)

type RouteSegmentController interface {
	GetRouteSegments(c echo.Context) error
	GetRouteSegment(c echo.Context) error
	CreateRouteSegment(c echo.Context) error
	CreateRouteSegmentFromWorkout(c echo.Context) error
	DeleteRouteSegment(c echo.Context) error
	RefreshRouteSegment(c echo.Context) error
	UpdateRouteSegment(c echo.Context) error
	DownloadRouteSegment(c echo.Context) error
	FindRouteSegmentMatches(c echo.Context) error
}

type routeSegmentController struct {
	context *container.Container
}

func NewRouteSegmentController(c *container.Container) RouteSegmentController {
	return &routeSegmentController{context: c}
}

func (rc *routeSegmentController) getRouteSegment(c echo.Context) (*model.RouteSegment, error) {
	id, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return nil, err
	}

	rs, err := model.GetRouteSegment(rc.context.GetDB(), id)
	if err != nil {
		return nil, err
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
// @Success      200  {object}  api.PaginatedResponse[dto.RouteSegmentResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /route-segments [get]
func (rc *routeSegmentController) GetRouteSegments(c echo.Context) error {
	var pagination dto.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	pagination.SetDefaults()

	var totalCount int64
	if err := rc.context.GetDB().Model(&model.RouteSegment{}).Count(&totalCount).Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	var routeSegments []*model.RouteSegment
	db := rc.context.GetDB().Preload("RouteSegmentMatches").
		Order("created_at DESC").
		Limit(pagination.PerPage).
		Offset(pagination.GetOffset())

	if err := db.Find(&routeSegments).Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	results := dto.NewRouteSegmentsResponse(routeSegments)

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
// @Success      200  {object}  api.Response[dto.RouteSegmentDetailResponse]
// @Failure      404  {object}  api.Response[any]
// @Router       /route-segments/{id} [get]
func (rc *routeSegmentController) GetRouteSegment(c echo.Context) error {
	rs, err := rc.getRouteSegment(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	resp := dto.Response[dto.RouteSegmentDetailResponse]{
		Results: dto.NewRouteSegmentDetailResponse(rs),
	}

	return c.JSON(http.StatusOK, resp)
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
// @Success      201  {object}  api.Response[dto.RouteSegmentsDetailResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /route-segments [post]
func (rc *routeSegmentController) CreateRouteSegment(c echo.Context) error {
	form, err := c.MultipartForm()
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	files := form.File["file"]
	errMsg := []string{}

	segments := []*dto.RouteSegmentResponse{}
	for _, file := range files {
		content, parseErr := uploadedRouteSegmentFile(file)
		if parseErr != nil {
			errMsg = append(errMsg, parseErr.Error())
			continue
		}

		notes := c.FormValue("notes")

		w, addErr := model.AddRouteSegment(rc.context.GetDB(), notes, file.Filename, content)
		if addErr != nil {
			errMsg = append(errMsg, addErr.Error())
			continue
		}

		resp := dto.NewRouteSegmentResponse(w)
		segments = append(segments, &resp)

		if err := worker.EnqueueRouteSegmentUpdate(c.Request().Context(), rc.context.GetGueClient(), w.ID); err != nil {
			rc.context.Logger().Error("Failed to enqueue route segment update", "route_segment_id", w.ID, "error", err)
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
// @Success      201  {object}  api.Response[dto.RouteSegmentDetailResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      404  {object}  api.Response[any]
// @Router       /workouts/{id}/route-segment [post]
func (rc *routeSegmentController) CreateRouteSegmentFromWorkout(c echo.Context) error {
	workoutID, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	workout, err := model.GetWorkoutDetails(rc.context.GetDB(), workoutID)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	var params model.RoutSegmentCreationParams
	if err := c.Bind(&params); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	content, err := model.RouteSegmentFromPoints(workout, &params)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	rs, err := model.AddRouteSegment(rc.context.GetDB(), "", params.Filename(), content)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if err := worker.EnqueueRouteSegmentUpdate(c.Request().Context(), rc.context.GetGueClient(), rs.ID); err != nil {
		rc.context.Logger().Error("Failed to enqueue route segment update", "route_segment_id", rs.ID, "error", err)
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
// @Success      200  {object}  api.Response[map[string]string]
// @Failure      404  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /route-segments/{id} [delete]
func (rc *routeSegmentController) DeleteRouteSegment(c echo.Context) error {
	rs, err := rc.getRouteSegment(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	if err := rs.Delete(rc.context.GetDB()); err != nil {
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
// @Success      200  {object}  api.Response[map[string]string]
// @Failure      404  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /route-segments/{id}/refresh [post]
func (rc *routeSegmentController) RefreshRouteSegment(c echo.Context) error {
	rs, err := rc.getRouteSegment(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	if err := rs.UpdateFromContent(); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if err := rs.Save(rc.context.GetDB()); err != nil {
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
// @Success      200  {object}  api.Response[dto.RouteSegmentDetailResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      404  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /route-segments/{id} [put]
func (rc *routeSegmentController) UpdateRouteSegment(c echo.Context) error {
	rs, err := rc.getRouteSegment(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	type updateParams struct {
		Name          string `json:"name"`
		Notes         string `json:"notes"`
		Bidirectional bool   `json:"bidirectional"`
		Circular      bool   `json:"circular"`
	}

	var params updateParams
	if err := c.Bind(&params); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	rs.Name = params.Name
	rs.Notes = params.Notes
	rs.Bidirectional = params.Bidirectional
	rs.Circular = params.Circular
	rs.Dirty = true

	if err := rs.Save(rc.context.GetDB()); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if err := worker.EnqueueRouteSegmentUpdate(c.Request().Context(), rc.context.GetGueClient(), rs.ID); err != nil {
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
// @Failure      404  {object}  api.Response[any]
// @Router       /route-segments/{id}/download [get]
func (rc *routeSegmentController) DownloadRouteSegment(c echo.Context) error {
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
// @Success      200  {object}  api.Response[map[string]string]
// @Failure      404  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /route-segments/{id}/matches [post]
func (rc *routeSegmentController) FindRouteSegmentMatches(c echo.Context) error {
	rs, err := rc.getRouteSegment(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	rs.Dirty = true
	if err := rs.Save(rc.context.GetDB()); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if err := worker.EnqueueRouteSegmentUpdate(c.Request().Context(), rc.context.GetGueClient(), rs.ID); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[map[string]string]{
		Results: map[string]string{"message": "Finding matches in background"},
	}

	return c.JSON(http.StatusOK, resp)
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
