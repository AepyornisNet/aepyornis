package controller

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/AepyornisNet/aepyornis/pkg/aputil"
	"github.com/AepyornisNet/aepyornis/pkg/config"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/model/dto"
	"github.com/AepyornisNet/aepyornis/pkg/repository"
	"github.com/AepyornisNet/aepyornis/pkg/service"
	"github.com/labstack/echo/v5"
	"github.com/samber/do/v2"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

type WorkoutController interface {
	GetWorkouts(c *echo.Context) error
	GetWorkout(c *echo.Context) error
	GetWorkoutLikes(c *echo.Context) error
	GetWorkoutReplies(c *echo.Context) error
	LikeWorkout(c *echo.Context) error
	LikeWorkoutByObject(c *echo.Context) error
	CreateReply(c *echo.Context) error
	GetWorkoutBreakdown(c *echo.Context) error
	GetWorkoutRangeStats(c *echo.Context) error
	GetWorkoutCalendar(c *echo.Context) error
	CreateWorkout(c *echo.Context) error
	GetRecentWorkouts(c *echo.Context) error
	DeleteWorkout(c *echo.Context) error
	UpdateWorkout(c *echo.Context) error
	ToggleWorkoutLock(c *echo.Context) error
	RefreshWorkout(c *echo.Context) error
	DownloadWorkout(c *echo.Context) error
	DownloadWorkoutAttachment(c *echo.Context) error
	GetWorkoutFilterOptions(c *echo.Context) error
	DownloadWorkoutsZip(c *echo.Context) error
	AddEquipmentToWorkouts(c *echo.Context) error
}

type workoutController struct {
	apOutboxRepo     repository.APOutbox
	cfg              *config.Config
	db               *gorm.DB
	workoutLikeRepo  repository.WorkoutLike
	workoutReplyRepo repository.WorkoutReply
	workoutSvc       service.WorkoutService
}

func NewWorkoutController(injector do.Injector) WorkoutController {
	return &workoutController{
		apOutboxRepo:     do.MustInvoke[repository.APOutbox](injector),
		cfg:              do.MustInvoke[*config.Config](injector),
		db:               do.MustInvoke[*gorm.DB](injector),
		workoutLikeRepo:  do.MustInvoke[repository.WorkoutLike](injector),
		workoutReplyRepo: do.MustInvoke[repository.WorkoutReply](injector),
		workoutSvc:       do.MustInvoke[service.WorkoutService](injector),
	}
}

func (wc *workoutController) getOwnedWorkout(c *echo.Context) (*model.Workout, error) {
	workoutID, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return nil, err
	}
	viewer := currentUser(c)
	return wc.workoutSvc.GetOwnedWorkout(viewer, workoutID)
}

func (wc *workoutController) getReadableWorkout(c *echo.Context, withDetails bool) (*model.Workout, error) {
	workoutID, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return nil, err
	}
	viewer := currentUser(c)
	return wc.workoutSvc.GetReadableWorkout(viewer, workoutID, withDetails)
}

