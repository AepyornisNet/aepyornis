package app

import (
	"net/http"
	"strconv"

	"github.com/jovandeginste/workout-tracker/v2/pkg/api"
	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
	"github.com/labstack/echo/v4"
)

func (a *App) registerAPIV2AdminRoutes(e *echo.Group) {
	apiAdminGroup := e.Group("/admin")
	apiAdminGroup.Use(a.ValidateAdminMiddleware)

	apiAdminGroup.GET("/users", a.apiV2AdminUsersHandler).Name = "api-v2-admin-users"
	apiAdminGroup.GET("/users/:id", a.apiV2AdminUserHandler).Name = "api-v2-admin-user"
	apiAdminGroup.PUT("/users/:id", a.apiV2AdminUserUpdateHandler).Name = "api-v2-admin-user-update"
	apiAdminGroup.DELETE("/users/:id", a.apiV2AdminUserDeleteHandler).Name = "api-v2-admin-user-delete"
	apiAdminGroup.PUT("/config", a.apiV2AdminConfigUpdateHandler).Name = "api-v2-admin-config-update"
}

// apiV2AdminUsersHandler returns all users (admin only)
// @Summary      List users (admin)
// @Tags         admin
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Success      200  {object}  api.Response[[]api.UserProfileResponse]
// @Failure      403  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /admin/users [get]
func (a *App) apiV2AdminUsersHandler(c echo.Context) error {
	users, err := database.GetUsers(a.db)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	results := make([]api.UserProfileResponse, len(users))
	for i, u := range users {
		results[i] = api.NewUserProfileResponse(u)
	}

	resp := api.Response[[]api.UserProfileResponse]{
		Results: results,
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2AdminUserHandler returns a specific user (admin only)
// @Summary      Get user (admin)
// @Tags         admin
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "User ID"
// @Produce      json
// @Success      200  {object}  api.Response[api.UserProfileResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      404  {object}  api.Response[any]
// @Router       /admin/users/{id} [get]
func (a *App) apiV2AdminUserHandler(c echo.Context) error {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	user, err := database.GetUserByID(a.db, userID)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusNotFound, err)
	}

	resp := api.Response[api.UserProfileResponse]{
		Results: api.NewUserProfileResponse(user),
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2AdminUserUpdateHandler updates a specific user (admin only)
// @Summary      Update user (admin)
// @Tags         admin
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "User ID"
// @Accept       json
// @Produce      json
// @Success      200  {object}  api.Response[api.UserProfileResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      404  {object}  api.Response[any]
// @Router       /admin/users/{id} [put]
func (a *App) apiV2AdminUserUpdateHandler(c echo.Context) error {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	user, err := database.GetUserByID(a.db, userID)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusNotFound, err)
	}

	// Bind the update data
	var updateData struct {
		Name     string `json:"name"`
		Username string `json:"username"`
		Admin    bool   `json:"admin"`
		Active   bool   `json:"active"`
		Password string `json:"password,omitempty"`
	}

	if err := c.Bind(&updateData); err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	// Update user fields
	user.Name = updateData.Name
	user.Username = updateData.Username
	user.Admin = updateData.Admin
	user.Active = updateData.Active

	// Update password if provided
	if updateData.Password != "" {
		if err := user.SetPassword(updateData.Password); err != nil {
			return a.renderAPIV2Error(c, http.StatusBadRequest, err)
		}
	}

	if err := user.Save(a.db); err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	resp := api.Response[api.UserProfileResponse]{
		Results: api.NewUserProfileResponse(user),
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2AdminUserDeleteHandler deletes a specific user (admin only)
// @Summary      Delete user (admin)
// @Tags         admin
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "User ID"
// @Produce      json
// @Success      200  {object}  api.Response[any]
// @Failure      400  {object}  api.Response[any]
// @Failure      404  {object}  api.Response[any]
// @Router       /admin/users/{id} [delete]
func (a *App) apiV2AdminUserDeleteHandler(c echo.Context) error {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	user, err := database.GetUserByID(a.db, userID)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusNotFound, err)
	}

	if err := user.Delete(a.db); err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	resp := api.Response[any]{
		Results: map[string]string{"message": "User deleted successfully"},
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2AdminConfigUpdateHandler updates application config (admin only)
// @Summary      Update config (admin)
// @Tags         admin
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Success      200  {object}  api.Response[api.AppInfoResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /admin/config [put]
func (a *App) apiV2AdminConfigUpdateHandler(c echo.Context) error {
	var cnf database.Config

	if err := c.Bind(&cnf); err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	if err := cnf.Save(a.db); err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	if err := a.ResetConfiguration(); err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	// Return the updated config
	resp := api.Response[api.AppInfoResponse]{
		Results: api.AppInfoResponse{
			Version:              a.Version.PrettyVersion(),
			RegistrationDisabled: a.Config.RegistrationDisabled,
			SocialsDisabled:      a.Config.SocialsDisabled,
		},
	}

	return c.JSON(http.StatusOK, resp)
}
