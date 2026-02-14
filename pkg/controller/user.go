package controller

import (
	"errors"
	"net/http"
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/model/dto"
	"github.com/jovandeginste/workout-tracker/v2/pkg/container"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/labstack/echo/v4"
	"github.com/spf13/cast"
)

type UserController interface {
	GetWhoami(c echo.Context) error
	GetTotals(c echo.Context) error
	GetRecords(c echo.Context) error
	GetRecordsRanking(c echo.Context) error
	GetClimbRecordsRanking(c echo.Context) error
	GetUserByID(c echo.Context) error
}

type userController struct {
	context *container.Container
}

func NewUserController(c *container.Container) UserController {
	return &userController{context: c}
}

// GetWhoami returns current user information
// @Summary      Get current user profile
// @Tags         user
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Success      200  {object}  api.Response[dto.UserProfileResponse]
// @Failure      401  {object}  api.Response[any]
// @Router       /whoami [get]
func (uc *userController) GetWhoami(c echo.Context) error {
	user := uc.context.GetUser(c)

	resp := dto.Response[dto.UserProfileResponse]{
		Results: dto.NewUserProfileResponse(user),
	}

	return c.JSON(http.StatusOK, resp)
}

// GetTotals returns user's workout totals
// @Summary      Get workout totals
// @Tags         user
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        start  query     string  false  "Start date (YYYY-MM-DD)"
// @Param        end    query     string  false  "End date (YYYY-MM-DD, inclusive)"
// @Produce      json
// @Success      200  {object}  api.Response[dto.TotalsResponse]
// @Failure      500  {object}  api.Response[any]
// @Router       /totals [get]
func (uc *userController) GetTotals(c echo.Context) error {
	user := uc.context.GetUser(c)

	startDate, endDate, err := parseDateRange(c)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	totals, err := user.GetDefaultTotals(startDate, endDate)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[dto.TotalsResponse]{
		Results: dto.NewTotalsResponse(totals),
	}

	return c.JSON(http.StatusOK, resp)
}

// GetRecords returns user's workout records
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
func (uc *userController) GetRecords(c echo.Context) error {
	user := uc.context.GetUser(c)

	startDate, endDate, err := parseDateRange(c)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	records, err := user.GetAllRecords(startDate, endDate)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[[]dto.WorkoutRecordResponse]{
		Results: dto.NewWorkoutRecordsResponse(records),
	}

	return c.JSON(http.StatusOK, resp)
}

// GetRecordsRanking returns ranked workouts for a given distance label
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
// @Success      200  {object}  api.PaginatedResponse[dto.DistanceRecordResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /records/ranking [get]
func (uc *userController) GetRecordsRanking(c echo.Context) error {
	user := uc.context.GetUser(c)

	workoutType := c.QueryParam("workout_type")
	label := c.QueryParam("label")

	if workoutType == "" || label == "" {
		return renderApiError(c, http.StatusBadRequest, errors.New("workout_type and label are required"))
	}

	wt := model.AsWorkoutType(workoutType)

	var pagination dto.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	pagination.SetDefaults()

	startDate, endDate, err := parseDateRange(c)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	records, totalCount, err := user.GetDistanceRecordRanking(wt, label, startDate, endDate, pagination.PerPage, pagination.GetOffset())
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.PaginatedResponse[dto.DistanceRecordResponse]{
		Results:    dto.NewDistanceRecordResponses(records),
		Page:       pagination.Page,
		PerPage:    pagination.PerPage,
		TotalPages: pagination.CalculateTotalPages(totalCount),
		TotalCount: totalCount,
	}

	return c.JSON(http.StatusOK, resp)
}

// GetClimbRecordsRanking returns ranked climb segments ordered by elevation gain
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
// @Success      200  {object}  api.PaginatedResponse[dto.ClimbRecordResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /records/climbs/ranking [get]
func (uc *userController) GetClimbRecordsRanking(c echo.Context) error {
	user := uc.context.GetUser(c)

	workoutType := c.QueryParam("workout_type")
	if workoutType == "" {
		return renderApiError(c, http.StatusBadRequest, errors.New("workout_type is required"))
	}

	wt := model.AsWorkoutType(workoutType)

	var pagination dto.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	pagination.SetDefaults()

	startDate, endDate, err := parseDateRange(c)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	records, totalCount, err := user.GetClimbRanking(wt, startDate, endDate, pagination.PerPage, pagination.GetOffset())
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.PaginatedResponse[dto.ClimbRecordResponse]{
		Results:    dto.NewClimbRecordResponses(records),
		Page:       pagination.Page,
		PerPage:    pagination.PerPage,
		TotalPages: pagination.CalculateTotalPages(totalCount),
		TotalCount: totalCount,
	}

	return c.JSON(http.StatusOK, resp)
}

// GetUserByID returns a specific user's workout records
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
func (uc *userController) GetUserByID(c echo.Context) error {
	id, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	u, err := model.GetUserByID(uc.context.GetDB(), id)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if u.IsAnonymous() {
		return renderApiError(c, http.StatusForbidden, dto.ErrNotAuthorized)
	}

	startDate, endDate, err := parseDateRange(c)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	records, err := u.GetAllRecords(startDate, endDate)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[[]dto.WorkoutRecordResponse]{
		Results: dto.NewWorkoutRecordsResponse(records),
	}

	return c.JSON(http.StatusOK, resp)
}

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
