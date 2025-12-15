package app

import (
	"bytes"
	"net/http"
	"path"

	"github.com/jovandeginste/workout-tracker/v2/pkg/api"
	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
	"github.com/labstack/echo/v4"
	"github.com/spf13/cast"
)

func (a *App) registerAPIV2RouteSegmentRoutes(apiGroup *echo.Group) {
	routeSegmentsGroup := apiGroup.Group("/route-segments")
	routeSegmentsGroup.GET("", a.apiV2RouteSegmentsHandler).Name = "api-v2-route-segments"
	routeSegmentsGroup.POST("", a.apiV2RouteSegmentCreateHandler).Name = "api-v2-route-segment-create"
	routeSegmentsGroup.GET("/:id", a.apiV2RouteSegmentGetHandler).Name = "api-v2-route-segment"
	routeSegmentsGroup.PUT("/:id", a.apiV2RouteSegmentUpdateHandler).Name = "api-v2-route-segment-update"
	routeSegmentsGroup.DELETE("/:id", a.apiV2RouteSegmentDeleteHandler).Name = "api-v2-route-segment-delete"
	routeSegmentsGroup.POST("/:id/refresh", a.apiV2RouteSegmentRefreshHandler).Name = "api-v2-route-segment-refresh"
	routeSegmentsGroup.POST("/:id/matches", a.apiV2RouteSegmentFindMatchesHandler).Name = "api-v2-route-segment-matches"
	routeSegmentsGroup.GET("/:id/download", a.apiV2RouteSegmentDownloadHandler).Name = "api-v2-route-segment-download"
	apiGroup.POST("/workouts/:id/route-segment", a.apiV2RouteSegmentCreateFromWorkoutHandler).Name = "api-v2-workout-route-segment-create"
}

// apiV2RouteSegmentsHandler returns a paginated list of route segments
// @Summary      List route segments
// @Tags         route-segments
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Param        page      query  int false "Page"
// @Param        per_page  query  int false "Items per page"
// @Success      200  {object}  api.PaginatedResponse[api.RouteSegmentResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /route-segments [get]
func (a *App) apiV2RouteSegmentsHandler(c echo.Context) error {
	// Parse pagination parameters
	var pagination api.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}
	pagination.SetDefaults()

	// Get total count
	var totalCount int64
	if err := a.db.Model(&database.RouteSegment{}).Count(&totalCount).Error; err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	// Get paginated route segments
	var routeSegments []*database.RouteSegment
	db := a.db.Preload("RouteSegmentMatches").
		Order("created_at DESC").
		Limit(pagination.PerPage).
		Offset(pagination.GetOffset())

	if err := db.Find(&routeSegments).Error; err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	// Convert to API response
	results := api.NewRouteSegmentsResponse(routeSegments)

	resp := api.PaginatedResponse[api.RouteSegmentResponse]{
		Results:    results,
		Page:       pagination.Page,
		PerPage:    pagination.PerPage,
		TotalPages: pagination.CalculateTotalPages(totalCount),
		TotalCount: totalCount,
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2RouteSegmentGetHandler returns a single route segment by ID with full details
// @Summary      Get route segment
// @Tags         route-segments
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "Route segment ID"
// @Produce      json
// @Success      200  {object}  api.Response[api.RouteSegmentDetailResponse]
// @Failure      404  {object}  api.Response[any]
// @Router       /route-segments/{id} [get]
func (a *App) apiV2RouteSegmentGetHandler(c echo.Context) error {
	rs, err := a.getRouteSegment(c)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusNotFound, err)
	}

	resp := api.Response[api.RouteSegmentDetailResponse]{
		Results: api.NewRouteSegmentDetailResponse(rs),
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2RouteSegmentCreateHandler uploads one or more route segment files
// @Summary      Create route segment
// @Tags         route-segments
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Accept       multipart/form-data
// @Produce      json
// @Param        file   formData  file   true  "GPX file"
// @Param        notes  formData  string false "Notes"
// @Success      201  {object}  api.Response[api.RouteSegmentsDetailResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /route-segments [post]
func (a *App) apiV2RouteSegmentCreateHandler(c echo.Context) error {
	form, err := c.MultipartForm()
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	files := form.File["file"]

	errMsg := []string{}

	segments := []*api.RouteSegmentResponse{}
	for _, file := range files {
		content, parseErr := uploadedFile(file)
		if parseErr != nil {
			errMsg = append(errMsg, parseErr.Error())
			continue
		}

		notes := c.FormValue("notes")

		w, addErr := database.AddRouteSegment(a.db, notes, file.Filename, content)
		if addErr != nil {
			errMsg = append(errMsg, addErr.Error())
			continue
		}

		resp := api.NewRouteSegmentResponse(w)
		segments = append(segments, &resp)
	}

	resp := api.Response[api.RouteSegmentsDetailResponse]{
		Results: segments,
		Errors:  errMsg,
	}

	return c.JSON(http.StatusCreated, resp)
}

// apiV2RouteSegmentCreateFromWorkoutHandler creates a route segment from a workout
// @Summary      Create route segment from workout
// @Tags         route-segments
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "Workout ID"
// @Accept       json
// @Produce      json
// @Success      201  {object}  api.Response[api.RouteSegmentDetailResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      404  {object}  api.Response[any]
// @Router       /workouts/{id}/route-segment [post]
func (a *App) apiV2RouteSegmentCreateFromWorkoutHandler(c echo.Context) error {
	workoutID, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	workout, err := database.GetWorkoutDetails(a.db, workoutID)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusNotFound, err)
	}

	var params database.RoutSegmentCreationParams
	if err := c.Bind(&params); err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	content, err := database.RouteSegmentFromPoints(workout, &params)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	rs, err := database.AddRouteSegment(a.db, "", params.Filename(), content)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	resp := api.Response[api.RouteSegmentDetailResponse]{
		Results: api.NewRouteSegmentDetailResponse(rs),
	}

	return c.JSON(http.StatusCreated, resp)
}

