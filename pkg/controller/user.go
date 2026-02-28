package controller

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	vocab "github.com/go-ap/activitypub"
	ap "github.com/jovandeginste/workout-tracker/v2/pkg/activitypub"
	"github.com/jovandeginste/workout-tracker/v2/pkg/container"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model/dto"
	"github.com/labstack/echo/v4"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

type UserController interface {
	GetWhoami(c echo.Context) error
	GetUserProfileByHandle(c echo.Context) error
	FollowUserByHandle(c echo.Context) error
	UnfollowUserByHandle(c echo.Context) error
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
// @Success      200  {object}  dto.Response[dto.UserProfileResponse]
// @Failure      401  {object}  dto.Response[any]
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
// @Success      200  {object}  dto.Response[dto.TotalsResponse]
// @Failure      500  {object}  dto.Response[any]
// @Router       /totals [get]
func (uc *userController) GetTotals(c echo.Context) error {
	targetUser, viewer, viewerActorIRI, err := uc.resolveTargetUserFromHandle(c)
	if err != nil {
		if errors.Is(err, dto.ErrNotAuthorized) {
			return renderApiError(c, http.StatusForbidden, err)
		}

		return renderApiError(c, http.StatusNotFound, err)
	}

	if targetUser == nil || viewer == nil {
		return renderApiError(c, http.StatusForbidden, dto.ErrNotAuthorized)
	}

	startDate, endDate, err := parseDateRange(c)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	totalsQuery := model.ScopeVisibleWorkouts(
		uc.context.GetDB().Table("workouts").
			Select(
				"count(*) as workouts",
				"max(workouts.type) as workout_type",
				"sum(total_duration) as duration",
				"sum(total_distance) as distance",
				"sum(total_up) as up",
				"'all' as bucket",
			).
			Joins("join map_data on workouts.id = map_data.workout_id"),
		targetUser.ID,
		viewer.ID,
		viewerActorIRI,
	)

	totalsShow := targetUser.Profile.TotalsShow

	if totalsShow == model.WorkoutTypeAutoDetect {
		totalsShow = model.WorkoutTypeRunning
	}

	totalsQuery = totalsQuery.Where("workouts.type = ?", totalsShow)

	if startDate != nil {
		totalsQuery = totalsQuery.Where("workouts.date >= ?", *startDate)
	}

	if endDate != nil {
		totalsQuery = totalsQuery.Where("workouts.date <= ?", *endDate)
	}

	totals := &model.Bucket{}
	err = totalsQuery.Scan(totals).Error
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
// @Success      200  {object}  dto.Response[[]dto.WorkoutRecordResponse]
// @Failure      500  {object}  dto.Response[any]
// @Router       /records [get]
func (uc *userController) GetRecords(c echo.Context) error {
	targetUser, viewer, viewerActorIRI, err := uc.resolveTargetUserFromHandle(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	startDate, endDate, err := parseDateRange(c)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	records, err := uc.getVisibleRecords(targetUser, viewer, viewerActorIRI, startDate, endDate)
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
// @Success      200  {object}  dto.PaginatedResponse[dto.DistanceRecordResponse]
// @Failure      400  {object}  dto.Response[any]
// @Failure      500  {object}  dto.Response[any]
// @Router       /records/ranking [get]
func (uc *userController) GetRecordsRanking(c echo.Context) error {
	targetUser, viewer, viewerActorIRI, err := uc.resolveTargetUserFromHandle(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

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

	records, totalCount, err := uc.getVisibleDistanceRanking(targetUser, viewer, viewerActorIRI, wt, label, startDate, endDate, pagination.PerPage, pagination.GetOffset())
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
// @Success      200  {object}  dto.PaginatedResponse[dto.ClimbRecordResponse]
// @Failure      400  {object}  dto.Response[any]
// @Failure      500  {object}  dto.Response[any]
// @Router       /records/climbs/ranking [get]
func (uc *userController) GetClimbRecordsRanking(c echo.Context) error {
	targetUser, viewer, viewerActorIRI, err := uc.resolveTargetUserFromHandle(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

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

	records, totalCount, err := uc.getVisibleClimbRanking(targetUser, viewer, viewerActorIRI, wt, startDate, endDate, pagination.PerPage, pagination.GetOffset())
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
// @Success      200  {object}  dto.Response[[]dto.WorkoutRecordResponse]
// @Failure      403  {object}  dto.Response[any]
// @Failure      404  {object}  dto.Response[any]
// @Router       /{id} [get]
// TODO: Add more data. This will be used for public profiles.
func (uc *userController) GetUserByID(c echo.Context) error {
	id, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	u, err := uc.context.UserRepo().GetByID(id)
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

// GetUserProfileByHandle returns profile header data and social stats for an ActivityPub-enabled user
// @Summary      Get user profile by ActivityPub handle
// @Tags         user
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        handle  query     string  false  "ActivityPub handle"
// @Produce      json
// @Success      200  {object}  dto.Response[dto.ActivityPubProfileSummaryResponse]
// @Failure      404  {object}  dto.Response[any]
// @Router       /user-profile [get]
func (uc *userController) GetUserProfileByHandle(c echo.Context) error {
	handle := strings.TrimSpace(c.QueryParam("handle"))
	if handle != "" {
		username, host, parsedAsRemote, err := uc.parseHandleWithHost(c, handle)
		if err != nil {
			return renderApiError(c, http.StatusNotFound, err)
		}

		if parsedAsRemote {
			return uc.getRemoteProfileSummary(c, username, host)
		}
	}

	targetUser, viewer, viewerActorIRI, err := uc.resolveTargetUserFromHandle(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	if viewer.ID != targetUser.ID && !targetUser.ActivityPubEnabled() {
		return renderApiError(c, http.StatusNotFound, gorm.ErrRecordNotFound)
	}

	postsQuery := model.ScopeVisibleWorkouts(
		uc.context.GetDB().Model(&model.Workout{}),
		targetUser.ID,
		viewer.ID,
		viewerActorIRI,
	)

	var postsCount int64
	if err := postsQuery.Count(&postsCount).Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	followersCount, err := uc.context.FollowerRepo().CountApprovedFollowers(targetUser.ID)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	targetActorIRI := uc.localActorIRI(c, targetUser)
	followingCount, err := uc.context.FollowerRepo().CountApprovedFollowingByActorIRI(targetActorIRI)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	isFollowing := false
	if viewer.ID != targetUser.ID {
		isFollowing, err = uc.context.FollowerRepo().IsFollowingApprovedByActorIRI(viewer.ID, targetActorIRI)
		if err != nil {
			return renderApiError(c, http.StatusInternalServerError, err)
		}
	}

	host := uc.context.GetConfig().Host
	if host == "" {
		host = c.Request().Host
	}

	memberSince := targetUser.CreatedAt.UTC()

	resp := dto.Response[dto.ActivityPubProfileSummaryResponse]{
		Results: dto.ActivityPubProfileSummaryResponse{
			ID:             targetUser.ID,
			Username:       targetUser.Username,
			Name:           targetUser.Name,
			Handle:         fmt.Sprintf("@%s@%s", targetUser.Username, host),
			ActorURL:       targetActorIRI,
			IconURL:        "",
			IsExternal:     false,
			IsOwn:          viewer.ID == targetUser.ID,
			IsFollowing:    isFollowing,
			PostsCount:     postsCount,
			FollowersCount: followersCount,
			FollowingCount: followingCount,
			MemberSince:    memberSince,
		},
	}

	return c.JSON(http.StatusOK, resp)
}

// FollowUserByHandle follows an ActivityPub-enabled local user
// @Summary      Follow user by ActivityPub handle
// @Tags         user
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        handle  query     string  true  "ActivityPub handle"
// @Produce      json
// @Success      200  {object}  dto.Response[dto.ActivityPubProfileSummaryResponse]
// @Failure      400  {object}  dto.Response[any]
// @Failure      404  {object}  dto.Response[any]
// @Router       /user-profile/follow [post]
func (uc *userController) FollowUserByHandle(c echo.Context) error {
	if handle := strings.TrimSpace(c.QueryParam("handle")); handle != "" {
		if _, _, parsedAsRemote, err := uc.parseHandleWithHost(c, handle); err == nil && parsedAsRemote {
			return uc.followRemoteUserByHandle(c, handle)
		}
	}

	targetUser, viewer, viewerActorIRI, err := uc.resolveTargetUserFromHandle(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	if viewer.ID == targetUser.ID {
		return renderApiError(c, http.StatusBadRequest, errors.New("cannot follow yourself"))
	}

	if !viewer.ActivityPubEnabled() {
		return renderApiError(c, http.StatusBadRequest, errors.New("activitypub must be enabled to follow users"))
	}

	if !targetUser.ActivityPubEnabled() {
		return renderApiError(c, http.StatusBadRequest, errors.New("target user does not have activitypub enabled"))
	}

	targetActorIRI := uc.localActorIRI(c, targetUser)
	following, err := uc.context.FollowerRepo().UpsertFollowingRequest(viewer.ID, targetActorIRI, targetActorIRI+"/inbox")
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if _, err := uc.context.FollowerRepo().UpsertFollowerRequest(targetUser.ID, viewerActorIRI, viewerActorIRI+"/inbox"); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	isFollowing := following.Approved

	followersCount, err := uc.context.FollowerRepo().CountApprovedFollowers(targetUser.ID)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	followingCount, err := uc.context.FollowerRepo().CountApprovedFollowingByActorIRI(targetActorIRI)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[dto.ActivityPubProfileSummaryResponse]{
		Results: dto.ActivityPubProfileSummaryResponse{
			ID:             targetUser.ID,
			Username:       targetUser.Username,
			Name:           targetUser.Name,
			Handle:         uc.renderHandle(c, targetUser.Username),
			ActorURL:       targetActorIRI,
			IconURL:        "",
			IsExternal:     false,
			IsOwn:          false,
			IsFollowing:    isFollowing,
			FollowersCount: followersCount,
			FollowingCount: followingCount,
			MemberSince:    targetUser.CreatedAt,
		},
	}

	return c.JSON(http.StatusOK, resp)
}

// UnfollowUserByHandle unfollows an ActivityPub-enabled local user
// @Summary      Unfollow user by ActivityPub handle
// @Tags         user
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        handle  query     string  true  "ActivityPub handle"
// @Produce      json
// @Success      200  {object}  dto.Response[dto.ActivityPubProfileSummaryResponse]
// @Failure      400  {object}  dto.Response[any]
// @Failure      404  {object}  dto.Response[any]
// @Router       /user-profile/unfollow [post]
func (uc *userController) UnfollowUserByHandle(c echo.Context) error {
	if handle := strings.TrimSpace(c.QueryParam("handle")); handle != "" {
		if _, _, parsedAsRemote, err := uc.parseHandleWithHost(c, handle); err == nil && parsedAsRemote {
			return uc.unfollowRemoteUserByHandle(c, handle)
		}
	}

	targetUser, viewer, viewerActorIRI, err := uc.resolveTargetUserFromHandle(c)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	if viewer.ID == targetUser.ID {
		return renderApiError(c, http.StatusBadRequest, errors.New("cannot unfollow yourself"))
	}

	if err := uc.context.FollowerRepo().DeleteFollowerByActorIRI(targetUser.ID, viewerActorIRI); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	targetActorIRI := uc.localActorIRI(c, targetUser)
	if err := uc.context.FollowerRepo().DeleteFollowingByActorIRI(viewer.ID, targetActorIRI); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}
	followersCount, err := uc.context.FollowerRepo().CountApprovedFollowers(targetUser.ID)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	followingCount, err := uc.context.FollowerRepo().CountApprovedFollowingByActorIRI(targetActorIRI)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[dto.ActivityPubProfileSummaryResponse]{
		Results: dto.ActivityPubProfileSummaryResponse{
			ID:             targetUser.ID,
			Username:       targetUser.Username,
			Name:           targetUser.Name,
			Handle:         uc.renderHandle(c, targetUser.Username),
			ActorURL:       targetActorIRI,
			IconURL:        "",
			IsExternal:     false,
			IsOwn:          false,
			IsFollowing:    false,
			FollowersCount: followersCount,
			FollowingCount: followingCount,
			MemberSince:    targetUser.CreatedAt,
		},
	}

	return c.JSON(http.StatusOK, resp)
}

func (uc *userController) resolveTargetUserFromHandle(c echo.Context) (*model.User, *model.User, string, error) {
	viewer := uc.context.GetUser(c)
	if viewer == nil {
		viewer = model.AnonymousUser()
	}

	handle := strings.TrimSpace(c.QueryParam("handle"))
	if handle == "" {
		if viewer.IsAnonymous() {
			return nil, nil, "", dto.ErrNotAuthorized
		}

		return viewer, viewer, uc.localActorIRI(c, viewer), nil
	}

	normalizedUsername, err := uc.parseLocalHandle(c, handle)
	if err != nil {
		return nil, nil, "", err
	}

	targetUser, err := uc.context.UserRepo().GetByUsername(normalizedUsername)
	if err != nil {
		return nil, nil, "", err
	}

	if targetUser == nil {
		return nil, nil, "", gorm.ErrRecordNotFound
	}

	return targetUser, viewer, uc.localActorIRI(c, viewer), nil
}

func (uc *userController) parseLocalHandle(c echo.Context, handle string) (string, error) {
	h := strings.TrimSpace(handle)
	h = strings.TrimPrefix(h, "@")

	if parsedURL, err := url.Parse(h); err == nil && parsedURL.Host != "" {
		segments := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
		if len(segments) == 3 && segments[0] == "ap" && segments[1] == "users" && segments[2] != "" {
			if uc.isLocalHost(c, parsedURL.Host) {
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

		if !uc.isLocalHost(c, parts[1]) {
			return "", gorm.ErrRecordNotFound
		}

		return parts[0], nil
	}

	if h == "" {
		return "", gorm.ErrRecordNotFound
	}

	return h, nil
}

func (uc *userController) parseHandleWithHost(c echo.Context, handle string) (string, string, bool, error) {
	h := strings.TrimSpace(strings.TrimPrefix(handle, "@"))
	if h == "" {
		return "", "", false, gorm.ErrRecordNotFound
	}

	if parsedURL, err := url.Parse(h); err == nil && parsedURL.Host != "" {
		segments := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
		if len(segments) == 3 && segments[0] == "ap" && segments[1] == "users" && segments[2] != "" {
			isRemote := !uc.isLocalHost(c, parsedURL.Host)
			return segments[2], parsedURL.Host, isRemote, nil
		}
	}

	if strings.Contains(h, "@") {
		parts := strings.SplitN(h, "@", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", false, gorm.ErrRecordNotFound
		}

		isRemote := !uc.isLocalHost(c, parts[1])
		return parts[0], parts[1], isRemote, nil
	}

	return h, "", false, nil
}

func (uc *userController) getRemoteProfileSummary(c echo.Context, username, host string) error {
	actorIRI, err := ap.ResolveActorIRIFromWebFinger(c.Request().Context(), username, host)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	actor, err := ap.LoadRemoteActor(c.Request().Context(), actorIRI)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	actorURL := actorIRI
	if actor != nil && actor.ID != "" {
		actorURL = actor.ID.String()
	}

	followersCount, _ := ap.LoadCollectionTotalItems(c.Request().Context(), itemIRIString(actor.Followers))
	followingCount, _ := ap.LoadCollectionTotalItems(c.Request().Context(), itemIRIString(actor.Following))
	postsCount, _ := ap.LoadCollectionTotalItems(c.Request().Context(), itemIRIString(actor.Outbox))

	name := username
	if actor != nil && actor.Name.String() != "" {
		name = actor.Name.String()
	}

	memberSince := time.Time{}
	if actor != nil {
		memberSince = actor.Published
	}

	iconURL := actorIconURL(actor)

	viewer := uc.context.GetUser(c)
	isFollowing := false
	if viewer != nil && !viewer.IsAnonymous() {
		isFollowing, _ = uc.context.FollowerRepo().IsFollowingActiveByActorIRI(viewer.ID, actorURL)
	}

	resp := dto.Response[dto.ActivityPubProfileSummaryResponse]{
		Results: dto.ActivityPubProfileSummaryResponse{
			ID:             0,
			Username:       username,
			Name:           name,
			Handle:         fmt.Sprintf("@%s@%s", username, host),
			ActorURL:       actorURL,
			IconURL:        iconURL,
			IsExternal:     true,
			IsOwn:          false,
			IsFollowing:    isFollowing,
			PostsCount:     postsCount,
			FollowersCount: followersCount,
			FollowingCount: followingCount,
			MemberSince:    memberSince,
		},
	}

	return c.JSON(http.StatusOK, resp)
}

func (uc *userController) followRemoteUserByHandle(c echo.Context, handle string) error {
	viewer := uc.context.GetUser(c)
	if viewer == nil || viewer.IsAnonymous() {
		return renderApiError(c, http.StatusForbidden, dto.ErrNotAuthorized)
	}

	if !viewer.ActivityPubEnabled() {
		return renderApiError(c, http.StatusBadRequest, errors.New("activitypub must be enabled to follow users"))
	}

	username, host, _, err := uc.parseHandleWithHost(c, handle)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	actorIRI, err := ap.ResolveActorIRIFromWebFinger(c.Request().Context(), username, host)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	actor, err := ap.LoadRemoteActor(c.Request().Context(), actorIRI)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	inbox := itemIRIString(actor.Inbox)
	if inbox == "" {
		return renderApiError(c, http.StatusBadRequest, errors.New("remote actor inbox not found"))
	}

	viewerActorIRI := uc.localActorIRI(c, viewer)
	if actorIRI == viewerActorIRI {
		return renderApiError(c, http.StatusBadRequest, errors.New("cannot follow yourself"))
	}

	if _, err := uc.context.FollowerRepo().UpsertFollowingRequest(viewer.ID, actorIRI, inbox); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if err := ap.SendFollow(c.Request().Context(), viewerActorIRI, viewer.PrivateKey, inbox, actorIRI); err != nil {
		return renderApiError(c, http.StatusBadGateway, err)
	}

	followersCount, _ := ap.LoadCollectionTotalItems(c.Request().Context(), itemIRIString(actor.Followers))
	followingCount, _ := ap.LoadCollectionTotalItems(c.Request().Context(), itemIRIString(actor.Following))
	postsCount, _ := ap.LoadCollectionTotalItems(c.Request().Context(), itemIRIString(actor.Outbox))

	name := username
	if actor != nil && actor.Name.String() != "" {
		name = actor.Name.String()
	}

	memberSince := time.Time{}
	if actor != nil {
		memberSince = actor.Published
	}

	iconURL := actorIconURL(actor)

	resp := dto.Response[dto.ActivityPubProfileSummaryResponse]{
		Results: dto.ActivityPubProfileSummaryResponse{
			ID:             0,
			Username:       username,
			Name:           name,
			Handle:         fmt.Sprintf("@%s@%s", username, host),
			ActorURL:       actorIRI,
			IconURL:        iconURL,
			IsExternal:     true,
			IsOwn:          false,
			IsFollowing:    true,
			PostsCount:     postsCount,
			FollowersCount: followersCount,
			FollowingCount: followingCount,
			MemberSince:    memberSince,
		},
	}

	return c.JSON(http.StatusOK, resp)
}

func (uc *userController) unfollowRemoteUserByHandle(c echo.Context, handle string) error {
	viewer := uc.context.GetUser(c)
	if viewer == nil || viewer.IsAnonymous() {
		return renderApiError(c, http.StatusForbidden, dto.ErrNotAuthorized)
	}

	if !viewer.ActivityPubEnabled() {
		return renderApiError(c, http.StatusBadRequest, errors.New("activitypub must be enabled to unfollow users"))
	}

	username, host, _, err := uc.parseHandleWithHost(c, handle)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	actorIRI, err := ap.ResolveActorIRIFromWebFinger(c.Request().Context(), username, host)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	actor, err := ap.LoadRemoteActor(c.Request().Context(), actorIRI)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	inbox := itemIRIString(actor.Inbox)
	if inbox == "" {
		return renderApiError(c, http.StatusBadRequest, errors.New("remote actor inbox not found"))
	}

	viewerActorIRI := uc.localActorIRI(c, viewer)
	if err := ap.SendUndoFollow(c.Request().Context(), viewerActorIRI, viewer.PrivateKey, inbox, actorIRI); err != nil {
		return renderApiError(c, http.StatusBadGateway, err)
	}

	if err := uc.context.FollowerRepo().DeleteFollowingByActorIRI(viewer.ID, actorIRI); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	followersCount, _ := ap.LoadCollectionTotalItems(c.Request().Context(), itemIRIString(actor.Followers))
	followingCount, _ := ap.LoadCollectionTotalItems(c.Request().Context(), itemIRIString(actor.Following))
	postsCount, _ := ap.LoadCollectionTotalItems(c.Request().Context(), itemIRIString(actor.Outbox))

	name := username
	if actor != nil && actor.Name.String() != "" {
		name = actor.Name.String()
	}

	memberSince := time.Time{}
	if actor != nil {
		memberSince = actor.Published
	}

	iconURL := actorIconURL(actor)

	resp := dto.Response[dto.ActivityPubProfileSummaryResponse]{
		Results: dto.ActivityPubProfileSummaryResponse{
			ID:             0,
			Username:       username,
			Name:           name,
			Handle:         fmt.Sprintf("@%s@%s", username, host),
			ActorURL:       actorIRI,
			IconURL:        iconURL,
			IsExternal:     true,
			IsOwn:          false,
			IsFollowing:    false,
			PostsCount:     postsCount,
			FollowersCount: followersCount,
			FollowingCount: followingCount,
			MemberSince:    memberSince,
		},
	}

	return c.JSON(http.StatusOK, resp)
}

func (uc *userController) isLocalHost(c echo.Context, host string) bool {
	configuredHost := uc.context.GetConfig().Host
	if configuredHost == "" {
		configuredHost = c.Request().Host
	}

	return strings.EqualFold(strings.TrimSpace(host), strings.TrimSpace(configuredHost))
}

func itemIRIString(it vocab.Item) string {
	if vocab.IsNil(it) {
		return ""
	}

	if vocab.IsIRI(it) {
		return it.GetLink().String()
	}

	var iri string
	_ = vocab.OnLink(it, func(link *vocab.Link) error {
		iri = link.Href.String()
		return nil
	})

	return iri
}

func actorIconURL(actor *vocab.Actor) string {
	if actor == nil || vocab.IsNil(actor.Icon) {
		return ""
	}

	if vocab.IsIRI(actor.Icon) {
		return actor.Icon.GetLink().String()
	}

	iconURL := itemIRIString(actor.Icon)
	if iconURL != "" {
		return iconURL
	}

	_ = vocab.OnObject(actor.Icon, func(object *vocab.Object) error {
		if object != nil && !vocab.IsNil(object.URL) {
			iconURL = itemIRIString(object.URL)
		}
		return nil
	})

	return iconURL
}

func (uc *userController) localActorIRI(c echo.Context, user *model.User) string {
	if user == nil {
		return ""
	}

	return ap.LocalActorURL(ap.LocalActorURLConfig{
		Host:           uc.context.GetConfig().Host,
		WebRoot:        uc.context.GetConfig().WebRoot,
		FallbackHost:   c.Request().Host,
		FallbackScheme: c.Scheme(),
	}, user.Username)
}

func (uc *userController) renderHandle(c echo.Context, username string) string {
	host := uc.context.GetConfig().Host
	if host == "" {
		host = c.Request().Host
	}

	return fmt.Sprintf("@%s@%s", username, host)
}

func (uc *userController) getVisibleRecords(targetUser, viewer *model.User, viewerActorIRI string, startDate, endDate *time.Time) ([]*model.WorkoutRecord, error) {
	if targetUser.IsAnonymous() {
		return nil, model.ErrAnonymousUser
	}

	rs := []*model.WorkoutRecord{}

	for _, w := range model.DistanceWorkoutTypes() {
		r, err := uc.getVisibleRecordForType(targetUser, viewer, viewerActorIRI, w, startDate, endDate)
		if err != nil {
			return nil, err
		}

		if r != nil {
			rs = append(rs, r)
		}
	}

	return rs, nil
}

func (uc *userController) getVisibleRecordForType(targetUser, viewer *model.User, viewerActorIRI string, t model.WorkoutType, startDate, endDate *time.Time) (*model.WorkoutRecord, error) {
	if t == "" {
		t = model.WorkoutTypeRunning
		if targetUser != nil {
			t = targetUser.Profile.TotalsShow
		}
	}

	r := &model.WorkoutRecord{WorkoutType: t}

	mapping := map[*model.Float64Record]string{
		&r.Distance:            "max(total_distance)",
		&r.MaxSpeed:            "max(max_speed)",
		&r.TotalUp:             "max(total_up)",
		&r.AverageSpeed:        "max(average_speed)",
		&r.AverageSpeedNoPause: "max(average_speed_no_pause)",
	}

	for k, v := range mapping {
		query := model.ScopeVisibleWorkouts(
			uc.context.GetDB().Table("workouts").Joins("join map_data on workouts.id = map_data.workout_id"),
			targetUser.ID,
			viewer.ID,
			viewerActorIRI,
		).
			Where("workouts.type = ?", t).
			Select("workouts.id as id", v+" as value", "workouts.date as date").
			Order(v + " DESC").
			Group("workouts.id").
			Limit(1)

		if startDate != nil {
			query = query.Where("workouts.date >= ?", *startDate)
		}

		if endDate != nil {
			query = query.Where("workouts.date <= ?", *endDate)
		}

		if err := query.Scan(k).Error; err != nil {
			return nil, err
		}
	}

	durationQuery := model.ScopeVisibleWorkouts(
		uc.context.GetDB().Table("workouts").Joins("join map_data on workouts.id = map_data.workout_id"),
		targetUser.ID,
		viewer.ID,
		viewerActorIRI,
	).Where("workouts.type = ?", t).
		Select("workouts.id as id", "max(total_duration) as value", "workouts.date as date").
		Order("max(total_duration) DESC").
		Group("workouts.id").
		Limit(1)

	if startDate != nil {
		durationQuery = durationQuery.Where("workouts.date >= ?", *startDate)
	}

	if endDate != nil {
		durationQuery = durationQuery.Where("workouts.date <= ?", *endDate)
	}

	if err := durationQuery.Scan(&r.Duration).Error; err != nil {
		return nil, err
	}

	r.Active = r.Distance.Value > 0 ||
		r.MaxSpeed.Value > 0 ||
		r.TotalUp.Value > 0 ||
		r.Duration.Value > 0

	return r, nil
}

func (uc *userController) getVisibleDistanceRanking(
	targetUser, viewer *model.User,
	viewerActorIRI string,
	t model.WorkoutType,
	label string,
	startDate, endDate *time.Time,
	limit, offset int,
) ([]model.DistanceRecord, int64, error) {
	rows := []struct {
		model.WorkoutIntervalRecord
		Date time.Time
	}{}

	base := model.ScopeVisibleWorkouts(
		uc.context.GetDB().Table("workout_interval_records").
			Select("workout_interval_records.*, workouts.date as date").
			Joins("join workouts on workouts.id = workout_interval_records.workout_id"),
		targetUser.ID,
		viewer.ID,
		viewerActorIRI,
	).Where("workouts.type = ?", t).
		Where("workout_interval_records.label = ?", label)

	if startDate != nil {
		base = base.Where("workouts.date >= ?", *startDate)
	}

	if endDate != nil {
		base = base.Where("workouts.date <= ?", *endDate)
	}

	var totalCount int64
	if err := base.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	q := base
	if limit > 0 {
		q = q.Limit(limit)
	}
	if offset > 0 {
		q = q.Offset(offset)
	}

	q = q.Order("workout_interval_records.duration_seconds asc, workout_interval_records.distance desc, workouts.date asc, workout_interval_records.workout_id asc")

	if err := q.Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	result := make([]model.DistanceRecord, 0, len(rows))
	for _, row := range rows {
		result = append(result, model.DistanceRecord{
			Label:          row.Label,
			TargetDistance: row.TargetDistance,
			Distance:       row.Distance,
			Duration:       time.Duration(row.DurationSeconds * float64(time.Second)),
			AverageSpeed:   row.AverageSpeed,
			WorkoutID:      row.WorkoutID,
			Date:           row.Date,
			StartIndex:     row.StartIndex,
			EndIndex:       row.EndIndex,
			Active:         true,
		})
	}

	return result, totalCount, nil
}

func (uc *userController) getVisibleClimbRanking(targetUser, viewer *model.User, viewerActorIRI string, t model.WorkoutType, startDate, endDate *time.Time, limit, offset int) ([]model.ClimbRecord, int64, error) {
	if !t.IsDistance() {
		return nil, 0, fmt.Errorf("climb ranking is only supported for distance workout types: %s", t)
	}

	var workouts []*model.Workout
	q := model.ScopeVisibleWorkouts(
		model.PreloadWorkoutData(uc.context.GetDB()),
		targetUser.ID,
		viewer.ID,
		viewerActorIRI,
	).Where("workouts.type = ?", t)

	if startDate != nil {
		q = q.Where("workouts.date >= ?", *startDate)
	}

	if endDate != nil {
		q = q.Where("workouts.date <= ?", *endDate)
	}

	if err := q.Find(&workouts).Error; err != nil {
		return nil, 0, err
	}

	records := make([]model.ClimbRecord, 0)
	for _, workout := range workouts {
		if workout == nil || workout.Data == nil {
			continue
		}

		for _, climb := range workout.Data.Climbs {
			if climb.Type != "climb" {
				continue
			}

			records = append(records, model.ClimbRecord{
				ElevationGain: climb.Gain,
				Distance:      climb.Length,
				AverageSlope:  climb.AvgSlope,
				WorkoutID:     workout.ID,
				Date:          workout.Date,
				StartIndex:    climb.StartIdx,
				EndIndex:      climb.EndIdx,
				Active:        true,
			})
		}
	}

	// Keep same ordering semantics as model.GetClimbRanking
	sort.SliceStable(records, func(i, j int) bool {
		a := records[i]
		b := records[j]

		if a.ElevationGain != b.ElevationGain {
			return a.ElevationGain > b.ElevationGain
		}

		if a.Distance != b.Distance {
			return a.Distance > b.Distance
		}

		if a.Date.Equal(b.Date) {
			return false
		}

		return a.Date.Before(b.Date)
	})

	totalCount := int64(len(records))
	start := offset
	if start > len(records) {
		start = len(records)
	}

	end := start + limit
	if limit <= 0 || end > len(records) {
		end = len(records)
	}

	return records[start:end], totalCount, nil
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
