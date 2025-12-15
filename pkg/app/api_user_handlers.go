package app

import (
	"net/http"

	"github.com/jovandeginste/workout-tracker/v2/pkg/api"
	"github.com/labstack/echo/v4"
)

// registerAPIV2UserRoutes wires user routes.
func (a *App) registerAPIV2UserRoutes(e *echo.Group) {
	e.GET("/whoami", a.apiV2WhoamiHandler).Name = "api-v2-whoami"
	e.GET("/totals", a.apiV2TotalsHandler).Name = "api-v2-totals"
	e.GET("/records", a.apiV2RecordsHandler).Name = "api-v2-records"
	e.GET("/:id", a.apiV2UserShowHandler).Name = "api-v2-user-show"
}

// apiV2WhoamiHandler returns current user information
// @Summary      Get current user profile
// @Tags         user
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Success      200  {object}  api.Response[api.UserProfileResponse]
// @Failure      401  {object}  api.Response[any]
// @Router       /whoami [get]
func (a *App) apiV2WhoamiHandler(c echo.Context) error {
	user := a.getCurrentUser(c)

	resp := api.Response[api.UserProfileResponse]{
		Results: api.NewUserProfileResponse(user),
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2TotalsHandler returns user's workout totals
// @Summary      Get workout totals
// @Tags         user
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Success      200  {object}  api.Response[api.TotalsResponse]
// @Failure      500  {object}  api.Response[any]
// @Router       /totals [get]
func (a *App) apiV2TotalsHandler(c echo.Context) error {
	user := a.getCurrentUser(c)

	totals, err := user.GetDefaultTotals()
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	resp := api.Response[api.TotalsResponse]{
		Results: api.NewTotalsResponse(totals),
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2RecordsHandler returns user's workout records
// @Summary      Get workout records
// @Tags         user
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Success      200  {object}  api.Response[[]api.WorkoutRecordResponse]
// @Failure      500  {object}  api.Response[any]
// @Router       /records [get]
func (a *App) apiV2RecordsHandler(c echo.Context) error {
	user := a.getCurrentUser(c)

	records, err := user.GetAllRecords()
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	resp := api.Response[[]api.WorkoutRecordResponse]{
		Results: api.NewWorkoutRecordsResponse(records),
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2UserShowHandler returns a specific user's workout records
// @Summary      Get user profile by ID
// @Tags         user
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path      int  true  "User ID"
// @Produce      json
// @Success      200  {object}  api.Response[[]api.WorkoutRecordResponse]
// @Failure      403  {object}  api.Response[any]
// @Failure      404  {object}  api.Response[any]
// @Router       /{id} [get]
// TODO: Add more data. This will be used for public profiles.
func (a *App) apiV2UserShowHandler(c echo.Context) error {
	u, err := a.getUser(c)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	if u.IsAnonymous() {
		return a.renderAPIV2Error(c, http.StatusForbidden, api.ErrNotAuthorized)
	}

	records, err := u.GetAllRecords()
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	resp := api.Response[[]api.WorkoutRecordResponse]{
		Results: api.NewWorkoutRecordsResponse(records),
	}

	return c.JSON(http.StatusOK, resp)
}
