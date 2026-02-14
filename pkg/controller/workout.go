package controller

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model/dto"
	"github.com/jovandeginste/workout-tracker/v2/pkg/container"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/labstack/echo/v4"
	"github.com/spf13/cast"
)

type WorkoutController interface {
	GetWorkouts(c echo.Context) error
	GetWorkout(c echo.Context) error
	GetWorkoutBreakdown(c echo.Context) error
	GetWorkoutRangeStats(c echo.Context) error
	GetWorkoutCalendar(c echo.Context) error
	CreateWorkout(c echo.Context) error
	GetRecentWorkouts(c echo.Context) error
	DeleteWorkout(c echo.Context) error
	UpdateWorkout(c echo.Context) error
	ToggleWorkoutLock(c echo.Context) error
	RefreshWorkout(c echo.Context) error
	CreateWorkoutShare(c echo.Context) error
	DeleteWorkoutShare(c echo.Context) error
	DownloadWorkout(c echo.Context) error
	GetPublicWorkout(c echo.Context) error
	GetPublicWorkoutBreakdown(c echo.Context) error
	GetPublicWorkoutRangeStats(c echo.Context) error
}

type workoutController struct {
	context *container.Container
}

var _ WorkoutController = (*workoutController)(nil)

func NewWorkoutController(c *container.Container) WorkoutController {
	return &workoutController{context: c}
}

func (wc *workoutController) getWorkout(c echo.Context) (*model.Workout, error) {
	id, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return nil, err
	}

	w, err := wc.context.GetUser(c).GetWorkout(wc.context.GetDB(), id)
	if err != nil {
		return nil, err
	}

	return w, nil
}

// GetWorkouts returns a paginated list of workouts for the current user
// @Summary      List workouts
// @Tags         workouts
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        page      query     int    false "Page"
// @Param        per_page  query     int    false "Per page"
// @Produce      json
// @Success      200  {object}  api.PaginatedResponse[dto.WorkoutResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /workouts [get]
func (wc *workoutController) GetWorkouts(c echo.Context) error {
	user := wc.context.GetUser(c)

	var pagination dto.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	pagination.SetDefaults()

	filters, err := model.GetWorkoutsFilters(c)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	var totalCount int64
	if err := filters.ToQuery(wc.context.GetDB().Model(&model.Workout{})).Where("user_id = ?", user.ID).Select("COUNT(workouts.id)").Count(&totalCount).Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	var workouts []*model.Workout
	db := filters.ToQuery(wc.context.GetDB().Model(&model.Workout{})).Preload("GPX").Preload("Data").
		Where("user_id = ?", user.ID).
		Order("date DESC").
		Limit(pagination.PerPage).
		Offset(pagination.GetOffset())

	if err := db.Find(&workouts).Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	results := dto.NewWorkoutsResponse(workouts)

	resp := dto.PaginatedResponse[dto.WorkoutResponse]{
		Results:    results,
		Page:       pagination.Page,
		PerPage:    pagination.PerPage,
		TotalPages: pagination.CalculateTotalPages(totalCount),
		TotalCount: totalCount,
	}

	return c.JSON(http.StatusOK, resp)
}

// GetWorkout returns a single workout for the current user
// @Summary      Get workout by ID
// @Tags         workouts
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path      int  true  "Workout ID"
// @Produce      json
// @Success      200  {object}  api.Response[dto.WorkoutDetailResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      404  {object}  api.Response[any]
// @Router       /workouts/{id} [get]
func (wc *workoutController) GetWorkout(c echo.Context) error {
	user := wc.context.GetUser(c)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	var workout model.Workout
	db := wc.context.GetDB().Preload("Data.Details").
		Preload("Data").
		Preload("GPX").
		Preload("Equipment").
		Preload("RouteSegmentMatches.RouteSegment").
		Where("user_id = ? AND id = ?", user.ID, id)

	if err := db.First(&workout).Error; err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	workout.User = user
	records, err := model.GetWorkoutIntervalRecordsWithRank(wc.context.GetDB(), user.ID, workout.Type, workout.ID)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	result := dto.NewWorkoutDetailResponse(&workout, records)

	resp := dto.Response[dto.WorkoutDetailResponse]{
		Results: result,
	}

	return c.JSON(http.StatusOK, resp)
}

