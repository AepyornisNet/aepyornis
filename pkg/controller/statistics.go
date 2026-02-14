package controller

import (
	"net/http"

	"github.com/jovandeginste/workout-tracker/v2/pkg/model/dto"
	"github.com/jovandeginste/workout-tracker/v2/pkg/container"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/labstack/echo/v4"
)

type StatisticsController interface {
	GetStatistics(c echo.Context) error
}

type statisticsController struct {
	context *container.Container
}

func NewStatisticsController(c *container.Container) StatisticsController {
	return &statisticsController{context: c}
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
// @Success      200  {object}  api.Response[dto.StatisticsResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /statistics [get]
func (sc *statisticsController) GetStatistics(c echo.Context) error {
	user := sc.context.GetUser(c)

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
