package controller

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/config"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/model/dto"
	"github.com/AepyornisNet/aepyornis/pkg/repository"
	"github.com/AepyornisNet/aepyornis/pkg/service"
	"github.com/AepyornisNet/aepyornis/pkg/version"
	"github.com/AepyornisNet/aepyornis/pkg/worker"
	"github.com/labstack/echo/v4"
	"github.com/samber/do/v2"
	"github.com/vgarvardt/gue/v6"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ProfileController interface {
	GetProfile(c echo.Context) error
	UpdateProfile(c echo.Context) error
	ChangePassword(c echo.Context) error
	ResetAPIKey(c echo.Context) error
	EnableActivityPub(c echo.Context) error
	ListFollowRequests(c echo.Context) error
	AcceptFollowRequest(c echo.Context) error
	RefreshWorkouts(c echo.Context) error
	UpdateVersion(c echo.Context) error
}

var ErrCurrentPasswordIncorrect = errors.New("current password is incorrect")

type profileController struct {
	cfg          *config.Config
	db           *gorm.DB
	followerRepo repository.Follower
	logger       *slog.Logger
	client       *gue.Client
	actorService service.ActivityPubActorService
	version      *version.Version
}

func NewProfileController(injector do.Injector) ProfileController {
	return &profileController{
		cfg:          do.MustInvoke[*config.Config](injector),
		db:           do.MustInvoke[*gorm.DB](injector),
		followerRepo: do.MustInvoke[repository.Follower](injector),
		logger:       do.MustInvoke[*slog.Logger](injector),
		client:       do.MustInvoke[*gue.Client](injector),
		actorService: do.MustInvoke[service.ActivityPubActorService](injector),
		version:      do.MustInvoke[*version.Version](injector),
	}
}

// GetProfile returns current user's full profile with settings
// @Summary      Get profile
// @Tags         profile
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Success      200  {object}  dto.Response[dto.UserProfileResponse]
// @Router       /profile [get]
func (pc *profileController) GetProfile(c echo.Context) error {
	user := currentUser(c)

	resp := dto.Response[dto.UserProfileResponse]{
		Results: dto.NewUserProfileResponse(user),
	}

	if !pc.cfg.AutoImportEnabled {
		resp.Results.Profile.AutoImportDirectory = ""
	}

	if user.APIActive {
		resp.Results.Profile.APIKey = user.APIKey
	}

	return c.JSON(http.StatusOK, resp)
}

// UpdateProfile updates current user's profile
// @Summary      Update profile
// @Tags         profile
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Success      200  {object}  dto.Response[dto.UserProfileResponse]
// @Failure      400  {object}  dto.Response[any]
// @Failure      500  {object}  dto.Response[any]
// @Router       /profile [put]
func (pc *profileController) UpdateProfile(c echo.Context) error {
	user := currentUser(c)

	var updateData dto.ProfileUpdateData
	if err := c.Bind(&updateData); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	if updateData.Birthdate != nil && *updateData.Birthdate != "" {
		t, err := time.Parse("2006-01-02", *updateData.Birthdate)
		if err != nil {
			return renderApiError(c, http.StatusBadRequest, err)
		}
		bd := datatypes.Date(t)
		user.Profile.Birthdate = &bd
	} else {
		user.Profile.Birthdate = nil
	}

	user.PreferredUnits = updateData.PreferredUnits
	user.Language = updateData.Language
	user.Theme = updateData.Theme
	user.TotalsShow = model.WorkoutType(updateData.TotalsShow)
	user.TZ = updateData.Timezone
	if !updateData.DefaultWorkoutVisibility.IsValid() {
		return renderApiError(c, http.StatusBadRequest, errors.New("invalid default workout visibility"))
	}
	user.DefaultWorkoutVisibility = updateData.DefaultWorkoutVisibility
	if !pc.cfg.AutoImportEnabled {
		if updateData.AutoImportDirectory != "" {
			return renderApiError(c, http.StatusBadRequest, errors.New("auto import is disabled"))
		}

		user.AutoImportDirectory = ""
	} else {
		user.AutoImportDirectory = updateData.AutoImportDirectory
	}
	user.APIActive = updateData.APIActive
	user.PreferFullDate = updateData.PreferFullDate
	userID := user.ID
	user.Profile.UserID = &userID

	if err := user.Profile.Save(pc.db); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if err := user.Save(pc.db); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[dto.UserProfileResponse]{
		Results: dto.NewUserProfileResponse(user),
	}

	if !pc.cfg.AutoImportEnabled {
		resp.Results.Profile.AutoImportDirectory = ""
	}

	if user.APIActive {
		resp.Results.Profile.APIKey = user.APIKey
	}

	return c.JSON(http.StatusOK, resp)
}