// GetWorkouts returns paginated workouts for the authenticated user
// @Summary      List user workouts
// @Tags         workouts
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        page      query  int     false "Page"
// @Param        per_page  query  int     false "Per page"
// @Param        sort      query  string  false "Sort field"
// @Param        order     query  string  false "Sort order"
// @Param        type      query  string  false "Filter by type"
// @Param        sub_type  query  string  false "Filter by subtype"
// @Param        start     query  string  false "Start date"
// @Param        end       query  string  false "End date"
// @Produce      json
// @Success      200  {object}  dto.PaginatedResponse[dto.WorkoutResponse]
// @Failure      400  {object}  dto.Response[string]
// @Failure      500  {object}  dto.Response[string]
// @Router       /workouts [get]
func (wc *workoutController) GetWorkouts(c *echo.Context) error {
	var pagination dto.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	pagination.SetDefaults()

	filters, err := model.GetWorkoutsFilters(c)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	user := currentUser(c)
	workouts, totalCount, err := wc.workoutSvc.GetWorkouts(c.Request().Context(), user, filters, pagination.PerPage, pagination.GetOffset())
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	results, err := wc.workoutSvc.EnrichWorkoutsMetadata(c.Request().Context(), user, workouts)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
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

// GetWorkout returns a single workout by ID
// @Summary      Get workout
// @Tags         workouts
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "Workout ID"
// @Produce      json
// @Success      200  {object}  dto.Response[dto.WorkoutDetailResponse]
// @Failure      400  {object}  dto.Response[string]
// @Failure      404  {object}  dto.Response[string]
// @Router       /workouts/{id} [get]
func (wc *workoutController) GetWorkout(c *echo.Context) error {
	workoutID, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	viewer := currentUser(c)
	workout, records, powerRecords, err := wc.workoutSvc.GetWorkoutDetails(c.Request().Context(), viewer, workoutID)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	result := dto.NewWorkoutDetailResponse(workout, records, powerRecords)
	ownerUserID := workoutOwnerUserID(workout)
	published, err := wc.apOutboxRepo.PublishedMap(ownerUserID, []uint64{workout.ID})
	if err == nil {
		result.ActivityPubPublished = published[workout.ID]
	}

	counts, err := wc.workoutLikeRepo.CountMapByWorkoutIDs([]uint64{workout.ID})
	if err == nil {
		result.LikesCount = counts[workout.ID]
	}

	liked, err := wc.workoutLikeRepo.LikedMapByProfile([]uint64{workout.ID}, viewer.Profile.ID)
	if err == nil {
		result.LikedByMe = liked[workout.ID]
	}

	replyCount, err := wc.workoutReplyRepo.CountByWorkoutID(workout.ID)
	if err == nil {
		result.RepliesCount = replyCount
	}

	resp := dto.Response[dto.WorkoutDetailResponse]{
		Results: result,
	}

	return c.JSON(http.StatusOK, resp)
}

// GetWorkoutLikes returns all likes for a workout
// @Summary      Get workout likes
// @Tags         workouts
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path      int  true  "Workout ID"
// @Produce      json
// @Success      200  {object}  dto.Response[[]dto.WorkoutLikeResponse]
// @Failure      400  {object}  dto.Response[string]
// @Failure      404  {object}  dto.Response[string]
// @Router       /workouts/{id}/likes [get]
func (wc *workoutController) GetWorkoutLikes(c *echo.Context) error {
	workout, err := wc.getReadableWorkout(c, false)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	results, err := wc.workoutSvc.GetWorkoutLikes(c.Request().Context(), workout.ID)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[[]dto.WorkoutLikeResponse]{
		Results: results,
	}

	return c.JSON(http.StatusOK, resp)
}

// GetWorkoutReplies returns paginated replies/comments for a workout
// @Summary      Get workout replies
// @Tags         workouts
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id        path      int  true  "Workout ID"
// @Param        page      query     int  false "Page"
// @Param        per_page  query     int  false "Per page"
// @Produce      json
// @Success      200  {object}  dto.PaginatedResponse[dto.WorkoutReplyResponse]
// @Failure      400  {object}  dto.Response[string]
// @Failure      404  {object}  dto.Response[string]
// @Router       /workouts/{id}/replies [get]
func (wc *workoutController) GetWorkoutReplies(c *echo.Context) error {
	workout, err := wc.getReadableWorkout(c, false)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	var pagination dto.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	pagination.SetDefaults()

	results, totalCount, err := wc.workoutSvc.GetWorkoutReplies(c.Request().Context(), workout.ID, pagination.Page, pagination.PerPage)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.PaginatedResponse[dto.WorkoutReplyResponse]{
		Results:    results,
		Page:       pagination.Page,
		PerPage:    pagination.PerPage,
		TotalPages: pagination.CalculateTotalPages(totalCount),
		TotalCount: totalCount,
	}

	return c.JSON(http.StatusOK, resp)
}

// LikeWorkout likes a local workout by ID
// @Summary      Like workout
// @Tags         workouts
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "Workout ID"
// @Produce      json
// @Success      200  {object}  dto.Response[map[string]string]
// @Failure      400  {object}  dto.Response[string]
// @Failure      404  {object}  dto.Response[string]
// @Router       /workouts/{id}/like [post]
func (wc *workoutController) LikeWorkout(c *echo.Context) error {
	viewer := currentUser(c)
	if viewer == nil || viewer.IsAnonymous() {
		return renderApiError(c, http.StatusForbidden, dto.ErrNotAuthorized)
	}

	workout, err := wc.getReadableWorkout(c, false)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	results, err := wc.workoutSvc.LikeWorkout(c.Request().Context(), viewer, workout)
	if err != nil {
		if err.Error() == "cannot like your own workout" {
			return renderApiError(c, http.StatusBadRequest, err)
		}
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[map[string]any]{
		Results: results,
	}

	return c.JSON(http.StatusOK, resp)
}

// LikeWorkoutByObject likes an ActivityPub workout object by object IRI
// @Summary      Like ActivityPub workout object
// @Tags         workouts
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Success      200  {object}  dto.Response[map[string]string]
// @Failure      400  {object}  dto.Response[string]
// @Failure      404  {object}  dto.Response[string]
// @Router       /workouts/like [post]
func (wc *workoutController) LikeWorkoutByObject(c *echo.Context) error {
	viewer := currentUser(c)
	if viewer == nil || viewer.IsAnonymous() {
		return renderApiError(c, http.StatusForbidden, dto.ErrNotAuthorized)
	}

	var params dto.LikeByObjectRequest
	if err := c.Bind(&params); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	if err := c.Validate(&params); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	params.ObjectID = strings.TrimSpace(params.ObjectID)

	results, status, err := wc.workoutSvc.LikeWorkoutByObject(c.Request().Context(), viewer, params.ObjectID, wc.localActorIRI(c, viewer))
	if err != nil {
		return renderApiError(c, status, err)
	}

	resp := dto.Response[map[string]any]{
		Results: results,
	}

	return c.JSON(status, resp)
}

// CreateReply creates a comment/reply for a workout
// @Summary      Create workout reply
// @Tags         workouts
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "Workout ID"
// @Accept       json
// @Produce      json
// @Success      201  {object}  dto.Response[dto.WorkoutReplyResponse]
// @Failure      400  {object}  dto.Response[string]
// @Failure      404  {object}  dto.Response[string]
// @Router       /workouts/{id}/replies [post]
func (wc *workoutController) CreateReply(c *echo.Context) error {
	viewer := currentUser(c)
	if viewer == nil || viewer.IsAnonymous() {
		return renderApiError(c, http.StatusForbidden, dto.ErrNotAuthorized)
	}

	workout, err := wc.getReadableWorkout(c, false)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	var req dto.WorkoutReplyCreateRequest
	if err := c.Bind(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	reply, err := wc.workoutSvc.CreateReply(c.Request().Context(), viewer, workout, strings.TrimSpace(req.Content))
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[dto.WorkoutReplyResponse]{
		Results: *reply,
	}

	return c.JSON(http.StatusCreated, resp)
}

// GetWorkoutBreakdown returns distance/lap breakdowns
// @Summary      Get workout breakdown
// @Tags         workouts
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id     path   int     true   "Workout ID"
// @Param        count  query  float64 false  "Distance per unit"
// @Param        unit   query  string  false  "Unit (km, mi, lap)"
// @Produce      json
// @Success      200  {object}  dto.Response[dto.WorkoutBreakdownResponse]
// @Failure      400  {object}  dto.Response[string]
// @Failure      404  {object}  dto.Response[string]
// @Router       /workouts/{id}/breakdown [get]
func (wc *workoutController) GetWorkoutBreakdown(c *echo.Context) error {
	var params dto.WorkoutBreakdownRequest
	if err := c.Bind(&params); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	if err := c.Validate(&params); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	workout, err := wc.getReadableWorkout(c, true)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	requester := currentUser(c)
	breakdown, err := wc.workoutSvc.GetWorkoutBreakdown(c.Request().Context(), workout, requester.PreferredUnits.Distance(), params)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	resp := dto.Response[dto.WorkoutBreakdownResponse]{
		Results: breakdown,
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
// @Failure      400  {object}  dto.Response[string]
// @Failure      404  {object}  dto.Response[string]
// @Router       /workouts/{id}/stats-range [get]
func (wc *workoutController) GetWorkoutRangeStats(c *echo.Context) error {
	var params dto.WorkoutRangeStatsRequest
	if err := c.Bind(&params); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	if err := c.Validate(&params); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	workout, err := wc.getReadableWorkout(c, false)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	stats, startIdx, endIdx, err := wc.workoutSvc.GetWorkoutRangeStats(workout, params.StartIndex, params.EndIndex)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	resp := dto.Response[dto.WorkoutRangeStatsResponse]{
		Results: dto.NewWorkoutRangeStatsResponse(*stats, startIdx, endIdx),
	}

	return c.JSON(http.StatusOK, resp)
}

// GetWorkoutCalendar returns workout calendar events
// @Summary      Get workout calendar events
// @Tags         workouts
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        handle    query  string false "User handle"
// @Param        start     query  string false "Start date"
// @Param        end       query  string false "End date"
// @Param        time_zone query  string false "Time zone"
// @Produce      json
// @Success      200  {object}  dto.Response[[]dto.CalendarEventResponse]
// @Failure      400  {object}  dto.Response[string]
// @Failure      404  {object}  dto.Response[string]
// @Failure      500  {object}  dto.Response[string]
// @Router       /workouts/calendar [get]
func (wc *workoutController) GetWorkoutCalendar(c *echo.Context) error {
	viewer := currentUser(c)
	targetUser := viewer

	var params dto.CalendarQueryParams
	if err := c.Bind(&params); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	events, err := wc.workoutSvc.GetWorkoutCalendar(c.Request().Context(), targetUser, viewer, params.Start, params.End, params.TimeZone)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
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
// @Param        file        formData  file   false "Workout file(s)"
// @Param        notes       formData  string false "Notes"
// @Param        type        formData  string false "Workout type"
// @Param        visibility  formData  string false "Visibility (public, followers, private)"
// @Param        name        formData  string false "Workout name"
// @Success      201  {object}  dto.Response[dto.WorkoutResponse]
// @Failure      400  {object}  dto.Response[string]
// @Failure      500  {object}  dto.Response[string]
// @Router       /workouts [post]
func (wc *workoutController) CreateWorkout(c *echo.Context) error {
	user := currentUser(c)

	if c.Request().Header.Get(echo.HeaderContentType) != "" &&
		strings.HasPrefix(c.Request().Header.Get(echo.HeaderContentType), echo.MIMEMultipartForm) {
		return wc.createWorkoutFromFile(c, user)
	}

	return wc.createWorkoutManual(c, user)
}

func (wc *workoutController) createWorkoutFromFile(c *echo.Context, user *model.User) error {
	form, err := c.MultipartForm()
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	files := form.File["file"]
	if len(files) == 0 {
		return renderApiError(c, http.StatusBadRequest, errors.New("no file uploaded"))
	}

	notes := multipartFormValue(form, c, "notes")
	workoutType := model.WorkoutType(multipartFormValue(form, c, "type"))
	visibilityRaw := multipartFormValue(form, c, "visibility")
	hasVisibility := hasMultipartFormValue(form, c, "visibility")

	var visibility *model.WorkoutVisibility
	if hasVisibility {
		if visibilityRaw == "private" {
			v := model.WorkoutVisibilityPrivate
			visibility = &v
		} else if vis := model.WorkoutVisibility(visibilityRaw); vis.IsValid() {
			visibility = &vis
		}
	}

	names := form.Value["name"]
	if len(names) == 0 {
		names = form.Value["names"]
	}

	uploadedFiles := make([]service.UploadedWorkoutFile, 0, len(files))
	for _, file := range files {
		content, parseErr := uploadedFile(file)
		if parseErr != nil {
			return renderApiError(c, http.StatusBadRequest, parseErr)
		}
		uploadedFiles = append(uploadedFiles, service.UploadedWorkoutFile{
			Filename: file.Filename,
			Content:  content,
		})
	}

	ws, errList := wc.workoutSvc.CreateWorkoutsFromFile(c.Request().Context(), user, uploadedFiles, notes, workoutType, visibility, names)

	createdWorkouts := make([]dto.WorkoutResponse, len(ws))
	for i, w := range ws {
		createdWorkouts[i] = dto.NewWorkoutResponse(w)
	}

	resp := dto.Response[[]dto.WorkoutResponse]{
		Results: createdWorkouts,
	}

	if len(errList) > 0 {
		resp.AddContextError(c.Request().Context(), errList...)
		for _, e := range errList {
			if code := apiErrorCode(e); code != "" {
				resp.ErrorCodes = append(resp.ErrorCodes, code)
			}
		}
	}

	statusCode := http.StatusCreated
	if len(createdWorkouts) == 0 && len(errList) > 0 {
		statusCode = http.StatusBadRequest
		allDuplicates := true
		for _, e := range errList {
			if !errors.Is(e, model.ErrWorkoutAlreadyExists) {
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

func (wc *workoutController) createWorkoutManual(c *echo.Context, user *model.User) error {
	d := &dto.ManualWorkout{Units: &user.PreferredUnits}
	if err := c.Bind(d); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	workout, err := wc.workoutSvc.CreateWorkoutManual(c.Request().Context(), user, d)
	if err != nil {
		if errors.Is(err, model.ErrWorkoutAlreadyExists) {
			return renderApiError(c, http.StatusConflict, err)
		}
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[dto.WorkoutResponse]{
		Results: dto.NewWorkoutResponse(workout),
	}

	return c.JSON(http.StatusCreated, resp)
}

// GetRecentWorkouts returns recent workouts from all users
// @Summary      List recent workouts
// @Tags         workouts
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        count  query  int  false "Count of workouts to return"
// @Param        offset query  int  false "Offset"
// @Produce      json
// @Success      200  {object}  dto.Response[[]dto.WorkoutResponse]
// @Failure      500  {object}  dto.Response[string]
// @Router       /workouts/recent [get]
func (wc *workoutController) GetRecentWorkouts(c *echo.Context) error {
	requester := currentUser(c)

	var req dto.RecentWorkoutsRequest
	req.Limit = 20
	if err := c.Bind(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	workouts, err := wc.workoutSvc.GetRecentWorkouts(c.Request().Context(), requester, req, c.Request().Host)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	results, err := wc.workoutSvc.EnrichWorkoutsMetadata(c.Request().Context(), requester, workouts)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

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
// @Failure      404  {object}  dto.Response[string]
// @Failure      500  {object}  dto.Response[string]
// @Router       /workouts/{id} [delete]
func (wc *workoutController) DeleteWorkout(c *echo.Context) error {
	user := currentUser(c)
	workout, err := wc.getOwnedWorkout(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	if err := wc.workoutSvc.DeleteWorkout(c.Request().Context(), user, workout); err != nil {
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
// @Failure      400  {object}  dto.Response[string]
// @Failure      404  {object}  dto.Response[string]
// @Router       /workouts/{id} [put]
func (wc *workoutController) UpdateWorkout(c *echo.Context) error {
	user := currentUser(c)
	workout, err := wc.getOwnedWorkout(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	d := &dto.ManualWorkout{Units: &user.PreferredUnits}
	if err := c.Bind(d); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	updated, err := wc.workoutSvc.UpdateWorkout(c.Request().Context(), user, workout, d)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[dto.WorkoutResponse]{
		Results: dto.NewWorkoutResponse(updated),
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
// @Failure      404  {object}  dto.Response[string]
// @Failure      403  {object}  dto.Response[string]
// @Router       /workouts/{id}/toggle-lock [post]
func (wc *workoutController) ToggleWorkoutLock(c *echo.Context) error {
	user := currentUser(c)
	workout, err := wc.getOwnedWorkout(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	if err := wc.workoutSvc.ToggleWorkoutLock(c.Request().Context(), user, workout); err != nil {
		if err.Error() == "not authorized" {
			return renderApiError(c, http.StatusForbidden, err)
		}
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[dto.WorkoutResponse]{
		Results: dto.NewWorkoutResponse(workout),
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
// @Failure      404  {object}  dto.Response[string]
// @Router       /workouts/{id}/refresh [post]
func (wc *workoutController) RefreshWorkout(c *echo.Context) error {
	workout, err := wc.getOwnedWorkout(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	if err := wc.workoutSvc.RefreshWorkout(c.Request().Context(), workout); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[map[string]string]{
		Results: map[string]string{"message": "Workout will be refreshed soon"},
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
// @Failure      404  {object}  dto.Response[string]
// @Router       /workouts/{id}/download [get]
func (wc *workoutController) DownloadWorkout(c *echo.Context) error {
	workout, err := wc.getOwnedWorkout(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	if !workout.HasFile() {
		return renderApiError(c, http.StatusNotFound, errors.New("workout has no file"))
	}

	basename := workout.File.Filename
	if basename == "" {
		basename = "workout_" + strconv.FormatUint(workout.ID, 10) + ".gpx"
	}

	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename=\""+basename+"\"")
	return c.Blob(http.StatusOK, "application/binary", workout.File.Content)
}

// DownloadWorkoutAttachment downloads a workout attachment
// @Summary      Download workout attachment
// @Tags         workouts
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id             path  int  true  "Workout ID"
// @Param        attachment_id  path  int  true  "Attachment ID"
// @Produce      octet-stream
// @Success      200  {string}  string  "binary attachment file"
// @Failure      404  {object}  dto.Response[string]
// @Router       /workouts/{id}/attachments/{attachment_id} [get]
func (wc *workoutController) DownloadWorkoutAttachment(c *echo.Context) error {
	workout, err := wc.getReadableWorkout(c, false)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	attachmentID, err := cast.ToUint64E(c.Param("attachment_id"))
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	var attachment model.WorkoutAttachment
	if err := wc.db.
		Where("id = ? AND workout_id = ?", attachmentID, workout.ID).
		First(&attachment).Error; err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	c.Response().Header().Set(echo.HeaderContentDisposition, "inline; filename=\""+attachment.Filename+"\"")
	return c.Blob(http.StatusOK, attachment.ContentType, attachment.Content)
}

func (wc *workoutController) GetWorkoutFilterOptions(c *echo.Context) error {
	user := currentUser(c)
	types, subTypesByType, err := wc.workoutSvc.GetWorkoutFilterOptions(c.Request().Context(), user.Profile.ID)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, dto.Response[map[string]any]{
		Results: map[string]any{
			"types":             types,
			"sub_types_by_type": subTypesByType,
		},
	})
}

// DownloadWorkoutsZip packs multiple workouts into a single ZIP file and returns it.
// @Summary      Download multiple workouts as ZIP
// @Tags         workouts
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        ids  query  []int  true  "Workout IDs to include"
// @Produce      octet-stream
// @Success      200  {string}  string  "binary zip file"
// @Failure      400  {object}  dto.Response[string]
// @Failure      404  {object}  dto.Response[string]
// @Failure      500  {object}  dto.Response[string]
// @Router       /workouts/download-zip [get]
func (wc *workoutController) DownloadWorkoutsZip(c *echo.Context) error {
	user := currentUser(c)

	var req dto.WorkoutsDownloadZipRequest
	if err := c.Bind(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	zipBytes, err := wc.workoutSvc.ExportWorkoutsZip(c.Request().Context(), user.Profile.ID, req.IDs)
	if err != nil {
		if err.Error() == "no workouts found" {
			return renderApiError(c, http.StatusNotFound, err)
		}
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename=\"workouts.zip\"")
	return c.Blob(http.StatusOK, "application/zip", zipBytes)
}

// AddEquipmentToWorkouts links multiple equipment items to multiple workouts.
// @Summary      Bulk add equipment to workouts
// @Tags         workouts
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Param        request  body  object  true  "Workout and Equipment IDs mapping"
// @Success      200  {object}  dto.Response[map[string]string]
// @Failure      400  {object}  dto.Response[string]
// @Failure      500  {object}  dto.Response[string]
// @Router       /workouts/add-equipment [post]
func (wc *workoutController) AddEquipmentToWorkouts(c *echo.Context) error {
	user := currentUser(c)

	var req dto.BulkAddEquipmentRequest
	if err := c.Bind(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	if err := wc.workoutSvc.AddEquipmentToWorkouts(c.Request().Context(), user.Profile.ID, req.WorkoutIDs, req.EquipmentIDs); err != nil {
		if err.Error() == "no valid equipment found" {
			return renderApiError(c, http.StatusBadRequest, err)
		}
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[map[string]string]{
		Results: map[string]string{"message": "Equipment added to workouts successfully"},
	}
	return c.JSON(http.StatusOK, resp)
}

func (wc *workoutController) localActorIRI(c *echo.Context, user *model.User) string {
	if user == nil {
		return ""
	}
	return aputil.LocalActorURL(aputil.LocalActorURLConfig{
		Host:           wc.cfg.Host,
		WebRoot:        wc.cfg.WebRoot,
		FallbackHost:   c.Request().Host,
		FallbackScheme: c.Scheme(),
	}, user.Profile.Username)
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

func workoutOwnerUserID(workout *model.Workout) uint64 {
	if workout == nil || workout.Profile == nil || workout.Profile.UserID == nil {
		return 0
	}
	return *workout.Profile.UserID
}
