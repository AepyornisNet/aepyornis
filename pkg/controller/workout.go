package controller

import (
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/aputil"
	"github.com/AepyornisNet/aepyornis/pkg/config"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/model/dto"
	"github.com/AepyornisNet/aepyornis/pkg/repository"
	"github.com/AepyornisNet/aepyornis/pkg/service"
	"github.com/AepyornisNet/aepyornis/pkg/worker"
	"github.com/labstack/echo/v4"
	"github.com/samber/do/v2"
	"github.com/spf13/cast"
	"github.com/vgarvardt/gue/v6"
	"gorm.io/gorm"
)

type WorkoutController interface {
	GetWorkouts(c echo.Context) error
	GetWorkout(c echo.Context) error
	GetWorkoutLikes(c echo.Context) error
	GetWorkoutReplies(c echo.Context) error
	LikeWorkout(c echo.Context) error
	LikeWorkoutByObject(c echo.Context) error
	CreateReply(c echo.Context) error
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
	DownloadWorkoutAttachment(c echo.Context) error
}

type workoutController struct {
	apOutboxRepo         repository.APOutbox
	apStatusDeliveryRepo repository.APStatusDelivery
	apProfileService     service.ActivityPubProfileService
	cfg                  *config.Config
	client               *gue.Client
	db                   *gorm.DB
	equipmentRepo        repository.Equipment
	logger               *slog.Logger
	actorService         service.ActivityPubActorService
	userRepo             repository.User
	workoutLikeRepo      repository.WorkoutLike
	workoutReplyRepo     repository.WorkoutReply
	workoutRepo          repository.Workout
}

var _ WorkoutController = (*workoutController)(nil)

func NewWorkoutController(injector do.Injector) WorkoutController {
	return &workoutController{
		apOutboxRepo:         do.MustInvoke[repository.APOutbox](injector),
		apStatusDeliveryRepo: do.MustInvoke[repository.APStatusDelivery](injector),
		apProfileService:     do.MustInvoke[service.ActivityPubProfileService](injector),
		cfg:                  do.MustInvoke[*config.Config](injector),
		client:               do.MustInvoke[*gue.Client](injector),
		db:                   do.MustInvoke[*gorm.DB](injector),
		equipmentRepo:        do.MustInvoke[repository.Equipment](injector),
		logger:               do.MustInvoke[*slog.Logger](injector),
		actorService:         do.MustInvoke[service.ActivityPubActorService](injector),
		userRepo:             do.MustInvoke[repository.User](injector),
		workoutLikeRepo:      do.MustInvoke[repository.WorkoutLike](injector),
		workoutReplyRepo:     do.MustInvoke[repository.WorkoutReply](injector),
		workoutRepo:          do.MustInvoke[repository.Workout](injector),
	}
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

func applyLikeMetadata(results []dto.WorkoutResponse, counts map[uint64]int64, liked map[uint64]bool) {
	for i := range results {
		results[i].LikesCount = counts[results[i].ID]
		results[i].LikedByMe = liked[results[i].ID]
	}
}

func applyReplyMetadata(results []dto.WorkoutResponse, counts map[uint64]int64) {
	for i := range results {
		results[i].RepliesCount = counts[results[i].ID]
	}
}

func (wc *workoutController) getOwnedWorkout(c echo.Context) (*model.Workout, error) {
	id, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return nil, err
	}

	user := currentUser(c)
	w, err := wc.workoutRepo.GetByUserID(user.Profile.ID, id)
	if err != nil {
		return nil, err
	}

	if w.Profile != nil {
		w.Profile.User = user
	}

	return w, nil
}

func workoutOwnerUserID(workout *model.Workout) uint64 {
	if workout == nil || workout.Profile == nil || workout.Profile.UserID == nil {
		return 0
	}

	return *workout.Profile.UserID
}