// ChangePassword changes the current user's password
// @Summary      Change password
// @Tags         profile
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Success      200  {object}  dto.Response[map[string]string]
// @Failure      400  {object}  dto.Response[any]
// @Failure      401  {object}  dto.Response[any]
// @Failure      500  {object}  dto.Response[any]
// @Router       /profile/change-password [post]
func (pc *profileController) ChangePassword(c echo.Context) error {
	user := currentUser(c)

	var changeData dto.ProfileChangePasswordData
	if err := c.Bind(&changeData); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	if changeData.CurrentPassword == "" || changeData.NewPassword == "" {
		return renderApiError(c, http.StatusBadRequest, dto.ErrBadRequest)
	}

	if !user.ValidLogin(changeData.CurrentPassword) {
		return renderApiError(c, http.StatusUnauthorized, ErrCurrentPasswordIncorrect)
	}

	if err := user.SetPassword(changeData.NewPassword); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	if err := user.Save(pc.db); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[map[string]string]{
		Results: map[string]string{"message": "Password changed successfully"},
	}

	return c.JSON(http.StatusOK, resp)
}

// ResetAPIKey resets current user's API key
// @Summary      Reset API key
// @Tags         profile
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Success      200  {object}  dto.Response[map[string]string]
// @Failure      500  {object}  dto.Response[any]
// @Router       /profile/reset-api-key [post]
func (pc *profileController) ResetAPIKey(c echo.Context) error {
	user := currentUser(c)

	user.GenerateAPIKey(true)

	if err := user.Save(pc.db); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[map[string]string]{
		Results: map[string]string{
			"api_key": user.APIKey,
			"message": "API key reset successfully",
		},
	}

	return c.JSON(http.StatusOK, resp)
}

// EnableActivityPub toggles current user's ActivityPub setting and generates keys if needed
// @Summary      Toggle ActivityPub support
// @Tags         profile
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Success      200  {object}  dto.Response[any]
// @Failure      500  {object}  dto.Response[any]
// @Router       /profile/enable-activity-pub [post]
func (pc *profileController) EnableActivityPub(c echo.Context) error {
	user := currentUser(c)

	user.ActivityPub = !user.ActivityPub

	if user.ActivityPub && (user.Profile.PublicKey == "" || user.Profile.PrivateKey == "") {
		user.Profile.User = user
		if err := user.Profile.GenerateActivityPubKeys(false); err != nil {
			return renderApiError(c, http.StatusInternalServerError, err)
		}
	}

	if err := user.Save(pc.db); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[map[string]any]{
		Results: map[string]any{
			"activity_pub": user.ActivityPub,
			"message":      "ActivityPub setting enabled",
		},
	}

	return c.JSON(http.StatusOK, resp)
}

