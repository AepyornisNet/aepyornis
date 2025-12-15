package app

import (
	"net/http"

	"github.com/jovandeginste/workout-tracker/v2/pkg/api"
	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
	"github.com/labstack/echo/v4"
)

func (a *App) registerAPIV2StatisticsRoutes(apiGroup *echo.Group) {
	apiGroup.GET("/statistics", a.apiV2StatisticsHandler).Name = "api-v2-statistics"
}

// apiV2StatisticsHandler returns user's workout statistics
// @Summary      Get workout statistics
// @Tags         statistics
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Param        since  query  string false "Relative start (e.g. '1 year')"
// @Param        per    query  string false "Aggregation period (day|week|month|year)"
// @Success      200  {object}  api.Response[api.StatisticsResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /statistics [get]
func (a *App) apiV2StatisticsHandler(c echo.Context) error {
	user := a.getCurrentUser(c)

	// Parse statistics config from query parameters
	var statConfig database.StatConfig
	if err := c.Bind(&statConfig); err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	// Set defaults if not provided
	if statConfig.Since == "" {
		statConfig.Since = "1 year"
	}
	if statConfig.Per == "" {
		statConfig.Per = "month"
	}

	statistics, err := user.GetStatisticsFor(statConfig.Since, statConfig.Per)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	resp := api.Response[api.StatisticsResponse]{
		Results: api.NewStatisticsResponse(statistics),
	}

	return c.JSON(http.StatusOK, resp)
}
