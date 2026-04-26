package controller

import (
	"net/http"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/model/dto"
	"github.com/labstack/echo/v4"
	"github.com/samber/do/v2"
)

type StatisticsController interface {
	GetStatistics(c echo.Context) error
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
// @Failure      400  {object}  dto.Response[any]
// @Failure      500  {object}  dto.Response[any]
// @Router       /statistics [get]
func (sc *statisticsController) GetStatistics(c echo.Context) error {
	user := currentUser(c)

	var statConfig model.StatConfig
	if err := c.Bind(&statConfig); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	if statConfig.Since == "" {
		statConfig.Since = "1 year"
	}

	if statConfig.Per == "" {
		statConfig.Per = "month"
	}

	statistics, err := user.GetStatisticsFor(statConfig.Since, statConfig.Per)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[dto.StatisticsResponse]{
		Results: dto.NewStatisticsResponse(statistics),
	}

	return c.JSON(http.StatusOK, resp)
}
