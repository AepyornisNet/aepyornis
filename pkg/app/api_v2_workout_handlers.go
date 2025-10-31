package app

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jovandeginste/workout-tracker/v2/pkg/api"
	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
	"github.com/labstack/echo/v4"
)

func (a *App) registerAPIV2WorkoutRoutes(apiGroup *echo.Group, apiGroupPublic *echo.Group) {
	apiGroup.GET("/workouts", a.apiV2WorkoutsHandler).Name = "api-v2-workouts"
	apiGroup.GET("/workouts/recent", a.apiV2RecentWorkoutsHandler).Name = "api-v2-workouts-recent"
	apiGroup.GET("/workouts/calendar", a.apiV2WorkoutsCalendarHandler).Name = "api-v2-workouts-calendar"
	apiGroup.POST("/workouts", a.apiV2WorkoutCreateHandler).Name = "api-v2-workouts-create"
	apiGroup.GET("/workouts/:id", a.apiV2WorkoutHandler).Name = "api-v2-workout"
	apiGroup.PUT("/workouts/:id", a.apiV2WorkoutUpdateHandler).Name = "api-v2-workout-update"
	apiGroup.DELETE("/workouts/:id", a.apiV2WorkoutDeleteHandler).Name = "api-v2-workout-delete"
	apiGroup.POST("/workouts/:id/toggle-lock", a.apiV2WorkoutToggleLockHandler).Name = "api-v2-workout-toggle-lock"
	apiGroup.POST("/workouts/:id/refresh", a.apiV2WorkoutRefreshHandler).Name = "api-v2-workout-refresh"
	apiGroup.POST("/workouts/:id/share", a.apiV2WorkoutShareHandler).Name = "api-v2-workout-share"
	apiGroup.DELETE("/workouts/:id/share", a.apiV2WorkoutShareDeleteHandler).Name = "api-v2-workout-share-delete"
	apiGroup.GET("/workouts/:id/download", a.apiV2WorkoutDownloadHandler).Name = "api-v2-workout-download"
	apiGroupPublic.GET("/workouts/public/:uuid", a.apiV2WorkoutPublicHandler).Name = "api-v2-workout-public"
}

