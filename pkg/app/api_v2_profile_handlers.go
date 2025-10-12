package app

import (
	"net/http"

	"github.com/jovandeginste/workout-tracker/v2/pkg/api"
	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
	"github.com/labstack/echo/v4"
)

func (a *App) registerAPIV2ProfileRoutes(apiGroup *echo.Group) {
	apiGroup.GET("/profile", a.apiV2ProfileHandler).Name = "api-v2-profile"
	apiGroup.PUT("/profile", a.apiV2ProfileUpdateHandler).Name = "api-v2-profile-update"
	apiGroup.POST("/profile/reset-api-key", a.apiV2ProfileResetAPIKeyHandler).Name = "api-v2-profile-reset-api-key"
	apiGroup.POST("/profile/refresh-workouts", a.apiV2ProfileRefreshWorkoutsHandler).Name = "api-v2-profile-refresh-workouts"
}

// apiV2ProfileHandler returns current user's full profile with settings
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
func (a *App) apiV2ProfileUpdateHandler(c echo.Context) error {
	user := a.getCurrentUser(c)

	var updateData struct {
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