func (wc *workoutController) canReadWorkout(requester *model.User, workout *model.Workout) (bool, error) {
	if requester == nil || workout == nil {
		return false, nil
	}

	ownerUserID := workoutOwnerUserID(workout)
	if ownerUserID != 0 && requester.ID == ownerUserID {
		return true, nil
	}

	switch workout.Visibility {
	case model.WorkoutVisibilityPublic:
		return true, nil
	case model.WorkoutVisibilityFollowers:
		if requester.Profile.ID == 0 {
			return false, nil
		}

		var count int64
		if err := wc.db.
			Model(&model.Follower{}).
			Where("profile_id = ? AND following_profile_id = ? AND approved = ?", requester.Profile.ID, workout.ProfileID, true).
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

	workout, err := wc.workoutRepo.GetByIDForRead(id, withDetails)
	if err != nil {
		return nil, err
	}

	allowed, err := wc.canReadWorkout(currentUser(c), workout)
	if err != nil {
		return nil, err
	}

	if !allowed {
		return nil, gorm.ErrRecordNotFound
	}

	return workout, nil
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
	user := currentUser(c)

	var pagination dto.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	pagination.SetDefaults()

	filters, err := model.GetWorkoutsFilters(c)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	totalCount, err := wc.workoutRepo.CountByUserAndFilters(user.ID, filters)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	workouts, err := wc.workoutRepo.ListByUserAndFilters(user.ID, filters, pagination.PerPage, pagination.GetOffset())
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	results := dto.NewWorkoutsResponse(workouts)
	published, err := wc.apOutboxRepo.PublishedMap(user.ID, workoutIDs(workouts))
	if err == nil {
		applyPublishedFlags(results, published)
	}

	counts, err := wc.workoutLikeRepo.CountMapByWorkoutIDs(workoutIDs(workouts))
	if err == nil {
		liked, likedErr := wc.workoutLikeRepo.LikedMapByProfile(workoutIDs(workouts), user.Profile.ID)
		if likedErr == nil {
			applyLikeMetadata(results, counts, liked)
		}
	}

	replyCounts, err := wc.workoutReplyRepo.CountMapByWorkoutIDs(workoutIDs(workouts))
	if err == nil {
		applyReplyMetadata(results, replyCounts)
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

	ownerUserID := workoutOwnerUserID(workout)
	records, err := model.GetWorkoutIntervalRecordsWithRank(wc.db, ownerUserID, workout.Type, workout.ID)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	result := dto.NewWorkoutDetailResponse(workout, records)
	published, err := wc.apOutboxRepo.PublishedMap(ownerUserID, []uint64{workout.ID})
	if err == nil {
		result.ActivityPubPublished = published[workout.ID]
	}

	counts, err := wc.workoutLikeRepo.CountMapByWorkoutIDs([]uint64{workout.ID})
	if err == nil {
		result.LikesCount = counts[workout.ID]
	}

	liked, err := wc.workoutLikeRepo.LikedMapByProfile([]uint64{workout.ID}, currentUser(c).Profile.ID)
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
// @Failure      400  {object}  dto.Response[any]
// @Failure      404  {object}  dto.Response[any]
// @Router       /workouts/{id}/likes [get]
func (wc *workoutController) GetWorkoutLikes(c echo.Context) error {
	workout, err := wc.getReadableWorkout(c, false)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	likes, err := wc.workoutLikeRepo.ListByWorkoutID(workout.ID)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	results := make([]dto.WorkoutLikeResponse, 0, len(likes))
	for i := range likes {
		likeResponse := dto.NewWorkoutLikeResponse(&likes[i])
		if likes[i].Profile != nil && likes[i].Profile.UserID == nil {
			if actorIRI := strings.TrimSpace(likes[i].Profile.ActorURL()); actorIRI != "" {
				profile, err := wc.apProfileService.GetByActorIRI(c.Request().Context(), actorIRI)
				if err == nil && profile != nil {
					if name := strings.TrimSpace(profile.DisplayName); name != "" {
						likeResponse.ActorName = &name
					}
					if profile.AvatarRemoteURL != nil && strings.TrimSpace(*profile.AvatarRemoteURL) != "" {
						avatarURL := strings.TrimSpace(*profile.AvatarRemoteURL)
						likeResponse.AvatarURL = &avatarURL
					}
				}
			}
		}

		results = append(results, likeResponse)
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
// @Failure      400  {object}  dto.Response[any]
// @Failure      404  {object}  dto.Response[any]
// @Router       /workouts/{id}/replies [get]
func (wc *workoutController) GetWorkoutReplies(c echo.Context) error {
	workout, err := wc.getReadableWorkout(c, false)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	var pagination dto.PaginationParams
	pagination.SetDefaults()
	if pageStr := c.QueryParam("page"); pageStr != "" {
		if page, parseErr := strconv.Atoi(pageStr); parseErr == nil {
			pagination.Page = page
		}
	}
	if perPageStr := c.QueryParam("per_page"); perPageStr != "" {
		if perPage, parseErr := strconv.Atoi(perPageStr); parseErr == nil {
			pagination.PerPage = perPage
		}
	}
	pagination.SetDefaults()

	totalCount, err := wc.workoutReplyRepo.CountByWorkoutID(workout.ID)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	replies, err := wc.workoutReplyRepo.ListByWorkoutID(workout.ID, pagination.PerPage, pagination.GetOffset())
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	results := make([]dto.WorkoutReplyResponse, 0, len(replies))
	for i := range replies {
		replyResponse := dto.NewWorkoutReplyResponse(&replies[i])
		if replies[i].Profile != nil && replies[i].Profile.UserID == nil {
			if actorIRI := strings.TrimSpace(replies[i].Profile.ActorURL()); actorIRI != "" {
				profile, err := wc.apProfileService.GetByActorIRI(c.Request().Context(), actorIRI)
				if err == nil && profile != nil {
					if replyResponse.ActorName == nil {
						if name := strings.TrimSpace(profile.DisplayName); name != "" {
							replyResponse.ActorName = &name
						}
					}
					if profile.AvatarRemoteURL != nil && strings.TrimSpace(*profile.AvatarRemoteURL) != "" {
						avatarURL := strings.TrimSpace(*profile.AvatarRemoteURL)
						replyResponse.AvatarURL = &avatarURL
					}
				}
			}
		}

		results = append(results, replyResponse)
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
// @Success      200  {object}  dto.Response[map[string]any]
// @Failure      400  {object}  dto.Response[any]
// @Failure      404  {object}  dto.Response[any]
// @Router       /workouts/{id}/like [post]
func (wc *workoutController) LikeWorkout(c echo.Context) error {
	viewer := currentUser(c)
	if viewer == nil || viewer.IsAnonymous() {
		return renderApiError(c, http.StatusForbidden, dto.ErrNotAuthorized)
	}

	workout, err := wc.getReadableWorkout(c, false)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	if workoutOwnerUserID(workout) == viewer.ID {
		return renderApiError(c, http.StatusBadRequest, errors.New("cannot like your own workout"))
	}

	if err := wc.workoutLikeRepo.LikeByProfile(workout.ID, viewer.Profile.ID); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	counts, err := wc.workoutLikeRepo.CountMapByWorkoutIDs([]uint64{workout.ID})
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[map[string]any]{
		Results: map[string]any{
			"workout_id":  workout.ID,
			"likes_count": counts[workout.ID],
			"liked":       true,
		},
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
// @Success      200  {object}  dto.Response[map[string]any]
// @Failure      400  {object}  dto.Response[any]
// @Failure      404  {object}  dto.Response[any]
// @Router       /workouts/like [post]
func (wc *workoutController) LikeWorkoutByObject(c echo.Context) error {
	viewer := currentUser(c)
	if viewer == nil || viewer.IsAnonymous() {
		return renderApiError(c, http.StatusForbidden, dto.ErrNotAuthorized)
	}

	params := struct {
		ObjectID string `json:"object_id" form:"object_id" query:"object_id"`
	}{}
	if err := c.Bind(&params); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	params.ObjectID = strings.TrimSpace(params.ObjectID)
	if params.ObjectID == "" {
		return renderApiError(c, http.StatusBadRequest, errors.New("object_id is required"))
	}

	localWorkoutID, localErr := wc.apOutboxRepo.ResolveWorkoutIDByObjectOrActivityID(0, params.ObjectID)
	if localErr == nil {
		results, status, err := wc.likeLocalWorkout(c, viewer, localWorkoutID)
		if err != nil {
			return renderApiError(c, status, err)
		}

		resp := dto.Response[map[string]any]{
			Results: results,
		}

		return c.JSON(status, resp)
	}

	if !errors.Is(localErr, gorm.ErrRecordNotFound) {
		return renderApiError(c, http.StatusInternalServerError, localErr)
	}

	if !viewer.ActivityPubEnabled() {
		return renderApiError(c, http.StatusBadRequest, errors.New("activitypub must be enabled to like remote workouts"))
	}

	actorIRI, inbox, err := aputil.ResolveObjectActorAndInbox(c.Request().Context(), params.ObjectID)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	viewerActorIRI := wc.localActorIRI(c, viewer)
	if actorIRI == viewerActorIRI {
		return renderApiError(c, http.StatusBadRequest, errors.New("cannot like your own workout"))
	}

	if err := wc.actorService.SendLike(c.Request().Context(), &viewer.Profile, inbox, params.ObjectID); err != nil {
		return renderApiError(c, http.StatusBadGateway, err)
	}

	resp := dto.Response[map[string]any]{
		Results: map[string]any{
			"object_id": params.ObjectID,
			"liked":     true,
		},
	}

	return c.JSON(http.StatusOK, resp)
}

func (wc *workoutController) likeLocalWorkout(c echo.Context, viewer *model.User, localWorkoutID uint64) (map[string]any, int, error) {
	workout, err := wc.workoutRepo.GetByIDForRead(localWorkoutID, false)
	if err != nil {
		return nil, http.StatusNotFound, err
	}

	allowed, err := wc.canReadWorkout(viewer, workout)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	if !allowed {
		return nil, http.StatusNotFound, gorm.ErrRecordNotFound
	}

	if workoutOwnerUserID(workout) == viewer.ID {
		return nil, http.StatusBadRequest, errors.New("cannot like your own workout")
	}

	if err := wc.workoutLikeRepo.LikeByProfile(localWorkoutID, viewer.Profile.ID); err != nil {
		return nil, http.StatusInternalServerError, err
	}

	counts, err := wc.workoutLikeRepo.CountMapByWorkoutIDs([]uint64{localWorkoutID})
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	return map[string]any{
		"workout_id":  localWorkoutID,
		"likes_count": counts[localWorkoutID],
		"liked":       true,
	}, http.StatusOK, nil
}

// CreateReply creates a reply/comment on a workout
// @Summary      Create a reply on a workout
// @Tags         workouts
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Param        id   path  int  true  "Workout ID"
// @Param        payload body  object{content=string}  true  "Reply content"
// @Success      201  {object}  dto.Response[dto.WorkoutReplyResponse]
// @Failure      400  {object}  dto.Response[any]
// @Failure      404  {object}  dto.Response[any]
// @Router       /workouts/{id}/replies [post]
func (wc *workoutController) CreateReply(c echo.Context) error {
	viewer := currentUser(c)

	workout, err := wc.getReadableWorkout(c, false)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	params := struct {
		Content string `json:"content"`
	}{}

	if err := c.Bind(&params); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	params.Content = strings.TrimSpace(params.Content)

	if params.Content == "" {
		return renderApiError(c, http.StatusBadRequest, errors.New("content is required"))
	}

	reply, err := wc.workoutReplyRepo.CreateLocalReply(workout.ID, viewer.Profile.ID, params.Content)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	// Reload reply with profile data
	if err := wc.db.Preload("Profile").Preload("Profile.User").First(reply, reply.ID).Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if err := worker.PublishReplyToActivityPub(c.Request().Context(), wc.client, wc.db, wc.cfg, wc.apOutboxRepo, wc.apStatusDeliveryRepo, viewer, workout, reply); err != nil {
		wc.logger.Warn("Failed to publish workout reply to ActivityPub", "reply_id", reply.ID, "error", err)
	}

	replyResponse := dto.NewWorkoutReplyResponse(reply)

	resp := dto.Response[dto.WorkoutReplyResponse]{
		Results: replyResponse,
	}

	return c.JSON(http.StatusCreated, resp)
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
	requester := currentUser(c)

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

	preferLaps := (params.Mode == "" || params.Mode == "auto" || params.Mode == "laps") && len(workout.Laps) > 1

	if preferLaps {
		resp.Results = dto.WorkoutBreakdownResponse{
			Mode:  "laps",
			Items: dto.NewWorkoutBreakdownItemsFromLaps(workout.Laps, workout.Records, &requester.PreferredUnits),
		}

		return c.JSON(http.StatusOK, resp)
	}

	if workout.Data == nil {
		return renderApiError(c, http.StatusBadRequest, errors.New("workout has no map data"))
	}

	breakdown, err := workout.StatisticsPer(params.Count, requester.PreferredUnits.Distance())
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	resp.Results = dto.WorkoutBreakdownResponse{
		Mode:  "unit",
		Items: dto.NewWorkoutBreakdownItemsFromUnit(breakdown.Items, breakdown.Unit, params.Count, &requester.PreferredUnits),
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
	requester := currentUser(c)

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

	if workout.Data == nil || len(workout.Records) == 0 {
		return renderApiError(c, http.StatusBadRequest, errors.New("workout has no map data"))
	}

	points := workout.Records
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

	stats, ok := model.StatsForRange(workout.Records, startIdx, endIdx)
	if !ok {
		return renderApiError(c, http.StatusBadRequest, errors.New("invalid range"))
	}

	resp := dto.Response[dto.WorkoutRangeStatsResponse]{
		Results: dto.NewWorkoutRangeStatsResponse(stats, startIdx, endIdx, &requester.PreferredUnits),
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
	viewer := currentUser(c)
	targetUser := viewer
	if handle := strings.TrimSpace(c.QueryParam("handle")); handle != "" {
		var err error
		host := wc.cfg.Host
		if host == "" {
			host = c.Request().Host
		}
		targetUser, err = wc.userRepo.GetByHandle(handle, host)
		if err != nil {
			return renderApiError(c, http.StatusNotFound, err)
		}
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
		model.PreloadWorkoutData(wc.db),
		targetUser.Profile.ID,
		viewer.Profile.ID,
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

		if w.TotalDistance > 0 {
			title += " - " + formatDistance(w.TotalDistance)
		}
		if w.TotalDuration.Seconds() > 0 {
			title += " " + formatDuration(int64(w.TotalDuration.Seconds()))
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

func (wc *workoutController) localActorIRI(c echo.Context, user *model.User) string {
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
	user := currentUser(c)

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

		user.Profile.User = user
		ws, addErr := user.Profile.AddWorkout(wc.db, workoutType, notes, file.Filename, content)
		if len(addErr) > 0 {
			for _, e := range addErr {
				errList = append(errList, e)
			}
			continue
		}

		for _, w := range ws {
			createdWorkouts = append(createdWorkouts, dto.NewWorkoutResponse(w))

			if err := worker.EnqueueWorkoutUpdate(c.Request().Context(), wc.client, w.ID); err != nil {
				wc.logger.Error("Failed to enqueue workout update", "workout_id", w.ID, "error", err)
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
	d := &dto.ManualWorkout{Units: &user.PreferredUnits}
	if err := c.Bind(d); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	workout := &model.Workout{}
	if err := d.Update(workout); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	if d.Visibility == nil {
		workout.Visibility = user.EffectiveDefaultWorkoutVisibility()
	}

	workout.Profile = &user.Profile
	workout.ProfileID = user.Profile.ID
	workout.Creator = "web-interface"

	equipment, err := wc.equipmentRepo.GetByUserIDs(user.Profile.ID, d.EquipmentIDs)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	if err := workout.Save(wc.db); err != nil {
		if errors.Is(err, model.ErrWorkoutAlreadyExists) {
			return renderApiError(c, http.StatusConflict, err)
		}

		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if err := wc.db.Model(&workout).Association("Equipment").Replace(equipment); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if err := model.PreloadWorkoutDetails(wc.db).Preload("Equipment").First(&workout, workout.ID).Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if err := worker.EnqueueWorkoutUpdate(c.Request().Context(), wc.client, workout.ID); err != nil {
		wc.logger.Error("Failed to enqueue workout update", "workout_id", workout.ID, "error", err)
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
// @Param        limit   query  int     false "Limit"
// @Param        offset  query  int     false "Offset"
// @Param        scope   query  string  false "Feed scope (following|global)"
// @Success      200  {object}  dto.Response[[]dto.WorkoutResponse]
// @Failure      500  {object}  dto.Response[any]
// @Router       /workouts/recent [get]
func (wc *workoutController) GetRecentWorkouts(c echo.Context) error {
	requester := currentUser(c)

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

	scope := "following"
	if c.QueryParam("scope") == "global" {
		scope = "global"
	}

	var workouts []*model.Workout
	query := wc.db.
		Scopes(model.PreloadWorkoutData).
		Preload("Profile").
		Preload("Profile.User")

	if scope == "global" {
		query = query.Where(
			`workouts.profile_id = ? OR workouts.visibility = ? OR (
				workouts.visibility = ? AND
				EXISTS (
					SELECT 1
					FROM followers f
					WHERE f.profile_id = ?
						AND f.following_profile_id = workouts.profile_id
						AND f.approved = ?
				)
			)`,
			requester.Profile.ID,
			model.WorkoutVisibilityPublic,
			model.WorkoutVisibilityFollowers,
			requester.Profile.ID,
			true,
		)
	} else {
		query = query.Where(
			`workouts.profile_id = ? OR (
				(workouts.visibility = ? OR workouts.visibility = ?) AND
				EXISTS (
					SELECT 1
					FROM followers f
					WHERE f.profile_id = ?
						AND f.following_profile_id = workouts.profile_id
						AND f.approved = ?
				)
			)`,
			requester.Profile.ID,
			model.WorkoutVisibilityPublic,
			model.WorkoutVisibilityFollowers,
			requester.Profile.ID,
			true,
		)
	}

	err := query.
		Order("date DESC").
		Limit(limit).
		Offset(offset).
		Find(&workouts).Error
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	results := dto.NewWorkoutsResponse(workouts)

	counts, err := wc.workoutLikeRepo.CountMapByWorkoutIDs(workoutIDs(workouts))
	if err == nil {
		liked, likedErr := wc.workoutLikeRepo.LikedMapByProfile(workoutIDs(workouts), requester.Profile.ID)
		if likedErr == nil {
			applyLikeMetadata(results, counts, liked)
		}
	}

	replyCounts, err := wc.workoutReplyRepo.CountMapByWorkoutIDs(workoutIDs(workouts))
	if err == nil {
		applyReplyMetadata(results, replyCounts)
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
// @Failure      404  {object}  dto.Response[any]
// @Failure      500  {object}  dto.Response[any]
// @Router       /workouts/{id} [delete]
func (wc *workoutController) DeleteWorkout(c echo.Context) error {
	user := currentUser(c)

	workout, err := wc.getOwnedWorkout(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	if err := wc.apOutboxRepo.DeleteEntryForWorkout(user.ID, workout.ID); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if err := workout.Delete(wc.db); err != nil {
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
	user := currentUser(c)

	workout, err := wc.getOwnedWorkout(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	d := &dto.ManualWorkout{Units: &user.PreferredUnits}
	if err := c.Bind(d); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	if err := d.Update(workout); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	if d.EquipmentIDs != nil {
		equipment, err := wc.equipmentRepo.GetByUserIDs(user.ID, d.EquipmentIDs)
		if err != nil {
			return renderApiError(c, http.StatusBadRequest, err)
		}
		if err := wc.db.Model(&workout).Association("Equipment").Replace(equipment); err != nil {
			return renderApiError(c, http.StatusInternalServerError, err)
		}
	}

	if err := workout.Save(wc.db); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if err := model.PreloadWorkoutDetails(wc.db).Preload("Equipment").First(&workout, workout.ID).Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if err := worker.EnqueueWorkoutUpdate(c.Request().Context(), wc.client, workout.ID); err != nil {
		wc.logger.Error("Failed to enqueue workout update", "workout_id", workout.ID, "error", err)
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
// @Success      200  {object}  dto.Response[dto.WorkoutResponse]
// @Failure      404  {object}  dto.Response[any]
// @Failure      403  {object}  dto.Response[any]
// @Router       /workouts/{id}/toggle-lock [post]
func (wc *workoutController) ToggleWorkoutLock(c echo.Context) error {
	user := currentUser(c)

	workout, err := wc.getOwnedWorkout(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	if workout.ProfileID != user.Profile.ID {
		return renderApiError(c, http.StatusForbidden, errors.New("not authorized"))
	}

	workout.Locked = !workout.Locked

	if err := workout.Save(wc.db); err != nil {
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

	if err := workout.Save(wc.db); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if err := worker.EnqueueWorkoutUpdate(c.Request().Context(), wc.client, workout.ID); err != nil {
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
// @Failure      404  {object}  dto.Response[any]
// @Router       /workouts/{id}/attachments/{attachment_id} [get]
func (wc *workoutController) DownloadWorkoutAttachment(c echo.Context) error {
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