// GetWorkoutBreakdown returns breakdown table data or laps for a workout
// @Summary      Get workout breakdown
// @Tags         workouts
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id     path   int     true  "Workout ID"
// @Param        unit   query  string  false "Unit"
// @Param        count  query  number  false "Count"
// @Produce      json
// @Success      200  {object}  api.Response[dto.WorkoutBreakdownResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      404  {object}  api.Response[any]
// @Router       /workouts/{id}/breakdown [get]
func (wc *workoutController) GetWorkoutBreakdown(c echo.Context) error {
	user := wc.context.GetUser(c)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	params := struct {
		Count float64 `query:"count"`
		Mode  string  `query:"mode"`
	}{
		Count: 1.0,
		Mode:  "auto",
	}

	if err := c.Bind(&params); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	if params.Count <= 0 {
		params.Count = 1.0
	}

	var workout model.Workout
	if err := wc.context.GetDB().Preload("Data").Preload("Data.Details").Preload("GPX").Where("user_id = ? AND id = ?", user.ID, id).First(&workout).Error; err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	workout.User = user

	resp := dto.Response[dto.WorkoutBreakdownResponse]{}

	preferLaps := (params.Mode == "" || params.Mode == "auto" || params.Mode == "laps") && workout.Data != nil && len(workout.Data.Laps) > 1

	if preferLaps {
		resp.Results = dto.WorkoutBreakdownResponse{
			Mode:  "laps",
			Items: dto.NewWorkoutBreakdownItemsFromLaps(workout.Data.Laps, workout.Data.Details.Points, user.PreferredUnits()),
		}

		return c.JSON(http.StatusOK, resp)
	}

	if workout.Data == nil || workout.Data.Details == nil {
		return renderApiError(c, http.StatusBadRequest, errors.New("workout has no map data"))
	}

	breakdown, err := workout.StatisticsPer(params.Count, user.PreferredUnits().Distance())
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	resp.Results = dto.WorkoutBreakdownResponse{
		Mode:  "unit",
		Items: dto.NewWorkoutBreakdownItemsFromUnit(breakdown.Items, breakdown.Unit, params.Count, user.PreferredUnits()),
	}

	return c.JSON(http.StatusOK, resp)
}

// GetWorkoutRangeStats returns aggregate statistics for a selection of map points
// @Summary      Get workout range statistics
// @Tags         workouts
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id           path   int  true  "Workout ID"
// @Param        start_index  query  int  false "Start point index (inclusive)"
// @Param        end_index    query  int  false "End point index (inclusive)"
// @Produce      json
// @Success      200  {object}  api.Response[dto.WorkoutRangeStatsResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      404  {object}  api.Response[any]
// @Router       /workouts/{id}/stats-range [get]
func (wc *workoutController) GetWorkoutRangeStats(c echo.Context) error {
	user := wc.context.GetUser(c)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	params := struct {
		StartIndex *int `query:"start_index"`
		EndIndex   *int `query:"end_index"`
	}{}

	if err := c.Bind(&params); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	var workout model.Workout
	if err := wc.context.GetDB().Preload("Data").Preload("Data.Details").Where("user_id = ? AND id = ?", user.ID, id).First(&workout).Error; err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	if workout.Data == nil || workout.Data.Details == nil || len(workout.Data.Details.Points) == 0 {
		return renderApiError(c, http.StatusBadRequest, errors.New("workout has no map data"))
	}

	points := workout.Data.Details.Points
	startIdx := 0
	endIdx := len(points) - 1

	if params.StartIndex != nil {
		startIdx = *params.StartIndex
	}

	if params.EndIndex != nil {
		endIdx = *params.EndIndex
	}

	if startIdx < 0 || endIdx >= len(points) || startIdx > endIdx {
		return renderApiError(c, http.StatusBadRequest, errors.New("invalid range"))
	}

	stats, ok := workout.Data.Details.StatsForRange(startIdx, endIdx)
	if !ok {
		return renderApiError(c, http.StatusBadRequest, errors.New("invalid range"))
	}

	resp := dto.Response[dto.WorkoutRangeStatsResponse]{
		Results: dto.NewWorkoutRangeStatsResponse(stats, startIdx, endIdx, user.PreferredUnits()),
	}

	return c.JSON(http.StatusOK, resp)
}

type calendarQueryParams struct {
	Start    *string `query:"start"`
	End      *string `query:"end"`
	TimeZone *string `query:"timeZone"`
}

