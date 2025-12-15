package app

import (
	"net/http"
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/api"
	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
)

func (a *App) registerAPIV2ProfileRoutes(apiGroup *echo.Group) {
	selfGroup := apiGroup.Group("/profile")
	selfGroup.GET("", a.apiV2ProfileHandler).Name = "api-v2-profile"
	selfGroup.PUT("", a.apiV2ProfileUpdateHandler).Name = "api-v2-profile-update"
	selfGroup.POST("/reset-api-key", a.apiV2ProfileResetAPIKeyHandler).Name = "api-v2-profile-reset-api-key"
	selfGroup.POST("/refresh-workouts", a.apiV2ProfileRefreshWorkoutsHandler).Name = "api-v2-profile-refresh-workouts"
	selfGroup.POST("/update-version", a.apiV2UserUpdateVersion).Name = "api-v2-user-update-version"
}

// apiV2ProfileHandler returns current user's full profile with settings
// @Summary      Get profile
// @Tags         profile
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Success      200  {object}  api.Response[api.UserProfileResponse]
// @Router       /profile [get]
func (a *App) apiV2ProfileHandler(c echo.Context) error {
	user := a.getCurrentUser(c)

	resp := api.Response[api.UserProfileResponse]{
		Results: api.NewUserProfileResponse(user),
	}

	// Include API key only if API is active
	if user.Profile.APIActive {
		resp.Results.Profile.APIKey = user.APIKey
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2ProfileUpdateHandler updates current user's profile
// @Summary      Update profile
// @Tags         profile
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Success      200  {object}  api.Response[api.UserProfileResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /profile [put]
func (a *App) apiV2ProfileUpdateHandler(c echo.Context) error {
	user := a.getCurrentUser(c)

	var updateData struct {
		Birthdate           *string                     `json:"birthdate"`
		PreferredUnits      database.UserPreferredUnits `json:"preferred_units"`
		Language            string                      `json:"language"`
		Theme               string                      `json:"theme"`
		TotalsShow          string                      `json:"totals_show"`
		Timezone            string                      `json:"timezone"`
		AutoImportDirectory string                      `json:"auto_import_directory"`
		APIActive           bool                        `json:"api_active"`
		SocialsDisabled     bool                        `json:"socials_disabled"`
		PreferFullDate      bool                        `json:"prefer_full_date"`
	}

	if err := c.Bind(&updateData); err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	// Update user birthdate if provided
	if updateData.Birthdate != nil && *updateData.Birthdate != "" {
		t, err := time.Parse("2006-01-02", *updateData.Birthdate)
		if err != nil {
			return a.renderAPIV2Error(c, http.StatusBadRequest, err)
		}
		bd := datatypes.Date(t)
		user.Birthdate = &bd
	} else {
		user.Birthdate = nil
	}

	// Update profile fields
	user.Profile.PreferredUnits = updateData.PreferredUnits
	user.Profile.Language = updateData.Language
	user.Profile.Theme = updateData.Theme
	user.Profile.TotalsShow = database.WorkoutType(updateData.TotalsShow)
	user.Profile.Timezone = updateData.Timezone
	user.Profile.AutoImportDirectory = updateData.AutoImportDirectory
	user.Profile.APIActive = updateData.APIActive
	user.Profile.SocialsDisabled = updateData.SocialsDisabled
	user.Profile.PreferFullDate = updateData.PreferFullDate
	user.Profile.UserID = user.ID

	if err := user.Profile.Save(a.db); err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	// Save user to update birthdate
	if err := user.Save(a.db); err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	// Return updated profile
	resp := api.Response[api.UserProfileResponse]{
		Results: api.NewUserProfileResponse(user),
	}

	if user.Profile.APIActive {
		resp.Results.Profile.APIKey = user.APIKey
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2ProfileResetAPIKeyHandler resets current user's API key
// @Summary      Reset API key
// @Tags         profile
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Success      200  {object}  api.Response[map[string]string]
// @Failure      500  {object}  api.Response[any]
// @Router       /profile/reset-api-key [post]
func (a *App) apiV2ProfileResetAPIKeyHandler(c echo.Context) error {
	user := a.getCurrentUser(c)

	user.GenerateAPIKey(true)

	if err := user.Save(a.db); err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	resp := api.Response[map[string]string]{
		Results: map[string]string{
			"api_key": user.APIKey,
			"message": "API key reset successfully",
		},
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2ProfileRefreshWorkoutsHandler marks all workouts for refresh
// @Summary      Refresh all workouts
// @Tags         profile
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Success      200  {object}  api.Response[map[string]string]
// @Failure      500  {object}  api.Response[any]
// @Router       /profile/refresh-workouts [post]
func (a *App) apiV2ProfileRefreshWorkoutsHandler(c echo.Context) error {
	user := a.getCurrentUser(c)

	if err := user.MarkWorkoutsDirty(a.db); err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	resp := api.Response[map[string]string]{
		Results: map[string]string{
			"message": "All workouts marked for refresh",
		},
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2UserUpdateVersion updates the user's last known app version
// @Summary      Update app version
// @Tags         profile
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Success      200  {string}  string
// @Failure      500  {string}  string
// @Router       /profile/update-version [post]
func (a *App) apiV2UserUpdateVersion(c echo.Context) error {
	u := a.getCurrentUser(c)

	u.LastVersion = a.Version.Sha
	if err := u.Save(a.db); err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, u.LastVersion)
}