// apiV2WorkoutsHandler returns a paginated list of workouts for the current user
func (a *App) apiV2WorkoutsHandler(c echo.Context) error {
	user := a.getCurrentUser(c)

	// Parse pagination parameters
	var pagination api.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}
	pagination.SetDefaults()

	// Parse filters
	filters, err := database.GetWorkoutsFilters(c)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	// Get total count
	var totalCount int64
	if err := filters.ToQuery(a.db.Model(&database.Workout{})).Where("user_id = ?", user.ID).Select("COUNT(workouts.id)").Count(&totalCount).Error; err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	// Get paginated workouts
	var workouts []*database.Workout
	db := filters.ToQuery(a.db.Model(&database.Workout{})).Preload("Data").
		Where("user_id = ?", user.ID).
		Order("date DESC").
		Limit(pagination.PerPage).
		Offset(pagination.GetOffset())

	if err := db.Find(&workouts).Error; err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	// Convert to API response
	results := api.NewWorkoutsResponse(workouts)

	resp := api.PaginatedResponse[api.WorkoutResponse]{
		Results:    results,
		Page:       pagination.Page,
		PerPage:    pagination.PerPage,
		TotalPages: pagination.CalculateTotalPages(totalCount),
		TotalCount: totalCount,
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2WorkoutHandler returns a single workout for the current user
func (a *App) apiV2WorkoutHandler(c echo.Context) error {
	user := a.getCurrentUser(c)

	// Parse workout ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	// Get workout with all details
	var workout database.Workout
	db := a.db.Preload("Data.Details").
		Preload("Data").
		Preload("Equipment").
		Preload("RouteSegmentMatches.RouteSegment").
		Where("user_id = ? AND id = ?", user.ID, id)

	if err := db.First(&workout).Error; err != nil {
		return a.renderAPIV2Error(c, http.StatusNotFound, err)
	}

	// Convert to API response
	result := api.NewWorkoutDetailResponse(&workout)

	resp := api.Response[api.WorkoutDetailResponse]{
		Results: result,
	}

	return c.JSON(http.StatusOK, resp)
}

// CalendarQueryParams represents query parameters for calendar endpoint
type CalendarQueryParams struct {
	Start    *string `query:"start"`
	End      *string `query:"end"`
	TimeZone *string `query:"timeZone"`
}

// apiV2WorkoutsCalendarHandler returns calendar events of workouts for the current user
func (a *App) apiV2WorkoutsCalendarHandler(c echo.Context) error {
	user := a.getCurrentUser(c)

	// Parse query parameters
	var params CalendarQueryParams
	if err := c.Bind(&params); err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	// Parse timezone
	tz := time.UTC
	if params.TimeZone != nil {
		location, err := time.LoadLocation(*params.TimeZone)
		if err == nil {
			tz = location
		}
	}

	// Build query
	db := a.db.Preload("Data").Where("user_id = ?", user.ID)

	// Apply date filters
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

	// Get workouts
	var workouts []*database.Workout
	if err := db.Find(&workouts).Error; err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	// Convert to calendar events
	events := make([]api.CalendarEventResponse, len(workouts))
	for i, w := range workouts {
		// Build title from workout name and type
		title := w.Name
		if title == "" {
			title = string(w.Type)
		}

		// Add distance/duration info if available
		if w.Data != nil {
			if w.Data.TotalDistance > 0 {
				title += " - " + formatDistance(w.Data.TotalDistance)
			}
			if w.Data.TotalDuration.Seconds() > 0 {
				title += " " + formatDuration(int64(w.Data.TotalDuration.Seconds()))
			}
		}

		events[i] = api.CalendarEventResponse{
			Title: title,
			Start: w.GetDate().In(tz),
			End:   w.GetEnd().In(tz),
			URL:   "/workouts/" + strconv.FormatUint(w.ID, 10),
		}
	}

	resp := api.Response[[]api.CalendarEventResponse]{
		Results: events,
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2WorkoutCreateHandler creates a new workout (file upload or manual entry)
func (a *App) apiV2WorkoutCreateHandler(c echo.Context) error {
	user := a.getCurrentUser(c)

	// Check if this is a multipart form (file upload)
	if c.Request().Header.Get(echo.HeaderContentType) != "" &&
		strings.HasPrefix(c.Request().Header.Get(echo.HeaderContentType), echo.MIMEMultipartForm) {
		return a.apiV2WorkoutCreateFromFileHandler(c, user)
	}

	// Manual workout creation
	return a.apiV2WorkoutCreateManualHandler(c, user)
}

// apiV2WorkoutCreateFromFileHandler handles file upload workout creation
func (a *App) apiV2WorkoutCreateFromFileHandler(c echo.Context, user *database.User) error {
	form, err := c.MultipartForm()
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	files := form.File["file"]
	if len(files) == 0 {
		return a.renderAPIV2Error(c, http.StatusBadRequest, errors.New("no file uploaded"))
	}

	notes := c.FormValue("notes")
	workoutType := database.WorkoutType(c.FormValue("type"))
	if workoutType == "" {
		workoutType = database.WorkoutTypeAutoDetect
	}

	createdWorkouts := []api.WorkoutResponse{}
	errs := []string{}

	for _, file := range files {
		content, parseErr := uploadedFile(file)
		if parseErr != nil {
			errs = append(errs, parseErr.Error())
			continue
		}

		ws, addErr := user.AddWorkout(a.db, workoutType, notes, file.Filename, content)
		if len(addErr) > 0 {
			for _, e := range addErr {
				errs = append(errs, e.Error())
			}
			continue
		}

		for _, w := range ws {
			createdWorkouts = append(createdWorkouts, api.NewWorkoutResponse(w))
		}
	}

	resp := api.Response[[]api.WorkoutResponse]{
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

// apiV2WorkoutCreateManualHandler handles manual workout creation
func (a *App) apiV2WorkoutCreateManualHandler(c echo.Context, user *database.User) error {
	d := &ManualWorkout{units: user.PreferredUnits()}
	if err := c.Bind(d); err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	workout := &database.Workout{}
	d.Update(workout)

	workout.User = user
	workout.UserID = user.ID
	workout.Data.Creator = "web-interface"

	// Handle equipment IDs
	equipment, err := database.GetEquipmentByIDs(a.db, user.ID, d.EquipmentIDs)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	if err := workout.Save(a.db); err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	if err := a.db.Model(&workout).Association("Equipment").Replace(equipment); err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	// Reload workout with associations
	if err := a.db.Preload("Data").Preload("Equipment").First(&workout, workout.ID).Error; err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	result := api.NewWorkoutResponse(workout)
	resp := api.Response[api.WorkoutResponse]{
		Results: result,
	}

	return c.JSON(http.StatusCreated, resp)
}

// Helper functions for formatting
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

// apiV2RecentWorkoutsHandler returns recent workouts from all users
func (a *App) apiV2RecentWorkoutsHandler(c echo.Context) error {
	// Parse limit parameter (default to 20, max 100)
	limit := 20
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			if parsedLimit > 0 && parsedLimit <= 100 {
				limit = parsedLimit
			}
		}
	}

	// Parse offset parameter (default to 0)
	offset := 0
	if offsetStr := c.QueryParam("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil {
			if parsedOffset >= 0 {
				offset = parsedOffset
			}
		}
	}

	// Get recent workouts from all users
	workouts, err := database.GetRecentWorkoutsWithOffset(a.db, limit, offset)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	// Convert to API response
	results := api.NewWorkoutsResponse(workouts)

	resp := api.Response[[]api.WorkoutResponse]{
		Results: results,
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2WorkoutDeleteHandler deletes a workout
func (a *App) apiV2WorkoutDeleteHandler(c echo.Context) error {
	// Get workout
	workout, err := a.getWorkout(c)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusNotFound, err)
	}

	// Delete workout
	if err := workout.Delete(a.db); err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	resp := api.Response[map[string]string]{
		Results: map[string]string{"message": "Workout deleted successfully"},
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2WorkoutUpdateHandler updates a workout
func (a *App) apiV2WorkoutUpdateHandler(c echo.Context) error {
	user := a.getCurrentUser(c)

	// Parse workout ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	// Get workout
	var workout database.Workout
	if err := a.db.Preload("Data").Where("user_id = ? AND id = ?", user.ID, id).First(&workout).Error; err != nil {
		return a.renderAPIV2Error(c, http.StatusNotFound, err)
	}

	// Bind update data
	d := &ManualWorkout{units: user.PreferredUnits()}
	if err := c.Bind(d); err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	// Update workout
	d.Update(&workout)

	// Handle equipment IDs
	if d.EquipmentIDs != nil {
		equipment, err := database.GetEquipmentByIDs(a.db, user.ID, d.EquipmentIDs)
		if err != nil {
			return a.renderAPIV2Error(c, http.StatusBadRequest, err)
		}
		if err := a.db.Model(&workout).Association("Equipment").Replace(equipment); err != nil {
			return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
		}
	}

	// Save workout
	if err := workout.Save(a.db); err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	// Reload workout with associations
	if err := a.db.Preload("Data").Preload("Equipment").First(&workout, workout.ID).Error; err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	result := api.NewWorkoutResponse(&workout)
	resp := api.Response[api.WorkoutResponse]{
		Results: result,
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2WorkoutToggleLockHandler toggles the locked status of a workout
func (a *App) apiV2WorkoutToggleLockHandler(c echo.Context) error {
	user := a.getCurrentUser(c)

	// Get workout with details (including GPX for HasFile check)
	workout, err := a.getWorkout(c)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusNotFound, err)
	}

	// Verify user owns this workout
	if workout.UserID != user.ID {
		return a.renderAPIV2Error(c, http.StatusForbidden, errors.New("not authorized"))
	}

	// Toggle locked status
	workout.Locked = !workout.Locked

	// Save workout
	if err := workout.Save(a.db); err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	result := api.NewWorkoutResponse(workout)
	resp := api.Response[api.WorkoutResponse]{
		Results: result,
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2WorkoutRefreshHandler marks a workout for refresh
func (a *App) apiV2WorkoutRefreshHandler(c echo.Context) error {
	// Get workout
	workout, err := a.getWorkout(c)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusNotFound, err)
	}

	// Mark as dirty for refresh
	workout.Dirty = true

	// Save workout
	if err := workout.Save(a.db); err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	resp := api.Response[map[string]string]{
		Results: map[string]string{"message": "Workout will be refreshed soon"},
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2WorkoutShareHandler generates or regenerates a public share link for a workout
func (a *App) apiV2WorkoutShareHandler(c echo.Context) error {
	// Get workout
	workout, err := a.getWorkout(c)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusNotFound, err)
	}

	// Generate new UUID
	u := uuid.New()
	workout.PublicUUID = &u

	// Save workout
	if err := workout.Save(a.db); err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	// Return share link
	resp := api.Response[map[string]string]{
		Results: map[string]string{
			"message":     "Public share link generated successfully",
			"public_uuid": u.String(),
			"share_url":   "/share/" + u.String(),
		},
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2WorkoutShareDeleteHandler deletes the public share link for a workout
func (a *App) apiV2WorkoutShareDeleteHandler(c echo.Context) error {
	// Get workout
	workout, err := a.getWorkout(c)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusNotFound, err)
	}

	// Remove share link
	workout.PublicUUID = nil

	// Save workout
	if err := workout.Save(a.db); err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	resp := api.Response[map[string]string]{
		Results: map[string]string{"message": "Public share link deleted successfully"},
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2WorkoutDownloadHandler downloads the original workout file
func (a *App) apiV2WorkoutDownloadHandler(c echo.Context) error {
	// Get workout with GPX preloaded
	workout, err := a.getWorkout(c)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusNotFound, err)
	}

	// Check if workout has a file
	if !workout.HasFile() {
		return a.renderAPIV2Error(c, http.StatusNotFound, errors.New("workout has no file"))
	}

	// Get filename
	basename := workout.GPX.Filename
	if basename == "" {
		basename = "workout_" + strconv.FormatUint(workout.ID, 10) + ".gpx"
	}

	// Set headers for download
	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename=\""+basename+"\"")

	return c.Blob(http.StatusOK, "application/binary", workout.GPX.Content)
}

// apiV2WorkoutPublicHandler returns a public workout by UUID
func (a *App) apiV2WorkoutPublicHandler(c echo.Context) error {
	// Parse UUID
	uuidParam := c.Param("uuid")
	u, err := uuid.Parse(uuidParam)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	// Get workout by UUID
	workout, err := database.GetWorkoutDetailsByUUID(a.db, u)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusNotFound, err)
	}

	// Convert to API response
	result := api.NewWorkoutDetailResponse(workout)

	resp := api.Response[api.WorkoutDetailResponse]{
		Results: result,
	}

	return c.JSON(http.StatusOK, resp)
}
