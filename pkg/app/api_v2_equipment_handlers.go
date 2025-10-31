package app

import (
	"net/http"

	"github.com/jovandeginste/workout-tracker/v2/pkg/api"
	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
	"github.com/labstack/echo/v4"
	"github.com/spf13/cast"
)

func (a *App) registerAPIV2EquipmentRoutes(apiGroup *echo.Group) {
	apiGroup.GET("/equipment", a.apiV2EquipmentHandler).Name = "api-v2-equipment"
	apiGroup.GET("/equipment/:id", a.apiV2EquipmentGetHandler).Name = "api-v2-equipment-get"
	apiGroup.POST("/equipment", a.apiV2EquipmentCreateHandler).Name = "api-v2-equipment-create"
	apiGroup.PUT("/equipment/:id", a.apiV2EquipmentUpdateHandler).Name = "api-v2-equipment-update"
	apiGroup.DELETE("/equipment/:id", a.apiV2EquipmentDeleteHandler).Name = "api-v2-equipment-delete"
}

// apiV2EquipmentHandler returns a paginated list of equipment for the current user
func (a *App) apiV2EquipmentHandler(c echo.Context) error {
	user := a.getCurrentUser(c)

	// Parse pagination parameters
	var pagination api.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}
	pagination.SetDefaults()

	// Get total count
	var totalCount int64
	if err := a.db.Model(&database.Equipment{}).Where(&database.Equipment{UserID: user.ID}).Count(&totalCount).Error; err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	// Get paginated equipment
	var equipment []*database.Equipment
	db := a.db.Where(&database.Equipment{UserID: user.ID}).
		Order("name DESC").
		Limit(pagination.PerPage).
		Offset(pagination.GetOffset())

	if err := db.Find(&equipment).Error; err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	// Convert to API response
	results := api.NewEquipmentListResponse(equipment)

	resp := api.PaginatedResponse[api.EquipmentResponse]{
		Results:    results,
		Page:       pagination.Page,
		PerPage:    pagination.PerPage,
		TotalPages: pagination.CalculateTotalPages(totalCount),
		TotalCount: totalCount,
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2EquipmentGetHandler returns a single equipment by ID
func (a *App) apiV2EquipmentGetHandler(c echo.Context) error {
	user := a.getCurrentUser(c)

	id, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	e, err := database.GetEquipment(a.db, id)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusNotFound, err)
	}

	if e.UserID != user.ID {
		return a.renderAPIV2Error(c, http.StatusForbidden, api.ErrNotAuthorized)
	}

	resp := api.Response[api.EquipmentResponse]{
		Results: api.NewEquipmentResponse(e),
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2EquipmentCreateHandler creates a new equipment
func (a *App) apiV2EquipmentCreateHandler(c echo.Context) error {
	user := a.getCurrentUser(c)

	var e database.Equipment
	if err := c.Bind(&e); err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	e.UserID = user.ID

	if err := e.Save(a.db); err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	resp := api.Response[api.EquipmentResponse]{
		Results: api.NewEquipmentResponse(&e),
	}

	return c.JSON(http.StatusCreated, resp)
}

// apiV2EquipmentUpdateHandler updates an existing equipment
func (a *App) apiV2EquipmentUpdateHandler(c echo.Context) error {
	user := a.getCurrentUser(c)

	id, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	e, err := database.GetEquipment(a.db, id)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusNotFound, err)
	}

	e.DefaultFor = nil

	if e.UserID != user.ID {
		return a.renderAPIV2Error(c, http.StatusForbidden, api.ErrNotAuthorized)
	}

	if err := c.Bind(e); err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	// Ensure user can't change ownership
	e.UserID = user.ID

	if err := e.Save(a.db); err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	resp := api.Response[api.EquipmentResponse]{
		Results: api.NewEquipmentResponse(e),
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2EquipmentDeleteHandler deletes an equipment
func (a *App) apiV2EquipmentDeleteHandler(c echo.Context) error {
	user := a.getCurrentUser(c)

	id, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	e, err := database.GetEquipment(a.db, id)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusNotFound, err)
	}

	if e.UserID != user.ID {
		return a.renderAPIV2Error(c, http.StatusForbidden, api.ErrNotAuthorized)
	}

	if err := e.Delete(a.db); err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	return c.NoContent(http.StatusNoContent)
}
