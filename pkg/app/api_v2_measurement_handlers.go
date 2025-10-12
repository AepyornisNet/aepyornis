package app

import (
	"net/http"
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/api"
	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
)

func (a *App) registerAPIV2MeasurementRoutes(apiGroup *echo.Group) {
	apiGroup.GET("/measurements", a.apiV2MeasurementsHandler).Name = "api-v2-measurements"
	apiGroup.POST("/measurements", a.apiV2MeasurementCreateHandler).Name = "api-v2-measurements-create"
	apiGroup.DELETE("/measurements/:date", a.apiV2MeasurementDeleteHandler).Name = "api-v2-measurements-delete"
}

// apiV2MeasurementsHandler returns a paginated list of measurements for the current user
func (a *App) apiV2MeasurementsHandler(c echo.Context) error {
	user := a.getCurrentUser(c)

	// Parse pagination parameters
	var pagination api.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}
	pagination.SetDefaults()

	// Get total count
	var totalCount int64
	if err := a.db.Model(&database.Measurement{}).Where("user_id = ?", user.ID).Count(&totalCount).Error; err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	// Get paginated measurements
	var measurements []*database.Measurement
	db := a.db.Where("user_id = ?", user.ID).
		Order("date DESC").
		Limit(pagination.PerPage).
		Offset(pagination.GetOffset())

	if err := db.Find(&measurements).Error; err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	// Convert to API response
	results := api.NewMeasurementsResponse(measurements)

	resp := api.PaginatedResponse[api.MeasurementResponse]{
		Results:    results,
		Page:       pagination.Page,
		PerPage:    pagination.PerPage,
		TotalPages: pagination.CalculateTotalPages(totalCount),
		TotalCount: totalCount,
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2MeasurementCreateHandler creates or updates a measurement for a specific date
func (a *App) apiV2MeasurementCreateHandler(c echo.Context) error {
	user := a.getCurrentUser(c)

	var req struct {
		Date   string   `json:"date"`
		Weight *float64 `json:"weight,omitempty"`
		Height *float64 `json:"height,omitempty"`
		Steps  *int     `json:"steps,omitempty"`
	}

	if err := c.Bind(&req); err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	// Parse date
	t, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	// Get or create measurement for this date
	m, err := user.GetMeasurementForDate(t)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	// Update fields
	if req.Weight != nil {
		m.Weight = *req.Weight
	}
	if req.Height != nil {
		m.Height = *req.Height
	}
	if req.Steps != nil {
		m.Steps = float64(*req.Steps)
	}

	if err := m.Save(a.db); err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	resp := api.Response[api.MeasurementResponse]{
		Results: api.NewMeasurementResponse(m),
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2MeasurementDeleteHandler deletes a measurement for a specific date
func (a *App) apiV2MeasurementDeleteHandler(c echo.Context) error {
	user := a.getCurrentUser(c)

	dateStr := c.Param("date")
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	// Find measurement for this date
	var m database.Measurement
	if err := a.db.Where("user_id = ? AND date = ?", user.ID, datatypes.Date(t)).First(&m).Error; err != nil {
		return a.renderAPIV2Error(c, http.StatusNotFound, err)
	}

	if err := m.Delete(a.db); err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	return c.NoContent(http.StatusNoContent)
}
