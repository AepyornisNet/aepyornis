package app

import (
	"net/http"
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/api"
	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
	"github.com/jovandeginste/workout-tracker/v2/pkg/templatehelpers"
	"github.com/labstack/echo/v4"
)

func (a *App) registerAPIV2MeasurementRoutes(apiGroup *echo.Group) {
	apiGroup.GET("/measurements", a.apiV2MeasurementsHandler).Name = "api-v2-measurements"
	apiGroup.POST("/measurements", a.apiV2MeasurementCreateHandler).Name = "api-v2-measurements-create"
	apiGroup.DELETE("/measurements/:date", a.apiV2MeasurementDeleteHandler).Name = "api-v2-measurements-delete"
}

// apiV2MeasurementsHandler returns a paginated list of measurements for the current user
// @Summary      List measurements
// @Tags         measurements
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        page      query     int false "Page"
// @Param        per_page  query     int false "Per page"
// @Produce      json
// @Success      200  {object}  api.PaginatedResponse[api.MeasurementResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /measurements [get]
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
// @Summary      Create or update measurement
// @Tags         measurements
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Success      200  {object}  api.Response[api.MeasurementResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /measurements [post]
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

type Measurement struct {
	Date       string  `form:"date" json:"date"`               // The date of the measurement
	Steps      float64 `form:"steps" json:"steps"`             // The number of steps taken
	WeightUnit string  `form:"weight_unit" json:"weight_unit"` // The unit of the weight (or the user's preferred unit)
	HeightUnit string  `form:"height_unit" json:"height_unit"` // The unit of the height (or the user's preferred unit)

	Weight           float64 `form:"weight" json:"weight"`                         // The weight of the user, in kilograms
	Height           float64 `form:"height" json:"height"`                         // The height of the user, in centimeter
	FTP              float64 `form:"ftp" json:"ftp"`                               // Functional Threshold Power, in watts
	RestingHeartRate float64 `form:"resting_heart_rate" json:"resting_heart_rate"` // Resting heart rate, in bpm
	MaxHeartRate     float64 `form:"max_heart_rate" json:"max_heart_rate"`         // Maximum heart rate, in bpm

	units *database.UserPreferredUnits
}

func (m *Measurement) Time() time.Time {
	if m.Date == "" {
		return time.Now()
	}

	d, err := time.Parse("2006-01-02", m.Date)
	if err != nil {
		return time.Now()
	}

	return d
}

func (m *Measurement) ToSteps() *float64 {
	if m.Steps == 0 {
		return nil
	}

	d := m.Steps

	return &d
}

func (m *Measurement) ToFTP() *float64 {
	if m.FTP == 0 {
		return nil
	}

	d := m.FTP

	return &d
}

func (m *Measurement) ToRestingHeartRate() *float64 {
	if m.RestingHeartRate == 0 {
		return nil
	}

	d := m.RestingHeartRate

	return &d
}

func (m *Measurement) ToMaxHeartRate() *float64 {
	if m.MaxHeartRate == 0 {
		return nil
	}

	d := m.MaxHeartRate

	return &d
}

func (m *Measurement) ToHeight() *float64 {
	if m.Height == 0 {
		return nil
	}

	if m.HeightUnit == "" {
		m.HeightUnit = m.units.Height()
	}

	d := templatehelpers.HeightToDatabase(m.Height, m.HeightUnit)

	return &d
}

func (m *Measurement) ToWeight() *float64 {
	if m.Weight == 0 {
		return nil
	}

	if m.WeightUnit == "" {
		m.WeightUnit = m.units.Weight()
	}

	d := templatehelpers.WeightToDatabase(m.Weight, m.WeightUnit)

	return &d
}

func (m *Measurement) Update(measurement *database.Measurement) {
	setIfNotNil(&measurement.Weight, m.ToWeight())
	setIfNotNil(&measurement.Height, m.ToHeight())
	setIfNotNil(&measurement.Steps, m.ToSteps())
	setIfNotNil(&measurement.FTP, m.ToFTP())
	setIfNotNil(&measurement.RestingHeartRate, m.ToRestingHeartRate())
	setIfNotNil(&measurement.MaxHeartRate, m.ToMaxHeartRate())
}
