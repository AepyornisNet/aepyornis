package controller

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	ap "github.com/jovandeginste/workout-tracker/v2/pkg/activitypub"
	"github.com/jovandeginste/workout-tracker/v2/pkg/container"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model/dto"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
)

type ProfileController interface {
	GetProfile(c echo.Context) error
	UpdateProfile(c echo.Context) error
	ResetAPIKey(c echo.Context) error
	EnableActivityPub(c echo.Context) error
	ListFollowRequests(c echo.Context) error
	AcceptFollowRequest(c echo.Context) error
	RefreshWorkouts(c echo.Context) error
	UpdateVersion(c echo.Context) error
}

type profileController struct {
	context *container.Container
}

func NewProfileController(c *container.Container) ProfileController {
	return &profileController{context: c}
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
	user := pc.context.GetUser(c)

	resp := dto.Response[dto.UserProfileResponse]{
		Results: dto.NewUserProfileResponse(user),
	}

	if !pc.context.GetConfig().AutoImportEnabled {
		resp.Results.Profile.AutoImportDirectory = ""
	}

	if user.Profile.APIActive {
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
	user := pc.context.GetUser(c)

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
		user.Birthdate = &bd
	} else {
		user.Birthdate = nil
	}

	user.Profile.PreferredUnits = updateData.PreferredUnits
	user.Profile.Language = updateData.Language
	user.Profile.Theme = updateData.Theme
	user.Profile.TotalsShow = model.WorkoutType(updateData.TotalsShow)
	user.Profile.Timezone = updateData.Timezone
	if !updateData.DefaultWorkoutVisibility.IsValid() {
		return renderApiError(c, http.StatusBadRequest, errors.New("invalid default workout visibility"))
	}
	user.Profile.DefaultWorkoutVisibility = updateData.DefaultWorkoutVisibility
	if !pc.context.GetConfig().AutoImportEnabled {
		if updateData.AutoImportDirectory != "" {
			return renderApiError(c, http.StatusBadRequest, errors.New("auto import is disabled"))
		}

		user.Profile.AutoImportDirectory = ""
	} else {
		user.Profile.AutoImportDirectory = updateData.AutoImportDirectory
	}
	user.Profile.APIActive = updateData.APIActive
	user.Profile.PreferFullDate = updateData.PreferFullDate
	user.Profile.UserID = user.ID

	if err := user.Profile.Save(pc.context.GetDB()); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if err := user.Save(pc.context.GetDB()); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[dto.UserProfileResponse]{
		Results: dto.NewUserProfileResponse(user),
	}

	if !pc.context.GetConfig().AutoImportEnabled {
		resp.Results.Profile.AutoImportDirectory = ""
	}

	if user.Profile.APIActive {
		resp.Results.Profile.APIKey = user.APIKey
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
	user := pc.context.GetUser(c)

	user.GenerateAPIKey(true)

	if err := user.Save(pc.context.GetDB()); err != nil {
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
	user := pc.context.GetUser(c)

	user.ActivityPub = !user.ActivityPub

	if user.ActivityPub && (user.PublicKey == "" || user.PrivateKey == "") {
		if err := user.GenerateActivityPubKeys(false); err != nil {
			return renderApiError(c, http.StatusInternalServerError, err)
		}
	}

	if err := user.Save(pc.context.GetDB()); err != nil {
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
	user := pc.context.GetUser(c)

	requests, err := model.ListFollowerRequests(pc.context.GetDB(), user.ID)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	results := make([]dto.FollowRequestResponse, 0, len(requests))
	for _, req := range requests {
		results = append(results, dto.NewFollowRequestResponse(req))
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
	user := pc.context.GetUser(c)

	rawID := c.Param("id")
	id, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	follower, err := model.ApproveFollowerRequest(pc.context.GetDB(), user.ID, id)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	actorURL := ap.LocalActorURL(ap.LocalActorURLConfig{
		Host:           pc.context.GetConfig().Host,
		WebRoot:        pc.context.GetConfig().WebRoot,
		FallbackHost:   c.Request().Host,
		FallbackScheme: c.Scheme(),
	}, user.Username)
	if err := ap.SendFollowAccept(c.Request().Context(), actorURL, user.PrivateKey, *follower); err != nil {
		return renderApiError(c, http.StatusBadGateway, err)
	}

	return c.JSON(http.StatusOK, dto.Response[dto.FollowRequestResponse]{
		Results: dto.NewFollowRequestResponse(*follower),
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
	user := pc.context.GetUser(c)

	if err := user.MarkWorkoutsDirty(pc.context.GetDB()); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[map[string]string]{
		Results: map[string]string{
			"message": "All workouts marked for refresh",
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
	u := pc.context.GetUser(c)

	v := pc.context.GetVersion()
	if v == nil {
		return c.String(http.StatusInternalServerError, "version not configured")
	}

	u.LastVersion = v.Sha
	if err := u.Save(pc.context.GetDB()); err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, u.LastVersion)
}
