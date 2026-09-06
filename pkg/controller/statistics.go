package controller

import (
	"net/http"

	"github.com/AepyornisNet/aepyornis/pkg/model/dto"
	"github.com/labstack/echo/v5"
	"github.com/samber/do/v2"
)

type StatisticsController interface {
	GetStatistics(c *echo.Context) error
}

type statisticsController struct{}

func NewStatisticsController(_ do.Injector) StatisticsController {
	return &statisticsController{}
}

// GetStatistics returns user's workout statistics
// @Summary      Get workout statistics
// @Tags         statistics
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Param        since  query  string false "Relative start (e.g. '1 year')"
// @Param        per    query  string false "Aggregation period (day|week|month|year)"
// @Success      200  {object}  dto.Response[dto.StatisticsResponse]
// @Failure      400  {object}  dto.Response[string]
// @Failure      500  {object}  dto.Response[string]
// @Router       /statistics [get]
func (sc *statisticsController) GetStatistics(c *echo.Context) error {
	user := currentUser(c)

	var req dto.StatisticsRequest
	if err := c.Bind(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	since := req.Since
	if since == "" {
		since = "1 year"
	}

	per := req.Per
	if per == "" {
		per = "month"
	}

	statistics, err := user.GetStatisticsFor(since, per)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[dto.StatisticsResponse]{
		Results: dto.NewStatisticsResponse(statistics),
	}

	return c.JSON(http.StatusOK, resp)
}
