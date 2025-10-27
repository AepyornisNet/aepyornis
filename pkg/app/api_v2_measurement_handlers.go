package app

import (
	"net/http"
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/api"
	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
	"github.com/labstack/echo/v4"
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

	d := &Measurement{units: a.getCurrentUser(c).PreferredUnits()}
	if err := c.Bind(d); err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	// Get or create measurement for this date
	m, err := user.GetMeasurementForDate(d.Time())
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	d.Update(m)

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
	u := a.getCurrentUser(c)

	dateStr := c.Param("date")
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	// Find measurement for this date
	m, err := u.GetMeasurementForDate(t)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusNotFound, err)
	}

	if err := m.Delete(a.db); err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	return c.NoContent(http.StatusNoContent)
}
