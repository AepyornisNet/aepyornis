package app

import (
	"errors"
	"net/http"
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/api"
	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
	"github.com/labstack/echo/v4"
)

// registerAPIV2UserRoutes wires user routes.
func (a *App) registerAPIV2UserRoutes(e *echo.Group) {
	e.GET("/whoami", a.apiV2WhoamiHandler).Name = "api-v2-whoami"
	e.GET("/totals", a.apiV2TotalsHandler).Name = "api-v2-totals"
	e.GET("/records", a.apiV2RecordsHandler).Name = "api-v2-records"
	e.GET("/records/climbs/ranking", a.apiV2ClimbRecordsRankingHandler).Name = "api-v2-records-climbs-ranking"
	e.GET("/records/ranking", a.apiV2RecordsRankingHandler).Name = "api-v2-records-ranking"
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
// @Param        start  query     string  false  "Start date (YYYY-MM-DD)"
// @Param        end    query     string  false  "End date (YYYY-MM-DD, inclusive)"
// @Produce      json
// @Success      200  {object}  api.Response[api.TotalsResponse]
// @Failure      500  {object}  api.Response[any]
// @Router       /totals [get]
func (a *App) apiV2TotalsHandler(c echo.Context) error {
	user := a.getCurrentUser(c)

	startDate, endDate, err := parseDateRange(c)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	totals, err := user.GetDefaultTotals(startDate, endDate)
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
// @Param        start  query     string  false  "Start date (YYYY-MM-DD)"
// @Param        end    query     string  false  "End date (YYYY-MM-DD, inclusive)"
// @Produce      json
// @Success      200  {object}  api.Response[[]api.WorkoutRecordResponse]
// @Failure      500  {object}  api.Response[any]
// @Router       /records [get]
func (a *App) apiV2RecordsHandler(c echo.Context) error {
	user := a.getCurrentUser(c)

	startDate, endDate, err := parseDateRange(c)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	records, err := user.GetAllRecords(startDate, endDate)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	resp := api.Response[[]api.WorkoutRecordResponse]{
		Results: api.NewWorkoutRecordsResponse(records),
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2RecordsRankingHandler returns ranked workouts for a given distance label
// @Summary      Get ranked distance records
// @Tags         user
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        workout_type  query     string  true   "Workout type (e.g. running)"
// @Param        label         query     string  true   "Distance label (e.g. 10 km)"
// @Param        start         query     string  false  "Start date (YYYY-MM-DD)"
// @Param        end           query     string  false  "End date (YYYY-MM-DD, inclusive)"
// @Param        page          query     int     false  "Page"
// @Param        per_page      query     int     false  "Per page"
// @Produce      json
// @Success      200  {object}  api.PaginatedResponse[api.DistanceRecordResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /records/ranking [get]
func (a *App) apiV2RecordsRankingHandler(c echo.Context) error {
	user := a.getCurrentUser(c)

	workoutType := c.QueryParam("workout_type")
	label := c.QueryParam("label")

	if workoutType == "" || label == "" {
		return a.renderAPIV2Error(c, http.StatusBadRequest, errors.New("workout_type and label are required"))
	}

	wt := database.AsWorkoutType(workoutType)

	var pagination api.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}
	pagination.SetDefaults()

	startDate, endDate, err := parseDateRange(c)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	records, totalCount, err := user.GetDistanceRecordRanking(wt, label, startDate, endDate, pagination.PerPage, pagination.GetOffset())
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	resp := api.PaginatedResponse[api.DistanceRecordResponse]{
		Results:    api.NewDistanceRecordResponses(records),
		Page:       pagination.Page,
		PerPage:    pagination.PerPage,
		TotalPages: pagination.CalculateTotalPages(totalCount),
		TotalCount: totalCount,
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2ClimbRecordsRankingHandler returns ranked climb segments ordered by elevation gain
// @Summary      Get ranked climb records
// @Tags         user
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        workout_type  query     string  true   "Workout type (e.g. cycling)"
// @Param        start         query     string  false  "Start date (YYYY-MM-DD)"
// @Param        end           query     string  false  "End date (YYYY-MM-DD, inclusive)"
// @Param        page          query     int     false  "Page"
// @Param        per_page      query     int     false  "Per page"
// @Produce      json
// @Success      200  {object}  api.PaginatedResponse[api.ClimbRecordResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /records/climbs/ranking [get]
func (a *App) apiV2ClimbRecordsRankingHandler(c echo.Context) error {
	user := a.getCurrentUser(c)

	workoutType := c.QueryParam("workout_type")
	if workoutType == "" {
		return a.renderAPIV2Error(c, http.StatusBadRequest, errors.New("workout_type is required"))
	}

	wt := database.AsWorkoutType(workoutType)

	var pagination api.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}
	pagination.SetDefaults()

	startDate, endDate, err := parseDateRange(c)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	records, totalCount, err := user.GetClimbRanking(wt, startDate, endDate, pagination.PerPage, pagination.GetOffset())
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	resp := api.PaginatedResponse[api.ClimbRecordResponse]{
		Results:    api.NewClimbRecordResponses(records),
		Page:       pagination.Page,
		PerPage:    pagination.PerPage,
		TotalPages: pagination.CalculateTotalPages(totalCount),
		TotalCount: totalCount,
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2UserShowHandler returns a specific user's workout records
// @Summary      Get user profile by ID
// @Tags         user
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        start  query     string  false  "Start date (YYYY-MM-DD)"
// @Param        end    query     string  false  "End date (YYYY-MM-DD, inclusive)"
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

	startDate, endDate, err := parseDateRange(c)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	records, err := u.GetAllRecords(startDate, endDate)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	resp := api.Response[[]api.WorkoutRecordResponse]{
		Results: api.NewWorkoutRecordsResponse(records),
	}

	return c.JSON(http.StatusOK, resp)
}

// parseDateRange parses optional start/end query parameters (YYYY-MM-DD) and returns pointers.
// End date is treated as inclusive by adding 23:59:59 to the parsed date.
func parseDateRange(c echo.Context) (*time.Time, *time.Time, error) {
	const layout = "2006-01-02"
	startStr := c.QueryParam("start")
	endStr := c.QueryParam("end")

	var startDate *time.Time
	var endDate *time.Time

	if startStr != "" {
		s, err := time.Parse(layout, startStr)
		if err != nil {
			return nil, nil, err
		}
		startDate = &s
	}

	if endStr != "" {
		e, err := time.Parse(layout, endStr)
		if err != nil {
			return nil, nil, err
		}
		end := e.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		endDate = &end
	}

	return startDate, endDate, nil
}