// GetWorkoutCalendar returns calendar events of workouts for the current user
// @Summary      Get workout calendar events
// @Tags         workouts
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Success      200  {object}  api.Response[[]api.CalendarEventResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /workouts/calendar [get]
func (wc *workoutController) GetWorkoutCalendar(c echo.Context) error {
	user := wc.context.GetUser(c)

	var params calendarQueryParams
	if err := c.Bind(&params); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	tz := time.UTC
	if params.TimeZone != nil {
		location, err := time.LoadLocation(*params.TimeZone)
		if err == nil {
			tz = location
		}
	}

	db := wc.context.GetDB().Preload("Data").Where("user_id = ?", user.ID)

	const calTS = "2006-01-02T15:04:05"
	if params.Start != nil {
		if start, err := time.ParseInLocation(calTS, *params.Start, tz); err == nil {
			db = db.Where("workouts.date >= ?", start)
		}
	}
	if params.End != nil {
		if end, err := time.ParseInLocation(calTS, *params.End, tz); err == nil {
			db = db.Where("workouts.date <= ?", end)
		}
	}

	var workouts []*model.Workout
	if err := db.Find(&workouts).Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	events := make([]dto.CalendarEventResponse, len(workouts))
	for i, w := range workouts {
		title := w.Name
		if title == "" {
			title = string(w.Type)
		}

		if w.Data != nil {
			if w.Data.TotalDistance > 0 {
				title += " - " + formatDistance(w.Data.TotalDistance)
			}
			if w.Data.TotalDuration.Seconds() > 0 {
				title += " " + formatDuration(int64(w.Data.TotalDuration.Seconds()))
			}
		}

		events[i] = dto.CalendarEventResponse{
			Title: title,
			Start: w.GetDate().In(tz),
			End:   w.GetEnd().In(tz),
			URL:   "/workouts/" + strconv.FormatUint(w.ID, 10),
		}
	}

	resp := dto.Response[[]dto.CalendarEventResponse]{
		Results: events,
	}

	return c.JSON(http.StatusOK, resp)
}

// CreateWorkout creates a new workout (file upload or manual entry)
// @Summary      Create workout
// @Tags         workouts
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Accept       multipart/form-data
// @Accept       json
// @Produce      json
// @Success      201  {object}  api.Response[dto.WorkoutResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /workouts [post]
func (wc *workoutController) CreateWorkout(c echo.Context) error {
	user := wc.context.GetUser(c)

	if c.Request().Header.Get(echo.HeaderContentType) != "" &&
		strings.HasPrefix(c.Request().Header.Get(echo.HeaderContentType), echo.MIMEMultipartForm) {
		return wc.createWorkoutFromFile(c, user)
	}

	return wc.createWorkoutManual(c, user)
}

func (wc *workoutController) createWorkoutFromFile(c echo.Context, user *model.User) error {
	form, err := c.MultipartForm()
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	files := form.File["file"]
	if len(files) == 0 {
		return renderApiError(c, http.StatusBadRequest, errors.New("no file uploaded"))
	}

	notes := c.FormValue("notes")
	workoutType := model.WorkoutType(c.FormValue("type"))
	if workoutType == "" {
		workoutType = model.WorkoutTypeAutoDetect
	}

	createdWorkouts := []dto.WorkoutResponse{}
	errs := []string{}

	for _, file := range files {
		content, parseErr := uploadedFile(file)
		if parseErr != nil {
			errs = append(errs, parseErr.Error())
			continue
		}

		ws, addErr := user.AddWorkout(wc.context.GetDB(), workoutType, notes, file.Filename, content)
		if len(addErr) > 0 {
			for _, e := range addErr {
				errs = append(errs, e.Error())
			}
			continue
		}

		for _, w := range ws {
			createdWorkouts = append(createdWorkouts, dto.NewWorkoutResponse(w))
		}
	}

	resp := dto.Response[[]dto.WorkoutResponse]{
		Results: createdWorkouts,
	}

	if len(errs) > 0 {
		for _, err := range errs {
			resp.AddError(errors.New(err))
		}
	}

	statusCode := http.StatusCreated
	if len(createdWorkouts) == 0 && len(errs) > 0 {
		statusCode = http.StatusBadRequest
	}

	return c.JSON(statusCode, resp)
}

func (wc *workoutController) createWorkoutManual(c echo.Context, user *model.User) error {
	d := &dto.ManualWorkout{Units: user.PreferredUnits()}
	if err := c.Bind(d); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	workout := &model.Workout{}
	d.Update(workout)

	workout.User = user
	workout.UserID = user.ID
	workout.Data.Creator = "web-interface"

	equipment, err := model.GetEquipmentByIDs(wc.context.GetDB(), user.ID, d.EquipmentIDs)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	if err := workout.Save(wc.context.GetDB()); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if err := wc.context.GetDB().Model(&workout).Association("Equipment").Replace(equipment); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if err := wc.context.GetDB().Preload("Data").Preload("Equipment").First(&workout, workout.ID).Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	result := dto.NewWorkoutResponse(workout)
	resp := dto.Response[dto.WorkoutResponse]{
		Results: result,
	}

	return c.JSON(http.StatusCreated, resp)
}

// GetRecentWorkouts returns recent workouts from all users
// @Summary      List recent workouts
// @Tags         workouts
// @Produce      json
// @Param        limit   query  int false "Limit"
// @Param        offset  query  int false "Offset"
// @Success      200  {object}  api.Response[[]api.WorkoutResponse]
// @Failure      500  {object}  api.Response[any]
// @Router       /workouts/recent [get]
func (wc *workoutController) GetRecentWorkouts(c echo.Context) error {
	limit := 20
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			if parsedLimit > 0 && parsedLimit <= 100 {
				limit = parsedLimit
			}
		}
	}

	offset := 0
	if offsetStr := c.QueryParam("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil {
			if parsedOffset >= 0 {
				offset = parsedOffset
			}
		}
	}

	workouts, err := model.GetRecentWorkoutsWithOffset(wc.context.GetDB(), limit, offset)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	results := dto.NewWorkoutsResponse(workouts)

	resp := dto.Response[[]dto.WorkoutResponse]{
		Results: results,
	}

	return c.JSON(http.StatusOK, resp)
}

