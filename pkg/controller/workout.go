package controller

import (
	"archive/zip"
	"bytes"
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
	"github.com/AepyornisNet/aepyornis/pkg/notification"
	"github.com/AepyornisNet/aepyornis/pkg/repository"
	"github.com/AepyornisNet/aepyornis/pkg/service"
	"github.com/AepyornisNet/aepyornis/pkg/worker"
	"github.com/labstack/echo/v5"
	"github.com/samber/do/v2"
	"github.com/spf13/cast"
	"github.com/vgarvardt/gue/v6"
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
	notify               service.NotificationService
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
		notify:               do.MustInvoke[service.NotificationService](injector),
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

func applyLikeMetadata(results []dto.WorkoutResponse, counts map[uint64]int64, liked map[uint64]bool, recentLikes map[uint64][]model.APStatusLike) {
	for i := range results {
		results[i].LikesCount = counts[results[i].ID]
		results[i].LikedByMe = liked[results[i].ID]
		if likes, ok := recentLikes[results[i].ID]; ok && len(likes) > 0 {
			items := make([]dto.WorkoutLikeResponse, 0, len(likes))
			for _, l := range likes {
				items = append(items, dto.NewWorkoutLikeResponse(&l))
			}
			results[i].RecentLikes = items
		}
	}
}

func applyReplyMetadata(results []dto.WorkoutResponse, counts map[uint64]int64) {
	for i := range results {
		results[i].RepliesCount = counts[results[i].ID]
	}
}

func (wc *workoutController) getOwnedWorkout(c *echo.Context) (*model.Workout, error) {
	id, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return nil, err
	}

	user := currentUser(c)
	w, err := wc.workoutRepo.GetByProfileID(user.Profile.ID, id)
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
	return model.CanReadWorkout(wc.db, requester, workout)
}

