package controller

import (
	"net/http"
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/container"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model/dto"
	"github.com/labstack/echo/v4"
)

type MeasurementController interface {
	GetMeasurements(c echo.Context) error
	CreateMeasurement(c echo.Context) error
	DeleteMeasurement(c echo.Context) error
}

type measurementController struct {
	context *container.Container
}

func NewMeasurementController(c *container.Container) MeasurementController {
	return &measurementController{context: c}
}

// GetMeasurements returns a paginated list of measurements for the current user
// @Summary      List measurements
// @Tags         measurements
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        page      query     int false "Page"
// @Param        per_page  query     int false "Per page"
// @Produce      json
// @Success      200  {object}  api.PaginatedResponse[dto.MeasurementResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /measurements [get]
func (mc *measurementController) GetMeasurements(c echo.Context) error {
	user := mc.context.GetUser(c)

	var pagination dto.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	pagination.SetDefaults()

	var totalCount int64
	if err := mc.context.GetDB().Model(&model.Measurement{}).Where("user_id = ?", user.ID).Count(&totalCount).Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	var measurements []*model.Measurement
	db := mc.context.GetDB().Where("user_id = ?", user.ID).
		Order("date DESC").
		Limit(pagination.PerPage).
		Offset(pagination.GetOffset())

	if err := db.Find(&measurements).Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	results := dto.NewMeasurementsResponse(measurements)

	resp := dto.PaginatedResponse[dto.MeasurementResponse]{
		Results:    results,
		Page:       pagination.Page,
		PerPage:    pagination.PerPage,
		TotalPages: pagination.CalculateTotalPages(totalCount),
		TotalCount: totalCount,
	}

	return c.JSON(http.StatusOK, resp)
}

// CreateMeasurement creates or updates a measurement for a specific date
// @Summary      Create or update measurement
// @Tags         measurements
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Success      200  {object}  api.Response[dto.MeasurementResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /measurements [post]
func (mc *measurementController) CreateMeasurement(c echo.Context) error {
	user := mc.context.GetUser(c)

	d := &dto.Measurement{Units: user.PreferredUnits()}
	if err := c.Bind(d); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	m, err := user.GetMeasurementForDate(d.Time())
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	d.Update(m)

	if err := m.Save(mc.context.GetDB()); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[dto.MeasurementResponse]{
		Results: dto.NewMeasurementResponse(m),
	}

	return c.JSON(http.StatusOK, resp)
}

// DeleteMeasurement deletes a measurement for a specific date
// @Summary      Delete measurement by date
// @Tags         measurements
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        date  path  string  true  "Date (YYYY-MM-DD)"
// @Produce      json
// @Success      204  {string}  string ""
// @Failure      400  {object}  api.Response[any]
// @Failure      404  {object}  api.Response[any]
// @Router       /measurements/{date} [delete]
func (mc *measurementController) DeleteMeasurement(c echo.Context) error {
	u := mc.context.GetUser(c)

	dateStr := c.Param("date")
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	m, err := u.GetMeasurementForDate(t)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	if err := m.Delete(mc.context.GetDB()); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	return c.NoContent(http.StatusNoContent)
}
