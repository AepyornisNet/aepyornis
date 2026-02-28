package controller

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
	"github.com/google/uuid"
	ap "github.com/jovandeginste/workout-tracker/v2/pkg/activitypub"
	"github.com/jovandeginste/workout-tracker/v2/pkg/container"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model/dto"
	"github.com/jovandeginste/workout-tracker/v2/pkg/worker"
	"github.com/labstack/echo/v4"
	"github.com/spf13/cast"
	"gorm.io/gorm"
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
	DownloadWorkout(c echo.Context) error
}

type workoutController struct {
	context *container.Container
}

var _ WorkoutController = (*workoutController)(nil)

func NewWorkoutController(c *container.Container) WorkoutController {
	return &workoutController{context: c}
}

func workoutIDs(ws []*model.Workout) []uint64 {
	ids := make([]uint64, 0, len(ws))
	for _, w := range ws {
		if w == nil {
			continue
		}

		ids = append(ids, w.ID)
	}

	return ids
}

func applyPublishedFlags(results []dto.WorkoutResponse, published map[uint64]bool) {
	for i := range results {
		results[i].ActivityPubPublished = published[results[i].ID]
	}
}

func (wc *workoutController) getOwnedWorkout(c echo.Context) (*model.Workout, error) {
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

func (wc *workoutController) canReadWorkout(c echo.Context, requester *model.User, workout *model.Workout) (bool, error) {
	if requester == nil || workout == nil {
		return false, nil
	}

	if requester.ID == workout.UserID {
		return true, nil
	}

	switch workout.Visibility {
	case model.WorkoutVisibilityPublic:
		return true, nil
	case model.WorkoutVisibilityFollowers:
		requesterActorIRI := ap.LocalActorURL(ap.LocalActorURLConfig{
			Host:           wc.context.GetConfig().Host,
			WebRoot:        wc.context.GetConfig().WebRoot,
			FallbackHost:   c.Request().Host,
			FallbackScheme: c.Scheme(),
		}, requester.Username)

		if requesterActorIRI == "" {
			return false, nil
		}

		var count int64
		if err := wc.context.GetDB().
			Model(&model.Follower{}).
			Where("user_id = ? AND actor_iri = ? AND approved = ?", workout.UserID, requesterActorIRI, true).
			Count(&count).Error; err != nil {
			return false, err
		}

		return count > 0, nil
	default:
		return false, nil
	}
}

func (wc *workoutController) getReadableWorkout(c echo.Context, withDetails bool) (*model.Workout, error) {
	id, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return nil, err
	}

	db := model.PreloadWorkoutDetails(wc.context.GetDB()).
		Preload("GPX").
		Preload("Equipment").
		Preload("User").
		Preload("User.Profile")

	if withDetails {
		db = db.Preload("RouteSegmentMatches.RouteSegment")
	}

	var workout model.Workout
	if err := db.First(&workout, id).Error; err != nil {
		return nil, err
	}

	allowed, err := wc.canReadWorkout(c, wc.context.GetUser(c), &workout)
	if err != nil {
		return nil, err
	}

	if !allowed {
		return nil, gorm.ErrRecordNotFound
	}

	return &workout, nil
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
// @Success      200  {object}  dto.PaginatedResponse[dto.WorkoutResponse]
// @Failure      400  {object}  dto.Response[any]
// @Failure      500  {object}  dto.Response[any]
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
	db := model.PreloadWorkoutData(filters.ToQuery(wc.context.GetDB().Model(&model.Workout{}))).Preload("GPX").
		Where("user_id = ?", user.ID).
		Order("date DESC").
		Limit(pagination.PerPage).
		Offset(pagination.GetOffset())

	if err := db.Find(&workouts).Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	results := dto.NewWorkoutsResponse(workouts)
	published, err := wc.context.APOutboxRepo().PublishedMap(user.ID, workoutIDs(workouts))
	if err == nil {
		applyPublishedFlags(results, published)
	}

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
// @Success      200  {object}  dto.Response[dto.WorkoutDetailResponse]
// @Failure      400  {object}  dto.Response[any]
// @Failure      404  {object}  dto.Response[any]
// @Router       /workouts/{id} [get]
func (wc *workoutController) GetWorkout(c echo.Context) error {
	workout, err := wc.getReadableWorkout(c, true)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	records, err := model.GetWorkoutIntervalRecordsWithRank(wc.context.GetDB(), workout.UserID, workout.Type, workout.ID)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	result := dto.NewWorkoutDetailResponse(workout, records)
	published, err := wc.context.APOutboxRepo().PublishedMap(workout.UserID, []uint64{workout.ID})
	if err == nil {
		result.ActivityPubPublished = published[workout.ID]
	}

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
// @Success      200  {object}  dto.Response[dto.WorkoutBreakdownResponse]
// @Failure      400  {object}  dto.Response[any]
// @Failure      404  {object}  dto.Response[any]
// @Router       /workouts/{id}/breakdown [get]
func (wc *workoutController) GetWorkoutBreakdown(c echo.Context) error {
	requester := wc.context.GetUser(c)

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

	workout, err := wc.getReadableWorkout(c, false)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	resp := dto.Response[dto.WorkoutBreakdownResponse]{}

	preferLaps := (params.Mode == "" || params.Mode == "auto" || params.Mode == "laps") && workout.Data != nil && len(workout.Data.Laps) > 1

	if preferLaps {
		resp.Results = dto.WorkoutBreakdownResponse{
			Mode:  "laps",
			Items: dto.NewWorkoutBreakdownItemsFromLaps(workout.Data.Laps, workout.Data.Details.Points, requester.PreferredUnits()),
		}

		return c.JSON(http.StatusOK, resp)
	}

	if workout.Data == nil || workout.Data.Details == nil {
		return renderApiError(c, http.StatusBadRequest, errors.New("workout has no map data"))
	}

	breakdown, err := workout.StatisticsPer(params.Count, requester.PreferredUnits().Distance())
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	resp.Results = dto.WorkoutBreakdownResponse{
		Mode:  "unit",
		Items: dto.NewWorkoutBreakdownItemsFromUnit(breakdown.Items, breakdown.Unit, params.Count, requester.PreferredUnits()),
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
// @Success      200  {object}  dto.Response[dto.WorkoutRangeStatsResponse]
// @Failure      400  {object}  dto.Response[any]
// @Failure      404  {object}  dto.Response[any]
// @Router       /workouts/{id}/stats-range [get]
func (wc *workoutController) GetWorkoutRangeStats(c echo.Context) error {
	requester := wc.context.GetUser(c)

	params := struct {
		StartIndex *int `query:"start_index"`
		EndIndex   *int `query:"end_index"`
	}{}

	if err := c.Bind(&params); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	workout, err := wc.getReadableWorkout(c, false)
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
		Results: dto.NewWorkoutRangeStatsResponse(stats, startIdx, endIdx, requester.PreferredUnits()),
	}

	return c.JSON(http.StatusOK, resp)
}

// GetWorkoutCalendar returns calendar events of workouts for the current user
// @Summary      Get workout calendar events
// @Tags         workouts
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Success      200  {object}  dto.Response[[]dto.CalendarEventResponse]
// @Failure      400  {object}  dto.Response[any]
// @Failure      500  {object}  dto.Response[any]
// @Router       /workouts/calendar [get]
func (wc *workoutController) GetWorkoutCalendar(c echo.Context) error {
	targetUser, viewer, viewerActorIRI, err := wc.resolveTargetUserFromHandle(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	var params dto.CalendarQueryParams
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

	db := model.ScopeVisibleWorkouts(
		model.PreloadWorkoutData(wc.context.GetDB()),
		targetUser.ID,
		viewer.ID,
		viewerActorIRI,
	)

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

func (wc *workoutController) resolveTargetUserFromHandle(c echo.Context) (*model.User, *model.User, string, error) {
	viewer := wc.context.GetUser(c)
	handle := strings.TrimSpace(c.QueryParam("handle"))
	if handle == "" {
		return viewer, viewer, wc.localActorIRI(c, viewer), nil
	}

	normalizedUsername, err := wc.parseLocalHandle(c, handle)
	if err != nil {
		return nil, nil, "", err
	}

	targetUser, err := model.GetUser(wc.context.GetDB(), normalizedUsername)
	if err != nil {
		return nil, nil, "", err
	}

	if viewer.ID != targetUser.ID && !targetUser.ActivityPubEnabled() {
		return nil, nil, "", gorm.ErrRecordNotFound
	}

	return targetUser, viewer, wc.localActorIRI(c, viewer), nil
}

func (wc *workoutController) parseLocalHandle(c echo.Context, handle string) (string, error) {
	h := strings.TrimSpace(handle)
	h = strings.TrimPrefix(h, "@")

	if parsedURL, err := url.Parse(h); err == nil && parsedURL.Host != "" {
		segments := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
		if len(segments) == 3 && segments[0] == "ap" && segments[1] == "users" && segments[2] != "" {
			if wc.isLocalHost(c, parsedURL.Host) {
				return segments[2], nil
			}
			return "", gorm.ErrRecordNotFound
		}
	}

	if strings.Contains(h, "@") {
		parts := strings.SplitN(h, "@", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", gorm.ErrRecordNotFound
		}

		if !wc.isLocalHost(c, parts[1]) {
			return "", gorm.ErrRecordNotFound
		}

		return parts[0], nil
	}

	if h == "" {
		return "", gorm.ErrRecordNotFound
	}

	return h, nil
}

func (wc *workoutController) isLocalHost(c echo.Context, host string) bool {
	configuredHost := wc.context.GetConfig().Host
	if configuredHost == "" {
		configuredHost = c.Request().Host
	}

	return strings.EqualFold(strings.TrimSpace(host), strings.TrimSpace(configuredHost))
}

func (wc *workoutController) localActorIRI(c echo.Context, user *model.User) string {
	if user == nil {
		return ""
	}

	return ap.LocalActorURL(ap.LocalActorURLConfig{
		Host:           wc.context.GetConfig().Host,
		WebRoot:        wc.context.GetConfig().WebRoot,
		FallbackHost:   c.Request().Host,
		FallbackScheme: c.Scheme(),
	}, user.Username)
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
// @Success      201  {object}  dto.Response[dto.WorkoutResponse]
// @Failure      400  {object}  dto.Response[any]
// @Failure      500  {object}  dto.Response[any]
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
	errList := []error{}

	for _, file := range files {
		content, parseErr := uploadedFile(file)
		if parseErr != nil {
			errList = append(errList, parseErr)
			continue
		}

		ws, addErr := user.AddWorkout(wc.context.GetDB(), workoutType, notes, file.Filename, content)
		if len(addErr) > 0 {
			for _, e := range addErr {
				errList = append(errList, e)
			}
			continue
		}

		for _, w := range ws {
			createdWorkouts = append(createdWorkouts, dto.NewWorkoutResponse(w))

			wc.syncWorkoutActivityPub(c, user, w, nil)

			if err := worker.EnqueueWorkoutUpdate(c.Request().Context(), wc.context, w.ID); err != nil {
				wc.context.Logger().Error("Failed to enqueue workout update", "workout_id", w.ID, "error", err)
			}
		}
	}

	resp := dto.Response[[]dto.WorkoutResponse]{
		Results: createdWorkouts,
	}

	if len(errList) > 0 {
		resp.AddError(errList...)

		for _, err := range errList {
			if code := apiErrorCode(err); code != "" {
				resp.ErrorCodes = append(resp.ErrorCodes, code)
			}
		}
	}

	statusCode := http.StatusCreated
	if len(createdWorkouts) == 0 && len(errList) > 0 {
		statusCode = http.StatusBadRequest

		allDuplicates := true
		for _, err := range errList {
			if !errors.Is(err, model.ErrWorkoutAlreadyExists) {
				allDuplicates = false
				break
			}
		}

		if allDuplicates {
			statusCode = http.StatusConflict
		}
	}

	return c.JSON(statusCode, resp)
}

func (wc *workoutController) createWorkoutManual(c echo.Context, user *model.User) error {
	d := &dto.ManualWorkout{Units: user.PreferredUnits()}
	if err := c.Bind(d); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	workout := &model.Workout{}
	if err := d.Update(workout); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	if d.Visibility == nil {
		workout.Visibility = user.Profile.EffectiveDefaultWorkoutVisibility()
	}

	workout.User = user
	workout.UserID = user.ID
	workout.Data.Creator = "web-interface"

	equipment, err := wc.context.EquipmentRepo().GetByUserIDs(user.ID, d.EquipmentIDs)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	if err := workout.Save(wc.context.GetDB()); err != nil {
		if errors.Is(err, model.ErrWorkoutAlreadyExists) {
			return renderApiError(c, http.StatusConflict, err)
		}

		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if err := wc.context.GetDB().Model(&workout).Association("Equipment").Replace(equipment); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if err := model.PreloadWorkoutDetails(wc.context.GetDB()).Preload("Equipment").First(&workout, workout.ID).Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	wc.syncWorkoutActivityPub(c, user, workout, nil)

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
// @Success      200  {object}  dto.Response[[]dto.WorkoutResponse]
// @Failure      500  {object}  dto.Response[any]
// @Router       /workouts/recent [get]
func (wc *workoutController) GetRecentWorkouts(c echo.Context) error {
	requester := wc.context.GetUser(c)

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

	requesterActorIRI := ap.LocalActorURL(ap.LocalActorURLConfig{
		Host:           wc.context.GetConfig().Host,
		WebRoot:        wc.context.GetConfig().WebRoot,
		FallbackHost:   c.Request().Host,
		FallbackScheme: c.Scheme(),
	}, requester.Username)

	var workouts []*model.Workout
	err := wc.context.GetDB().
		Scopes(model.PreloadWorkoutData).
		Preload("User").
		Where(
			"workouts.user_id = ? OR workouts.visibility = ? OR (workouts.visibility = ? AND EXISTS (SELECT 1 FROM followers f WHERE f.user_id = workouts.user_id AND f.actor_iri = ? AND f.approved = ?))",
			requester.ID,
			model.WorkoutVisibilityPublic,
			model.WorkoutVisibilityFollowers,
			requesterActorIRI,
			true,
		).
		Order("date DESC").
		Limit(limit).
		Offset(offset).
		Find(&workouts).Error
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
// @Success      200  {object}  dto.Response[map[string]string]
// @Failure      404  {object}  dto.Response[any]
// @Failure      500  {object}  dto.Response[any]
// @Router       /workouts/{id} [delete]
func (wc *workoutController) DeleteWorkout(c echo.Context) error {
	user := wc.context.GetUser(c)

	workout, err := wc.getOwnedWorkout(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	if err := wc.context.APOutboxRepo().DeleteEntryForWorkout(user.ID, workout.ID); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return renderApiError(c, http.StatusInternalServerError, err)
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
// @Success      200  {object}  dto.Response[dto.WorkoutResponse]
// @Failure      400  {object}  dto.Response[any]
// @Failure      404  {object}  dto.Response[any]
// @Router       /workouts/{id} [put]
func (wc *workoutController) UpdateWorkout(c echo.Context) error {
	user := wc.context.GetUser(c)

	workout, err := wc.getOwnedWorkout(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	d := &dto.ManualWorkout{Units: user.PreferredUnits()}
	if err := c.Bind(d); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	previousVisibility := workout.Visibility

	if err := d.Update(workout); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	if d.EquipmentIDs != nil {
		equipment, err := wc.context.EquipmentRepo().GetByUserIDs(user.ID, d.EquipmentIDs)
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

	if err := model.PreloadWorkoutDetails(wc.context.GetDB()).Preload("Equipment").First(&workout, workout.ID).Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	wc.syncWorkoutActivityPub(c, user, workout, &previousVisibility)

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
// @Success      200  {object}  dto.Response[dto.WorkoutResponse]
// @Failure      404  {object}  dto.Response[any]
// @Failure      403  {object}  dto.Response[any]
// @Router       /workouts/{id}/toggle-lock [post]
func (wc *workoutController) ToggleWorkoutLock(c echo.Context) error {
	user := wc.context.GetUser(c)

	workout, err := wc.getOwnedWorkout(c)
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
// @Success      200  {object}  dto.Response[map[string]string]
// @Failure      404  {object}  dto.Response[any]
// @Router       /workouts/{id}/refresh [post]
func (wc *workoutController) RefreshWorkout(c echo.Context) error {
	workout, err := wc.getOwnedWorkout(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	workout.Dirty = true

	if err := workout.Save(wc.context.GetDB()); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if err := worker.EnqueueWorkoutUpdate(c.Request().Context(), wc.context, workout.ID); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[map[string]string]{
		Results: map[string]string{"message": "Workout will be refreshed soon"},
	}

	return c.JSON(http.StatusOK, resp)
}

func (wc *workoutController) publishWorkoutToActivityPub(c echo.Context, user *model.User, workout *model.Workout) error {
	fitContent, err := ap.GenerateWorkoutFIT(workout)
	if err != nil {
		return err
	}

	entryUUID := uuid.New()
	actorURL := ap.LocalActorURL(ap.LocalActorURLConfig{
		Host:           wc.context.GetConfig().Host,
		WebRoot:        wc.context.GetConfig().WebRoot,
		FallbackHost:   c.Request().Host,
		FallbackScheme: c.Scheme(),
	}, user.Username)

	entryURL := fmt.Sprintf("%s/outbox/%s", actorURL, entryUUID.String())
	objectURL := entryURL + "#object"
	fitURL := entryURL + "/fit"
	routeImageURL := entryURL + "/route-image"
	publishedAt := time.Now().UTC()
	noteContent := ap.WorkoutNoteContent(workout)

	attachments := vocab.ItemCollection{}
	routeImageContent, routeImageErr := ap.GenerateWorkoutRouteImage(workout)
	if routeImageErr == nil && len(routeImageContent) > 0 {
		attachments = append(attachments, &vocab.Object{
			Type:      vocab.ImageType,
			Name:      vocab.DefaultNaturalLanguage(ap.WorkoutRouteImageFilename(workout)),
			MediaType: vocab.MimeType(ap.RouteImageMIMEType),
			URL:       vocab.IRI(routeImageURL),
		})
	} else if routeImageErr != nil && !errors.Is(routeImageErr, ap.ErrWorkoutMissingCoordinates) {
		wc.context.Logger().Warn("Failed to generate workout route image", "workout_id", workout.ID, "error", routeImageErr)
	}

	note := ap.NewWorkoutNote()
	note.ID = vocab.ID(objectURL)
	note.AttributedTo = vocab.IRI(actorURL)
	note.Published = publishedAt
	note.Content = vocab.DefaultNaturalLanguage(noteContent)
	note.Attachment = attachments
	// TODO: Workout location might not be set at this point
	note.PopulateFromWorkout(workout, vocab.IRI(fitURL))

	to := vocab.ItemCollection{vocab.IRI(actorURL + "/followers")}
	cc := vocab.ItemCollection{}
	if workout.Visibility == model.WorkoutVisibilityPublic {
		to = vocab.ItemCollection{vocab.IRI("https://www.w3.org/ns/activitystreams#Public")}
		cc = vocab.ItemCollection{vocab.IRI(actorURL + "/followers")}
	}

	activity := vocab.Activity{
		ID:        vocab.ID(entryURL),
		Type:      vocab.CreateType,
		Actor:     vocab.IRI(actorURL),
		Published: publishedAt,
		To:        to,
		CC:        cc,
		Object:    note,
	}

	activityJSON, err := jsonld.WithContext(
		ap.WorkoutJSONLDContext(),
	).Marshal(activity)
	if err != nil {
		return err
	}

	noteJSON, err := jsonld.WithContext(
		ap.WorkoutJSONLDContext(),
	).Marshal(note)
	if err != nil {
		return err
	}

	outboxWorkout := &model.APOutboxWorkout{
		UserID:         user.ID,
		WorkoutID:      workout.ID,
		FitFilename:    ap.WorkoutFITFilename(workout),
		FitContent:     fitContent,
		FitContentType: ap.FitMIMEType,
	}

	if len(routeImageContent) > 0 {
		outboxWorkout.RouteImageFilename = ap.WorkoutRouteImageFilename(workout)
		outboxWorkout.RouteImageContent = routeImageContent
		outboxWorkout.RouteImageContentType = ap.RouteImageMIMEType
	}

	if err := wc.context.APOutboxRepo().CreateWorkout(outboxWorkout); err != nil {
		return err
	}

	entry := &model.APOutboxEntry{
		PublicUUID:        entryUUID,
		UserID:            user.ID,
		APOutboxWorkoutID: &outboxWorkout.ID,
		Kind:              model.APOutboxWorkoutKind,
		ActivityID:        entryURL,
		ObjectID:          objectURL,
		Activity:          activityJSON,
		Payload:           noteJSON,
		NoteText:          noteContent,
		PublishedAt:       publishedAt,
	}

	if err := wc.context.APOutboxRepo().CreateEntry(entry); err != nil {
		return err
	}

	if err := worker.EnqueueAPDeliveriesForEntry(c.Request().Context(), wc.context, entry.ID); err != nil {
		wc.context.Logger().Error("Failed to enqueue ActivityPub deliveries", "entry_id", entry.ID, "error", err)
	}

	return nil
}

func (wc *workoutController) updateWorkoutActivityPubAudience(c echo.Context, user *model.User, entry *model.APOutboxEntry, workout *model.Workout) error {
	if entry == nil {
		return errors.New("outbox entry is nil")
	}

	actorURL := ap.LocalActorURL(ap.LocalActorURLConfig{
		Host:           wc.context.GetConfig().Host,
		WebRoot:        wc.context.GetConfig().WebRoot,
		FallbackHost:   c.Request().Host,
		FallbackScheme: c.Scheme(),
	}, user.Username)

	activity := vocab.Activity{}
	if err := jsonld.Unmarshal(entry.Activity, &activity); err != nil {
		return err
	}

	note := ap.NewWorkoutNote()
	if len(entry.Payload) > 0 {
		if err := jsonld.Unmarshal(entry.Payload, note); err != nil {
			return err
		}
	}

	activity.To = vocab.ItemCollection{vocab.IRI(actorURL + "/followers")}
	activity.CC = vocab.ItemCollection{}
	activity.Object = note
	if workout.Visibility == model.WorkoutVisibilityPublic {
		activity.To = vocab.ItemCollection{vocab.IRI("https://www.w3.org/ns/activitystreams#Public")}
		activity.CC = vocab.ItemCollection{vocab.IRI(actorURL + "/followers")}
	}

	activityJSON, err := jsonld.WithContext(
		ap.WorkoutJSONLDContext(),
	).Marshal(activity)
	if err != nil {
		return err
	}

	return wc.context.GetDB().Model(&model.APOutboxEntry{}).
		Where("id = ?", entry.ID).
		Update("activity", activityJSON).Error
}

func (wc *workoutController) syncWorkoutActivityPub(c echo.Context, user *model.User, workout *model.Workout, previousVisibility *model.WorkoutVisibility) {
	if user == nil || workout == nil {
		return
	}

	if previousVisibility != nil && *previousVisibility == workout.Visibility {
		return
	}

	entry, err := wc.context.APOutboxRepo().GetEntryForWorkout(user.ID, workout.ID)
	hasOutboxEntry := err == nil
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		wc.context.Logger().Warn("Failed to check ActivityPub outbox entry", "workout_id", workout.ID, "error", err)
		return
	}

	shouldPublish := user.ActivityPubEnabled() &&
		(workout.Visibility == model.WorkoutVisibilityPublic || workout.Visibility == model.WorkoutVisibilityFollowers)

	if !shouldPublish {
		if hasOutboxEntry {
			if err := wc.context.APOutboxRepo().DeleteEntryForWorkout(user.ID, workout.ID); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				wc.context.Logger().Warn("Failed to remove ActivityPub outbox entry", "workout_id", workout.ID, "error", err)
			}
		}

		return
	}

	if hasOutboxEntry {
		if err := wc.updateWorkoutActivityPubAudience(c, user, entry, workout); err != nil {
			wc.context.Logger().Warn("Failed to update ActivityPub audience", "workout_id", workout.ID, "error", err)
		}

		// Already published: do not repost existing workouts and avoid duplicate deliveries.
		return
	}

	if err := wc.publishWorkoutToActivityPub(c, user, workout); err != nil {
		wc.context.Logger().Warn("Failed to publish workout to ActivityPub", "workout_id", workout.ID, "error", err)
	}
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
// @Failure      404  {object}  dto.Response[any]
// @Router       /workouts/{id}/download [get]
func (wc *workoutController) DownloadWorkout(c echo.Context) error {
	workout, err := wc.getOwnedWorkout(c)
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
