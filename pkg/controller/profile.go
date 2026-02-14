package controller

import (
	"net/http"
	"time"

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
// @Success      200  {object}  api.Response[dto.UserProfileResponse]
// @Router       /profile [get]
func (pc *profileController) GetProfile(c echo.Context) error {
	user := pc.context.GetUser(c)

	resp := dto.Response[dto.UserProfileResponse]{
		Results: dto.NewUserProfileResponse(user),
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
// @Success      200  {object}  api.Response[dto.UserProfileResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
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
	user.Profile.AutoImportDirectory = updateData.AutoImportDirectory
	user.Profile.APIActive = updateData.APIActive
	user.Profile.SocialsDisabled = updateData.SocialsDisabled
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
// @Success      200  {object}  api.Response[map[string]string]
// @Failure      500  {object}  api.Response[any]
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

// RefreshWorkouts marks all workouts for refresh
// @Summary      Refresh all workouts
// @Tags         profile
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Success      200  {object}  api.Response[map[string]string]
// @Failure      500  {object}  api.Response[any]
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