func (wc *workoutController) getReadableWorkout(c *echo.Context, withDetails bool) (*model.Workout, error) {
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
// @Failure      400  {object}  dto.Response[string]
// @Failure      500  {object}  dto.Response[string]
// @Router       /workouts [get]
func (wc *workoutController) GetWorkouts(c *echo.Context) error {
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

	totalCount, err := wc.workoutRepo.CountByProfileAndFilters(user.Profile.ID, filters)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	workouts, err := wc.workoutRepo.ListByProfileAndFilters(user.Profile.ID, filters, pagination.PerPage, pagination.GetOffset())
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	results := dto.NewWorkoutsResponse(workouts)
	published, err := wc.apOutboxRepo.PublishedMap(user.Profile.ID, workoutIDs(workouts))
	if err == nil {
		applyPublishedFlags(results, published)
	}

	counts, err := wc.workoutLikeRepo.CountMapByWorkoutIDs(workoutIDs(workouts))
	if err == nil {
		liked, likedErr := wc.workoutLikeRepo.LikedMapByProfile(workoutIDs(workouts), user.Profile.ID)
		recentLikes, _ := wc.workoutLikeRepo.RecentLikesMapByWorkoutIDs(workoutIDs(workouts), 3)
		if likedErr == nil {
			applyLikeMetadata(results, counts, liked, recentLikes)
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
// @Failure      400  {object}  dto.Response[string]
// @Failure      404  {object}  dto.Response[string]
// @Router       /workouts/{id} [get]
func (wc *workoutController) GetWorkout(c *echo.Context) error {
	workout, err := wc.getReadableWorkout(c, true)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	records, err := model.GetWorkoutIntervalRecordsWithRank(wc.db, workout.ProfileID, workout.Type, workout.ID)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	user := currentUser(c)
	if len(workout.RouteSegmentMatches) > 0 {
		visibleMatches := make([]*model.RouteSegmentMatch, 0, len(workout.RouteSegmentMatches))
		for _, m := range workout.RouteSegmentMatches {
			if m != nil && m.RouteSegment != nil {
				canRead, _ := model.CanReadRouteSegment(wc.db, user, m.RouteSegment)
				if canRead {
					visibleMatches = append(visibleMatches, m)
				}
			}
		}
		workout.RouteSegmentMatches = visibleMatches
	}

	result := dto.NewWorkoutDetailResponse(workout, records)
	ownerUserID := workoutOwnerUserID(workout)
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
// @Failure      400  {object}  dto.Response[string]
// @Failure      404  {object}  dto.Response[string]
// @Router       /workouts/{id}/likes [get]
func (wc *workoutController) GetWorkoutLikes(c *echo.Context) error {
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

	if workout.ProfileID == viewer.Profile.ID {
		return renderApiError(c, http.StatusBadRequest, errors.New("cannot like your own workout"))
	}

	if err := wc.workoutLikeRepo.LikeByProfile(workout.ID, viewer.Profile.ID); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if ownerID := workoutOwnerUserID(workout); ownerID != 0 && ownerID != viewer.ID {
		var owner model.User
		if err := wc.db.First(&owner, ownerID).Error; err == nil {
			_ = wc.notify.Send(c.Request().Context(), &owner, notification.NewWorkoutLike(viewer.Profile.DisplayName, workout.ID))
		}
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

func (wc *workoutController) likeLocalWorkout(c *echo.Context, viewer *model.User, localWorkoutID uint64) (map[string]any, int, error) {
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

	if ownerID := workoutOwnerUserID(workout); ownerID != 0 && ownerID != viewer.ID {
		var owner model.User
		if err := wc.db.First(&owner, ownerID).Error; err == nil {
			_ = wc.notify.Send(c.Request().Context(), &owner, notification.NewWorkoutLike(viewer.Profile.DisplayName, workout.ID))
		}
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
// @Failure      400  {object}  dto.Response[string]
// @Failure      404  {object}  dto.Response[string]
// @Router       /workouts/{id}/replies [post]
func (wc *workoutController) CreateReply(c *echo.Context) error {
	viewer := currentUser(c)

	workout, err := wc.getReadableWorkout(c, false)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	var params dto.WorkoutReplyCreateRequest
	if err := c.Bind(&params); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	if err := c.Validate(&params); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	params.Content = strings.TrimSpace(params.Content)

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

	if ownerID := workoutOwnerUserID(workout); ownerID != 0 && ownerID != viewer.ID {
		var owner model.User
		if err := wc.db.First(&owner, ownerID).Error; err == nil {
			_ = wc.notify.Send(c.Request().Context(), &owner, notification.NewWorkoutReply(viewer.Profile.DisplayName, workout.ID))
		}
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
// @Failure      400  {object}  dto.Response[string]
// @Failure      404  {object}  dto.Response[string]
// @Router       /workouts/{id}/breakdown [get]
func (wc *workoutController) GetWorkoutBreakdown(c *echo.Context) error {
	requester := currentUser(c)

	params := dto.WorkoutBreakdownRequest{
		Count: 1.0,
		Mode:  "auto",
	}

	if err := c.Bind(&params); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	if err := c.Validate(&params); err != nil {
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
			Items: dto.NewWorkoutBreakdownItemsFromLaps(workout.Laps, workout.Records),
		}

		return c.JSON(http.StatusOK, resp)
	}

	if len(workout.Records) == 0 {
		return renderApiError(c, http.StatusBadRequest, errors.New("workout has no records"))
	}

	breakdown, err := workout.StatisticsPer(params.Count, requester.PreferredUnits.Distance())
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	resp.Results = dto.WorkoutBreakdownResponse{
		Mode:  "unit",
		Items: dto.NewWorkoutBreakdownItemsFromUnit(breakdown.Items, params.Count),
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

	if len(workout.Records) == 0 {
		return renderApiError(c, http.StatusBadRequest, errors.New("workout has no records"))
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
		Results: dto.NewWorkoutRangeStatsResponse(stats, startIdx, endIdx),
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
// @Failure      400  {object}  dto.Response[string]
// @Failure      500  {object}  dto.Response[string]
// @Router       /workouts/calendar [get]
func (wc *workoutController) GetWorkoutCalendar(c *echo.Context) error {
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

func (wc *workoutController) createWorkoutFromFile(c *echo.Context, user *model.User) error { //nolint:gocyclo
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
	if workoutType == "" {
		workoutType = model.WorkoutTypeAutoDetect
	}
	visibilityRaw := multipartFormValue(form, c, "visibility")
	hasVisibility := hasMultipartFormValue(form, c, "visibility")

	names := form.Value["name"]
	if len(names) == 0 {
		names = form.Value["names"]
	}

	createdWorkouts := []dto.WorkoutResponse{}
	errList := []error{}

	for i, file := range files {
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

		var customName string
		if i < len(names) {
			customName = strings.TrimSpace(names[i])
		}
		if customName == "" && len(names) == 1 && len(files) == 1 {
			customName = strings.TrimSpace(names[0])
		}

		for _, w := range ws {
			needsSave := false
			if customName != "" {
				w.Name = customName
				needsSave = true
			}
			if hasVisibility {
				if visibilityRaw == "private" {
					w.Visibility = model.WorkoutVisibilityPrivate
					needsSave = true
				} else if vis := model.WorkoutVisibility(visibilityRaw); vis.IsValid() {
					w.Visibility = vis
					needsSave = true
				}
			}
			if needsSave {
				_ = w.Save(wc.db)
			}
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
		resp.AddContextError(c.Request().Context(), errList...)

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

func (wc *workoutController) createWorkoutManual(c *echo.Context, user *model.User) error {
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
// @Failure      500  {object}  dto.Response[string]
// @Router       /workouts/recent [get]
//

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
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}
	if req.Offset < 0 {
		req.Offset = 0
	}
	limit := req.Limit
	offset := req.Offset
	handle := strings.TrimSpace(req.Handle)

	var workouts []*model.Workout
	query := wc.db.
		Scopes(model.PreloadWorkoutData).
		Preload("Profile").
		Preload("Profile.User")

	if handle != "" {
		host := wc.cfg.Host
		if host == "" {
			host = c.Request().Host
		}
		targetUser, err := wc.userRepo.GetByHandle(handle, host)
		if err != nil {
			return renderApiError(c, http.StatusNotFound, err)
		}
		query = model.ScopeVisibleWorkouts(query, targetUser.Profile.ID, requester.Profile.ID)
	} else {
		scope := req.Scope
		if scope == "" {
			scope = "following"
		}

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
		recentLikes, _ := wc.workoutLikeRepo.RecentLikesMapByWorkoutIDs(workoutIDs(workouts), 3)
		if likedErr == nil {
			applyLikeMetadata(results, counts, liked, recentLikes)
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
// @Failure      404  {object}  dto.Response[string]
// @Failure      500  {object}  dto.Response[string]
// @Router       /workouts/{id} [delete]
func (wc *workoutController) DeleteWorkout(c *echo.Context) error {
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
// @Failure      404  {object}  dto.Response[string]
// @Failure      403  {object}  dto.Response[string]
// @Router       /workouts/{id}/toggle-lock [post]
func (wc *workoutController) ToggleWorkoutLock(c *echo.Context) error {
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
// @Failure      404  {object}  dto.Response[string]
// @Router       /workouts/{id}/refresh [post]
func (wc *workoutController) RefreshWorkout(c *echo.Context) error {
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

	var types []string
	if err := wc.db.Model(&model.Workout{}).
		Where("profile_id = ?", user.Profile.ID).
		Where("type IS NOT NULL AND type != ''").
		Order("type").
		Distinct("type").
		Pluck("type", &types).Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	type typeSubPair struct {
		Type    string `gorm:"column:type"`
		SubType string `gorm:"column:sub_type"`
	}
	var pairs []typeSubPair
	if err := wc.db.Model(&model.Workout{}).
		Select("type, sub_type").
		Where("profile_id = ?", user.Profile.ID).
		Where("type IS NOT NULL AND type != ''").
		Where("sub_type IS NOT NULL AND sub_type != ''").
		Group("type, sub_type").
		Order("type, sub_type").
		Find(&pairs).Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	subTypesByType := make(map[string][]string)
	for _, p := range pairs {
		subTypesByType[p.Type] = append(subTypesByType[p.Type], p.SubType)
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

	var workouts []model.Workout
	if err := wc.db.Preload("File").Where("profile_id = ?", user.Profile.ID).Find(&workouts, req.IDs).Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if len(workouts) == 0 {
		return renderApiError(c, http.StatusNotFound, errors.New("no workouts found"))
	}

	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	for _, w := range workouts {
		if !w.HasFile() {
			continue
		}

		filename := w.File.Filename
		if filename == "" {
			filename = "workout_" + strconv.FormatUint(w.ID, 10) + ".gpx"
		}

		f, err := zipWriter.Create(filename)
		if err != nil {
			return renderApiError(c, http.StatusInternalServerError, err)
		}

		if _, err := f.Write(w.File.Content); err != nil {
			return renderApiError(c, http.StatusInternalServerError, err)
		}
	}

	if err := zipWriter.Close(); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename=\"workouts.zip\"")
	return c.Blob(http.StatusOK, "application/zip", buf.Bytes())
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

	// Fetch equipment
	equipment, err := wc.equipmentRepo.GetByUserIDs(user.Profile.ID, req.EquipmentIDs)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	if len(equipment) == 0 {
		return renderApiError(c, http.StatusBadRequest, errors.New("no valid equipment found"))
	}

	// Fetch workouts
	var workouts []model.Workout
	if err := wc.db.Where("profile_id = ?", user.Profile.ID).Find(&workouts, req.WorkoutIDs).Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	// Link each equipment to each workout
	tx := wc.db.Begin()
	for _, w := range workouts {
		if err := tx.Model(&w).Association("Equipment").Append(equipment); err != nil {
			tx.Rollback()
			return renderApiError(c, http.StatusInternalServerError, err)
		}
	}
	if err := tx.Commit().Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[map[string]string]{
		Results: map[string]string{"message": "Equipment added to workouts successfully"},
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