// apiV2RouteSegmentDeleteHandler deletes a route segment
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
func (a *App) apiV2RouteSegmentDeleteHandler(c echo.Context) error {
	rs, err := a.getRouteSegment(c)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusNotFound, err)
	}

	if err := rs.Delete(a.db); err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	resp := api.Response[map[string]string]{
		Results: map[string]string{"message": "Route segment deleted successfully"},
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2RouteSegmentRefreshHandler marks a route segment for refresh
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
func (a *App) apiV2RouteSegmentRefreshHandler(c echo.Context) error {
	rs, err := a.getRouteSegment(c)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusNotFound, err)
	}

	if err := rs.UpdateFromContent(); err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	if err := rs.Save(a.db); err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	resp := api.Response[map[string]string]{
		Results: map[string]string{"message": "Route segment refreshed successfully"},
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2RouteSegmentUpdateHandler updates a route segment
// @Summary      Update route segment
// @Tags         route-segments
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "Route segment ID"
// @Accept       json
// @Produce      json
// @Success      200  {object}  api.Response[api.RouteSegmentDetailResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      404  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /route-segments/{id} [put]
func (a *App) apiV2RouteSegmentUpdateHandler(c echo.Context) error {
	rs, err := a.getRouteSegment(c)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusNotFound, err)
	}

	type UpdateParams struct {
		Name          string `json:"name"`
		Notes         string `json:"notes"`
		Bidirectional bool   `json:"bidirectional"`
		Circular      bool   `json:"circular"`
	}

	var params UpdateParams
	if err := c.Bind(&params); err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	rs.Name = params.Name
	rs.Notes = params.Notes
	rs.Bidirectional = params.Bidirectional
	rs.Circular = params.Circular
	rs.Dirty = true

	if err := rs.Save(a.db); err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	resp := api.Response[api.RouteSegmentDetailResponse]{
		Results: api.NewRouteSegmentDetailResponse(rs),
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2RouteSegmentDownloadHandler downloads the original route segment file
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
func (a *App) apiV2RouteSegmentDownloadHandler(c echo.Context) error {
	rs, err := a.getRouteSegment(c)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusNotFound, err)
	}

	basename := path.Base(rs.Filename)

	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename=\""+basename+"\"")

	return c.Stream(http.StatusOK, "application/binary", bytes.NewReader(rs.Content))
}

// apiV2RouteSegmentFindMatchesHandler finds matching workouts for a route segment
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
func (a *App) apiV2RouteSegmentFindMatchesHandler(c echo.Context) error {
	rs, err := a.getRouteSegment(c)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusNotFound, err)
	}

	rs.Dirty = true
	if err := rs.Save(a.db); err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	resp := api.Response[map[string]string]{
		Results: map[string]string{"message": "Finding matches in background"},
	}

	return c.JSON(http.StatusOK, resp)
}
