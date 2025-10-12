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
	apiGroup.GET("/route-segments", a.apiV2RouteSegmentsHandler).Name = "api-v2-route-segments"
	apiGroup.GET("/route-segments/:id", a.apiV2RouteSegmentGetHandler).Name = "api-v2-route-segment"
	apiGroup.PUT("/route-segments/:id", a.apiV2RouteSegmentUpdateHandler).Name = "api-v2-route-segment-update"
	apiGroup.DELETE("/route-segments/:id", a.apiV2RouteSegmentDeleteHandler).Name = "api-v2-route-segment-delete"
	apiGroup.POST("/route-segments/:id/refresh", a.apiV2RouteSegmentRefreshHandler).Name = "api-v2-route-segment-refresh"
	apiGroup.POST("/route-segments/:id/matches", a.apiV2RouteSegmentFindMatchesHandler).Name = "api-v2-route-segment-matches"
	apiGroup.GET("/route-segments/:id/download", a.apiV2RouteSegmentDownloadHandler).Name = "api-v2-route-segment-download"
	apiGroup.POST("/workouts/:id/route-segment", a.apiV2RouteSegmentCreateFromWorkoutHandler).Name = "api-v2-workout-route-segment-create"
}

// apiV2RouteSegmentsHandler returns a paginated list of route segments
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
func (a *App) apiV2RouteSegmentGetHandler(c echo.Context) error {
	id, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	rs, err := database.GetRouteSegment(a.db, id)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusNotFound, err)
	}

	resp := api.Response[api.RouteSegmentDetailResponse]{
		Results: api.NewRouteSegmentDetailResponse(rs),
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2RouteSegmentCreateFromWorkoutHandler creates a route segment from a workout
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
func (a *App) apiV2RouteSegmentDeleteHandler(c echo.Context) error {
	id, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	rs, err := database.GetRouteSegment(a.db, id)
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
func (a *App) apiV2RouteSegmentRefreshHandler(c echo.Context) error {
	id, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	rs, err := database.GetRouteSegment(a.db, id)
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
func (a *App) apiV2RouteSegmentUpdateHandler(c echo.Context) error {
	id, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	rs, err := database.GetRouteSegment(a.db, id)
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
func (a *App) apiV2RouteSegmentDownloadHandler(c echo.Context) error {
	id, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	rs, err := database.GetRouteSegment(a.db, id)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusNotFound, err)
	}

	basename := path.Base(rs.Filename)

	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename=\""+basename+"\"")

	return c.Stream(http.StatusOK, "application/binary", bytes.NewReader(rs.Content))
}

// apiV2RouteSegmentFindMatchesHandler finds matching workouts for a route segment
func (a *App) apiV2RouteSegmentFindMatchesHandler(c echo.Context) error {
	id, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	rs, err := database.GetRouteSegment(a.db, id)
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