// DeleteWorkout deletes a workout
// @Summary      Delete workout
// @Tags         workouts
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "Workout ID"
// @Produce      json
// @Success      200  {object}  api.Response[map[string]string]
// @Failure      404  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /workouts/{id} [delete]
func (wc *workoutController) DeleteWorkout(c echo.Context) error {
	workout, err := wc.getWorkout(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	if err := workout.Delete(wc.context.GetDB()); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[map[string]string]{
		Results: map[string]string{"message": "Workout deleted successfully"},
	}

	return c.JSON(http.StatusOK, resp)
}

// UpdateWorkout updates a workout
// @Summary      Update workout
// @Tags         workouts
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "Workout ID"
// @Accept       json
// @Produce      json
// @Success      200  {object}  api.Response[dto.WorkoutResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      404  {object}  api.Response[any]
// @Router       /workouts/{id} [put]
func (wc *workoutController) UpdateWorkout(c echo.Context) error {
	user := wc.context.GetUser(c)

	workout, err := wc.getWorkout(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	d := &dto.ManualWorkout{Units: user.PreferredUnits()}
	if err := c.Bind(d); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	d.Update(workout)

	if d.EquipmentIDs != nil {
		equipment, err := model.GetEquipmentByIDs(wc.context.GetDB(), user.ID, d.EquipmentIDs)
		if err != nil {
			return renderApiError(c, http.StatusBadRequest, err)
		}
		if err := wc.context.GetDB().Model(&workout).Association("Equipment").Replace(equipment); err != nil {
			return renderApiError(c, http.StatusInternalServerError, err)
		}
	}

	if err := workout.Save(wc.context.GetDB()); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if err := wc.context.GetDB().Preload("Data").Preload("Equipment").First(&workout, workout.ID).Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	result := dto.NewWorkoutResponse(workout)
	resp := dto.Response[dto.WorkoutResponse]{
		Results: result,
	}

	return c.JSON(http.StatusOK, resp)
}

// ToggleWorkoutLock toggles the locked status of a workout
// @Summary      Toggle workout lock
// @Tags         workouts
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "Workout ID"
// @Produce      json
// @Success      200  {object}  api.Response[dto.WorkoutResponse]
// @Failure      404  {object}  api.Response[any]
// @Failure      403  {object}  api.Response[any]
// @Router       /workouts/{id}/toggle-lock [post]
func (wc *workoutController) ToggleWorkoutLock(c echo.Context) error {
	user := wc.context.GetUser(c)

	workout, err := wc.getWorkout(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	if workout.UserID != user.ID {
		return renderApiError(c, http.StatusForbidden, errors.New("not authorized"))
	}

	workout.Locked = !workout.Locked

	if err := workout.Save(wc.context.GetDB()); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	result := dto.NewWorkoutResponse(workout)
	resp := dto.Response[dto.WorkoutResponse]{
		Results: result,
	}

	return c.JSON(http.StatusOK, resp)
}

// RefreshWorkout marks a workout for refresh
// @Summary      Refresh workout
// @Tags         workouts
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "Workout ID"
// @Produce      json
// @Success      200  {object}  api.Response[map[string]string]
// @Failure      404  {object}  api.Response[any]
// @Router       /workouts/{id}/refresh [post]
func (wc *workoutController) RefreshWorkout(c echo.Context) error {
	workout, err := wc.getWorkout(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	workout.Dirty = true

	if err := workout.Save(wc.context.GetDB()); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[map[string]string]{
		Results: map[string]string{"message": "Workout will be refreshed soon"},
	}

	return c.JSON(http.StatusOK, resp)
}

// CreateWorkoutShare generates or regenerates a public share link for a workout
// @Summary      Create or regenerate share link
// @Tags         workouts
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "Workout ID"
// @Produce      json
// @Success      200  {object}  api.Response[map[string]string]
// @Failure      404  {object}  api.Response[any]
// @Router       /workouts/{id}/share [post]
func (wc *workoutController) CreateWorkoutShare(c echo.Context) error {
	workout, err := wc.getWorkout(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	u := uuid.New()
	workout.PublicUUID = &u

	if err := workout.Save(wc.context.GetDB()); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[map[string]string]{
		Results: map[string]string{
			"message":     "Public share link generated successfully",
			"public_uuid": u.String(),
			"share_url":   "/share/" + u.String(),
		},
	}

	return c.JSON(http.StatusOK, resp)
}

// DeleteWorkoutShare deletes the public share link for a workout
// @Summary      Delete workout share link
// @Tags         workouts
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "Workout ID"
// @Produce      json
// @Success      200  {object}  api.Response[map[string]string]
// @Failure      404  {object}  api.Response[any]
// @Router       /workouts/{id}/share [delete]
func (wc *workoutController) DeleteWorkoutShare(c echo.Context) error {
	workout, err := wc.getWorkout(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	workout.PublicUUID = nil

	if err := workout.Save(wc.context.GetDB()); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[map[string]string]{
		Results: map[string]string{"message": "Public share link deleted successfully"},
	}

	return c.JSON(http.StatusOK, resp)
}

// DownloadWorkout downloads the original workout file
// @Summary      Download workout file
// @Tags         workouts
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "Workout ID"
// @Produce      octet-stream
// @Success      200  {string}  string  "binary workout file"
// @Failure      404  {object}  api.Response[any]
// @Router       /workouts/{id}/download [get]
func (wc *workoutController) DownloadWorkout(c echo.Context) error {
	workout, err := wc.getWorkout(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	if !workout.HasFile() {
		return renderApiError(c, http.StatusNotFound, errors.New("workout has no file"))
	}

	basename := workout.GPX.Filename
	if basename == "" {
		basename = "workout_" + strconv.FormatUint(workout.ID, 10) + ".gpx"
	}

	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename=\""+basename+"\"")

	return c.Blob(http.StatusOK, "application/binary", workout.GPX.Content)
}

// GetPublicWorkout returns a public workout by UUID
// @Summary      Get public workout
// @Tags         workouts
// @Param        uuid  path  string  true  "Public UUID"
// @Produce      json
// @Success      200  {object}  api.Response[dto.WorkoutDetailResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      404  {object}  api.Response[any]
// @Router       /workouts/public/{uuid} [get]
func (wc *workoutController) GetPublicWorkout(c echo.Context) error {
	uuidParam := c.Param("uuid")
	u, err := uuid.Parse(uuidParam)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	workout, err := model.GetWorkoutDetailsByUUID(wc.context.GetDB(), u)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	records, err := model.GetWorkoutIntervalRecordsWithRank(wc.context.GetDB(), workout.UserID, workout.Type, workout.ID)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	result := dto.NewWorkoutDetailResponse(workout, records)

	resp := dto.Response[dto.WorkoutDetailResponse]{
		Results: result,
	}

	return c.JSON(http.StatusOK, resp)
}

// GetPublicWorkoutBreakdown returns breakdown table data or laps for a public workout
// @Summary      Get public workout breakdown
// @Tags         workouts
// @Param        uuid   path   string  true  "Public UUID"
// @Param        unit   query  string  false "Unit"
// @Param        count  query  number  false "Count"
// @Produce      json
// @Success      200  {object}  api.Response[dto.WorkoutBreakdownResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      404  {object}  api.Response[any]
// @Router       /workouts/public/{uuid}/breakdown [get]
func (wc *workoutController) GetPublicWorkoutBreakdown(c echo.Context) error {
	uuidParam := c.Param("uuid")
	u, err := uuid.Parse(uuidParam)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	params := struct {
		Count float64 `query:"count"`
		Mode  string  `query:"mode"`
	}{
		Count: 1.0,
		Mode:  "auto",
	}

	if err := c.Bind(&params); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	if params.Count <= 0 {
		params.Count = 1.0
	}

	workout, err := model.GetWorkoutDetailsByUUID(wc.context.GetDB(), u)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	owner, err := model.GetUserByID(wc.context.GetDB(), workout.UserID)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	resp := dto.Response[dto.WorkoutBreakdownResponse]{}

	preferLaps := (params.Mode == "" || params.Mode == "auto" || params.Mode == "laps") && workout.Data != nil && len(workout.Data.Laps) > 1

	if preferLaps {
		resp.Results = dto.WorkoutBreakdownResponse{
			Mode:  "laps",
			Items: dto.NewWorkoutBreakdownItemsFromLaps(workout.Data.Laps, workout.Data.Details.Points, owner.PreferredUnits()),
		}

		return c.JSON(http.StatusOK, resp)
	}

	if workout.Data == nil || workout.Data.Details == nil {
		return renderApiError(c, http.StatusBadRequest, errors.New("workout has no map data"))
	}

	breakdown, err := workout.StatisticsPer(params.Count, owner.PreferredUnits().Distance())
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	resp.Results = dto.WorkoutBreakdownResponse{
		Mode:  "unit",
		Items: dto.NewWorkoutBreakdownItemsFromUnit(breakdown.Items, breakdown.Unit, params.Count, owner.PreferredUnits()),
	}

	return c.JSON(http.StatusOK, resp)
}

// GetPublicWorkoutRangeStats returns aggregate statistics for a selection of map points in a public workout
// @Summary      Get public workout range statistics
// @Tags         workouts
// @Param        uuid         path   string  true  "Public UUID"
// @Param        start_index  query  int  false "Start point index (inclusive)"
// @Param        end_index    query  int  false "End point index (inclusive)"
// @Produce      json
// @Success      200  {object}  api.Response[dto.WorkoutRangeStatsResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      404  {object}  api.Response[any]
// @Router       /workouts/public/{uuid}/stats-range [get]
func (wc *workoutController) GetPublicWorkoutRangeStats(c echo.Context) error {
	uuidParam := c.Param("uuid")
	u, err := uuid.Parse(uuidParam)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	params := struct {
		StartIndex *int `query:"start_index"`
		EndIndex   *int `query:"end_index"`
	}{}

	if err := c.Bind(&params); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	workout, err := model.GetWorkoutDetailsByUUID(wc.context.GetDB(), u)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	owner, err := model.GetUserByID(wc.context.GetDB(), workout.UserID)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	if workout.Data == nil || workout.Data.Details == nil || len(workout.Data.Details.Points) == 0 {
		return renderApiError(c, http.StatusBadRequest, errors.New("workout has no map data"))
	}

	points := workout.Data.Details.Points
	startIdx := 0
	endIdx := len(points) - 1

	if params.StartIndex != nil {
		startIdx = *params.StartIndex
	}

	if params.EndIndex != nil {
		endIdx = *params.EndIndex
	}

	if startIdx < 0 || endIdx >= len(points) || startIdx > endIdx {
		return renderApiError(c, http.StatusBadRequest, errors.New("invalid range"))
	}

	stats, ok := workout.Data.Details.StatsForRange(startIdx, endIdx)
	if !ok {
		return renderApiError(c, http.StatusBadRequest, errors.New("invalid range"))
	}

	resp := dto.Response[dto.WorkoutRangeStatsResponse]{
		Results: dto.NewWorkoutRangeStatsResponse(stats, startIdx, endIdx, owner.PreferredUnits()),
	}

	return c.JSON(http.StatusOK, resp)
}

func uploadedFile(file *multipart.FileHeader) ([]byte, error) {
	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	content, err := io.ReadAll(src)
	if err != nil {
		return nil, err
	}

	return content, nil
}

func formatDistance(meters float64) string {
	km := meters / 1000
	if km < 10 {
		return strconv.FormatFloat(km, 'f', 2, 64) + " km"
	}
	return strconv.FormatFloat(km, 'f', 1, 64) + " km"
}

func formatDuration(seconds int64) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	if hours > 0 {
		return strconv.FormatInt(hours, 10) + "h " + strconv.FormatInt(minutes, 10) + "m"
	}
	return strconv.FormatInt(minutes, 10) + "m"
}
