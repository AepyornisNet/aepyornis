package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/model/dto"
	"github.com/AepyornisNet/aepyornis/pkg/repository"
	"github.com/AepyornisNet/aepyornis/pkg/service"
	"github.com/labstack/echo/v5"
	"github.com/samber/do/v2"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

type UserController interface {
	GetWhoami(c *echo.Context) error
	GetUserProfileByHandle(c *echo.Context) error
	SearchProfiles(c *echo.Context) error
	FollowUserByHandle(c *echo.Context) error
	UnfollowUserByHandle(c *echo.Context) error
	GetTotals(c *echo.Context) error
	GetRecords(c *echo.Context) error
	GetRecordsRanking(c *echo.Context) error
	GetClimbRecordsRanking(c *echo.Context) error
	GetUserByID(c *echo.Context) error
}

type userController struct {
	userSvc  service.UserService
	userRepo repository.User
}

func NewUserController(injector do.Injector) UserController {
	return &userController{
		userSvc:  do.MustInvoke[service.UserService](injector),
		userRepo: do.MustInvoke[repository.User](injector),
	}
}

// GetWhoami returns current user information
// @Summary      Get current user profile
// @Tags         user
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Success      200  {object}  dto.Response[dto.UserProfileResponse]
// @Failure      401  {object}  dto.Response[string]
// @Router       /whoami [get]
func (uc *userController) GetWhoami(c *echo.Context) error {
	user := currentUser(c)

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
// @Failure      500  {object}  dto.Response[string]
// @Router       /totals [get]
func (uc *userController) GetTotals(c *echo.Context) error {
	var req dto.TotalsRequest
	if err := c.Bind(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	viewer := currentUser(c)
	targetUser, err := uc.userSvc.ResolveTargetUser(strings.TrimSpace(req.Handle), c.Request().Host, viewer)
	if err != nil {
		if errors.Is(err, dto.ErrNotAuthorized) {
			return renderApiError(c, http.StatusForbidden, dto.ErrNotAuthorized)
		}
		return renderApiError(c, http.StatusNotFound, err)
	}

	startDate, endDate, err := parseDateRange(req.Start, req.End)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	totals, err := uc.userSvc.GetTotals(c.Request().Context(), targetUser, viewer, startDate, endDate)
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
// @Failure      500  {object}  dto.Response[string]
// @Router       /records [get]
func (uc *userController) GetRecords(c *echo.Context) error {
	var req dto.RecordsRequest
	if err := c.Bind(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	viewer := currentUser(c)
	targetUser, err := uc.userSvc.ResolveTargetUser(strings.TrimSpace(req.Handle), c.Request().Host, viewer)
	if err != nil {
		if errors.Is(err, dto.ErrNotAuthorized) {
			return renderApiError(c, http.StatusNotFound, dto.ErrNotAuthorized)
		}
		return renderApiError(c, http.StatusNotFound, err)
	}

	startDate, endDate, err := parseDateRange(req.Start, req.End)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	records, err := uc.userSvc.GetVisibleRecords(c.Request().Context(), targetUser, viewer, startDate, endDate)
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
// @Failure      400  {object}  dto.Response[string]
// @Failure      500  {object}  dto.Response[string]
// @Router       /records/ranking [get]
func (uc *userController) GetRecordsRanking(c *echo.Context) error {
	var req dto.RecordsRankingRequest
	if err := c.Bind(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	req.SetDefaults()

	viewer := currentUser(c)
	targetUser, err := uc.userSvc.ResolveTargetUser(strings.TrimSpace(req.Handle), c.Request().Host, viewer)
	if err != nil {
		if errors.Is(err, dto.ErrNotAuthorized) {
			return renderApiError(c, http.StatusNotFound, dto.ErrNotAuthorized)
		}
		return renderApiError(c, http.StatusNotFound, err)
	}

	wt := model.AsWorkoutType(req.WorkoutType)
	startDate, endDate, err := parseDateRange(req.Start, req.End)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	isPower := false
	for _, target := range model.PowerRecordTargetsFor(wt) {
		if target.Label == req.Label {
			isPower = true
			break
		}
	}

	var totalCount int64
	var recordResponses []dto.DistanceRecordResponse

	if isPower {
		powerRecords, count, err := uc.userSvc.GetPowerRanking(c.Request().Context(), targetUser, viewer, wt, req.Label, startDate, endDate, req.PerPage, req.GetOffset())
		if err != nil {
			return renderApiError(c, http.StatusInternalServerError, err)
		}
		totalCount = count
		recordResponses = dto.NewPowerRecordRankingResponses(powerRecords)
	} else {
		records, count, err := uc.userSvc.GetDistanceRanking(c.Request().Context(), targetUser, viewer, wt, req.Label, startDate, endDate, req.PerPage, req.GetOffset())
		if err != nil {
			return renderApiError(c, http.StatusInternalServerError, err)
		}
		totalCount = count
		recordResponses = dto.NewDistanceRecordResponses(records)
	}

	resp := dto.PaginatedResponse[dto.DistanceRecordResponse]{
		Results:    recordResponses,
		Page:       req.Page,
		PerPage:    req.PerPage,
		TotalPages: req.CalculateTotalPages(totalCount),
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
// @Failure      400  {object}  dto.Response[string]
// @Failure      500  {object}  dto.Response[string]
// @Router       /records/climbs/ranking [get]
func (uc *userController) GetClimbRecordsRanking(c *echo.Context) error {
	var req dto.ClimbRecordsRankingRequest
	if err := c.Bind(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	req.SetDefaults()

	viewer := currentUser(c)
	targetUser, err := uc.userSvc.ResolveTargetUser(strings.TrimSpace(req.Handle), c.Request().Host, viewer)
	if err != nil {
		if errors.Is(err, dto.ErrNotAuthorized) {
			return renderApiError(c, http.StatusNotFound, dto.ErrNotAuthorized)
		}
		return renderApiError(c, http.StatusNotFound, err)
	}

	wt := model.AsWorkoutType(req.WorkoutType)
	startDate, endDate, err := parseDateRange(req.Start, req.End)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	records, totalCount, err := uc.userSvc.GetClimbRanking(c.Request().Context(), targetUser, viewer, wt, startDate, endDate, req.PerPage, req.GetOffset())
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.PaginatedResponse[dto.ClimbRecordResponse]{
		Results:    dto.NewClimbRecordResponses(records),
		Page:       req.Page,
		PerPage:    req.PerPage,
		TotalPages: req.CalculateTotalPages(totalCount),
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
// @Failure      403  {object}  dto.Response[string]
// @Failure      404  {object}  dto.Response[string]
// @Router       /{id} [get]
// TODO: Add more data. This will be used for public profiles.
func (uc *userController) GetUserByID(c *echo.Context) error {
	id, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	var req dto.DateRangeRequest
	if err := c.Bind(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	u, err := uc.userRepo.GetByID(id)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if u.IsAnonymous() {
		return renderApiError(c, http.StatusForbidden, dto.ErrNotAuthorized)
	}

	startDate, endDate, err := parseDateRange(req.Start, req.End)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	records, err := u.GetAllPersonalRecords(startDate, endDate)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[[]dto.WorkoutRecordResponse]{
		Results: dto.NewWorkoutRecordsResponse(records),
	}

	return c.JSON(http.StatusOK, resp)
}

// SearchProfiles searches followable profiles by username, display name, or exact ActivityPub handle
// @Summary      Search user profiles
// @Tags         user
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        q  query     string  false  "Search query or ActivityPub handle"
// @Produce      json
// @Success      200  {object}  dto.Response[[]dto.ActivityPubProfileSummaryResponse]
// @Failure      403  {object}  dto.Response[string]
// @Failure      500  {object}  dto.Response[string]
// @Router       /user-profile/search [get]
func (uc *userController) SearchProfiles(c *echo.Context) error {
	viewer := currentUser(c)
	if viewer.IsAnonymous() {
		return renderApiError(c, http.StatusForbidden, dto.ErrNotAuthorized)
	}

	var req dto.ProfileSearchRequest
	if err := c.Bind(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	results, err := uc.userSvc.SearchProfiles(c.Request().Context(), viewer, req.Query, c.Request().Host, c.Scheme())
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[[]dto.ActivityPubProfileSummaryResponse]{
		Results: results,
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
// @Failure      404  {object}  dto.Response[string]
// @Router       /user-profile [get]
func (uc *userController) GetUserProfileByHandle(c *echo.Context) error {
	var req dto.UserHandleRequest
	if err := c.Bind(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	viewer := currentUser(c)
	summary, err := uc.userSvc.GetUserProfileByHandle(c.Request().Context(), viewer, req.Handle, c.Request().Host, c.Scheme())
	if err != nil {
		if errors.Is(err, dto.ErrNotAuthorized) || errors.Is(err, gorm.ErrRecordNotFound) {
			return renderApiError(c, http.StatusNotFound, err)
		}
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[dto.ActivityPubProfileSummaryResponse]{
		Results: summary,
	}

	return c.JSON(http.StatusOK, resp)
}

// FollowUserByHandle follows an ActivityPub-enabled local or remote user
// @Summary      Follow user by ActivityPub handle
// @Tags         user
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        handle  query     string  true  "ActivityPub handle"
// @Produce      json
// @Success      200  {object}  dto.Response[dto.ActivityPubProfileSummaryResponse]
// @Failure      400  {object}  dto.Response[string]
// @Failure      404  {object}  dto.Response[string]
// @Router       /user-profile/follow [post]
func (uc *userController) FollowUserByHandle(c *echo.Context) error {
	var req dto.UserHandleRequest
	if err := c.Bind(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	viewer := currentUser(c)
	summary, err := uc.userSvc.FollowUserByHandle(c.Request().Context(), viewer, req.Handle, c.Request().Host, c.Scheme())
	if err != nil {
		if errors.Is(err, service.ErrCannotFollowSelf) ||
			errors.Is(err, service.ErrActivityPubRequiredToFollow) ||
			errors.Is(err, service.ErrTargetNoActivityPub) ||
			errors.Is(err, service.ErrRemoteInboxNotFound) {
			return renderApiError(c, http.StatusBadRequest, err)
		}
		if errors.Is(err, dto.ErrNotAuthorized) {
			return renderApiError(c, http.StatusForbidden, err)
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return renderApiError(c, http.StatusNotFound, err)
		}
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, dto.Response[dto.ActivityPubProfileSummaryResponse]{
		Results: summary,
	})
}

// UnfollowUserByHandle unfollows an ActivityPub-enabled local or remote user
// @Summary      Unfollow user by ActivityPub handle
// @Tags         user
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        handle  query     string  true  "ActivityPub handle"
// @Produce      json
// @Success      200  {object}  dto.Response[dto.ActivityPubProfileSummaryResponse]
// @Failure      400  {object}  dto.Response[string]
// @Failure      404  {object}  dto.Response[string]
// @Router       /user-profile/unfollow [post]
func (uc *userController) UnfollowUserByHandle(c *echo.Context) error {
	var req dto.UserHandleRequest
	if err := c.Bind(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	viewer := currentUser(c)
	summary, err := uc.userSvc.UnfollowUserByHandle(c.Request().Context(), viewer, req.Handle, c.Request().Host, c.Scheme())
	if err != nil {
		if errors.Is(err, service.ErrCannotUnfollowSelf) ||
			errors.Is(err, service.ErrActivityPubRequiredToFollow) ||
			errors.Is(err, service.ErrRemoteInboxNotFound) {
			return renderApiError(c, http.StatusBadRequest, err)
		}
		if errors.Is(err, dto.ErrNotAuthorized) {
			return renderApiError(c, http.StatusForbidden, err)
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return renderApiError(c, http.StatusNotFound, err)
		}
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, dto.Response[dto.ActivityPubProfileSummaryResponse]{
		Results: summary,
	})
}

func parseDateRange(startStr, endStr string) (*time.Time, *time.Time, error) {
	var startDate, endDate *time.Time
	if startStr != "" {
		parsedStart, err := parseDateFlexible(startStr)
		if err != nil {
			return nil, nil, err
		}
		startDate = &parsedStart
	}
	if endStr != "" {
		parsedEnd, err := parseDateFlexible(endStr)
		if err != nil {
			return nil, nil, err
		}
		endOfDay := time.Date(parsedEnd.Year(), parsedEnd.Month(), parsedEnd.Day(), 23, 59, 59, 999999999, parsedEnd.Location())
		endDate = &endOfDay
	}
	return startDate, endDate, nil
}

func parseDateFlexible(dateStr string) (time.Time, error) {
	if sec, err := strconv.ParseInt(dateStr, 10, 64); err == nil {
		return time.Unix(sec, 0).UTC(), nil
	}
	layouts := []string{
		"2006-01-02",
		time.RFC3339,
		"2006-01-02T15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("invalid date format: expected YYYY-MM-DD, RFC3339, or unix timestamp")
}