// ListFollowRequests returns pending ActivityPub follow requests for the current user
// @Summary      List ActivityPub follow requests
// @Tags         profile
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Success      200  {object}  dto.Response[[]dto.FollowRequestResponse]
// @Failure      500  {object}  dto.Response[any]
// @Router       /profile/follow-requests [get]
func (pc *profileController) ListFollowRequests(c echo.Context) error {
	user := currentUser(c)

	requests, err := pc.followerRepo.ListFollowerRequests(user.Profile.ID)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	results := make([]dto.FollowRequestResponse, 0, len(requests))
	for _, req := range requests {
		actorURL, _ := pc.actorService.ActorURL(req.Profile)
		results = append(results, dto.NewFollowRequestResponse(req, actorURL))
	}

	return c.JSON(http.StatusOK, dto.Response[[]dto.FollowRequestResponse]{
		Results: results,
	})
}

// AcceptFollowRequest approves a pending ActivityPub follow request and sends Accept
// @Summary      Accept ActivityPub follow request
// @Tags         profile
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "Follow request ID"
// @Produce      json
// @Success      200  {object}  dto.Response[dto.FollowRequestResponse]
// @Failure      400  {object}  dto.Response[any]
// @Failure      500  {object}  dto.Response[any]
// @Failure      502  {object}  dto.Response[any]
// @Router       /profile/follow-requests/{id}/accept [post]
func (pc *profileController) AcceptFollowRequest(c echo.Context) error {
	user := currentUser(c)

	rawID := c.Param("id")
	id, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	follower, err := pc.followerRepo.ApproveFollowerRequest(user.Profile.ID, id)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if err := pc.actorService.SendFollowAccept(c.Request().Context(), &user.Profile, *follower); err != nil {
		return renderApiError(c, http.StatusBadGateway, err)
	}

	return c.JSON(http.StatusOK, dto.Response[dto.FollowRequestResponse]{
		Results: dto.NewFollowRequestResponse(*follower, follower.Profile.ActorURL()),
	})
}

// RefreshWorkouts marks all workouts for refresh
// @Summary      Refresh all workouts
// @Tags         profile
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Success      200  {object}  dto.Response[map[string]string]
// @Failure      500  {object}  dto.Response[any]
// @Router       /profile/refresh-workouts [post]
func (pc *profileController) RefreshWorkouts(c echo.Context) error {
	user := currentUser(c)
	db := pc.db

	var workoutIDs []uint64
	if err := db.Model(&model.Workout{}).
		Where("profile_id = ?", user.Profile.ID).
		Pluck("id", &workoutIDs).Error; err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if err := user.Profile.MarkWorkoutsDirty(db); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	enqueued := 0
	failed := 0
	for _, workoutID := range workoutIDs {
		if err := worker.EnqueueWorkoutUpdate(c.Request().Context(), pc.client, workoutID); err != nil {
			failed++
			pc.logger.Error("Failed to enqueue workout update", "workout_id", workoutID, "error", err)
			continue
		}

		enqueued++
	}

	if len(workoutIDs) > 0 && failed == len(workoutIDs) {
		return renderApiError(c, http.StatusInternalServerError, errors.New("failed to enqueue workout refresh jobs"))
	}

	var message string
	switch {
	case len(workoutIDs) == 0:
		message = "No workouts found"
	case failed > 0:
		message = "All workouts marked for refresh; some workouts could not be scheduled"
	default:
		message = "Workouts will be refreshed soon"
	}

	resp := dto.Response[map[string]string]{
		Results: map[string]string{
			"message": message,
			"count":   strconv.Itoa(enqueued),
		},
	}

	return c.JSON(http.StatusOK, resp)
}

// UpdateVersion updates the user's last known app version
// @Summary      Update app version
// @Tags         profile
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Success      200  {string}  string
// @Failure      500  {string}  string
// @Router       /profile/update-version [post]
func (pc *profileController) UpdateVersion(c echo.Context) error {
	u := currentUser(c)

	v := pc.version
	if v == nil {
		return c.String(http.StatusInternalServerError, "version not configured")
	}

	u.LastVersion = v.Sha
	if err := u.Save(pc.db); err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, u.LastVersion)
}
