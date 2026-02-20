package controller

import (
	"net/http"
	"strconv"

	"github.com/jovandeginste/workout-tracker/v2/pkg/container"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model/dto"
	"github.com/labstack/echo/v4"
)

type AdminController interface {
	GetUsers(c echo.Context) error
	GetUser(c echo.Context) error
	UpdateUser(c echo.Context) error
	DeleteUser(c echo.Context) error
	UpdateConfig(c echo.Context) error
}

type adminController struct {
	context            *container.Container
	resetConfiguration func() error
}

func NewAdminController(c *container.Container, resetConfiguration func() error) AdminController {
	return &adminController{context: c, resetConfiguration: resetConfiguration}
}

// GetUsers returns all users (admin only)
// @Summary      List users (admin)
// @Tags         admin
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Success      200  {object}  dto.Response[[]dto.UserProfileResponse]
// @Failure      403  {object}  dto.Response[any]
// @Failure      500  {object}  dto.Response[any]
// @Router       /admin/users [get]
func (ac *adminController) GetUsers(c echo.Context) error {
	users, err := model.GetUsers(ac.context.GetDB())
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	results := make([]dto.UserProfileResponse, len(users))
	for i, u := range users {
		results[i] = dto.NewUserProfileResponse(u)
	}

	resp := dto.Response[[]dto.UserProfileResponse]{
		Results: results,
	}

	return c.JSON(http.StatusOK, resp)
}

// GetUser returns a specific user (admin only)
// @Summary      Get user (admin)
// @Tags         admin
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "User ID"
// @Produce      json
// @Success      200  {object}  dto.Response[dto.UserProfileResponse]
// @Failure      400  {object}  dto.Response[any]
// @Failure      404  {object}  dto.Response[any]
// @Router       /admin/users/{id} [get]
func (ac *adminController) GetUser(c echo.Context) error {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	user, err := model.GetUserByID(ac.context.GetDB(), userID)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	resp := dto.Response[dto.UserProfileResponse]{
		Results: dto.NewUserProfileResponse(user),
	}

	return c.JSON(http.StatusOK, resp)
}

// UpdateUser updates a specific user (admin only)
// @Summary      Update user (admin)
// @Tags         admin
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "User ID"
// @Accept       json
// @Produce      json
// @Success      200  {object}  dto.Response[dto.UserProfileResponse]
// @Failure      400  {object}  dto.Response[any]
// @Failure      404  {object}  dto.Response[any]
// @Router       /admin/users/{id} [put]
func (ac *adminController) UpdateUser(c echo.Context) error {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	user, err := model.GetUserByID(ac.context.GetDB(), userID)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	var updateData dto.AdminUserUpdateData
	if err := c.Bind(&updateData); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	user.Name = updateData.Name
	user.Username = updateData.Username
	user.Admin = updateData.Admin
	user.Active = updateData.Active

	if updateData.Password != "" {
		if err := user.SetPassword(updateData.Password); err != nil {
			return renderApiError(c, http.StatusBadRequest, err)
		}
	}

	if err := user.Save(ac.context.GetDB()); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[dto.UserProfileResponse]{
		Results: dto.NewUserProfileResponse(user),
	}

	return c.JSON(http.StatusOK, resp)
}

// DeleteUser deletes a specific user (admin only)
// @Summary      Delete user (admin)
// @Tags         admin
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Param        id   path  int  true  "User ID"
// @Produce      json
// @Success      200  {object}  dto.Response[any]
// @Failure      400  {object}  dto.Response[any]
// @Failure      404  {object}  dto.Response[any]
// @Router       /admin/users/{id} [delete]
func (ac *adminController) DeleteUser(c echo.Context) error {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	user, err := model.GetUserByID(ac.context.GetDB(), userID)
	if err != nil {
		return renderApiError(c, http.StatusNotFound, err)
	}

	if err := user.Delete(ac.context.GetDB()); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[any]{
		Results: map[string]string{"message": "User deleted successfully"},
	}

	return c.JSON(http.StatusOK, resp)
}

// UpdateConfig updates application config (admin only)
// @Summary      Update config (admin)
// @Tags         admin
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Accept       json
// @Produce      json
// @Success      200  {object}  dto.Response[dto.AppInfoResponse]
// @Failure      400  {object}  dto.Response[any]
// @Failure      500  {object}  dto.Response[any]
// @Router       /admin/config [put]
func (ac *adminController) UpdateConfig(c echo.Context) error {
	var cnf container.Config

	if err := c.Bind(&cnf); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	if err := cnf.Save(ac.context.GetDB()); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	if err := ac.resetConfiguration(); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	cfg := ac.context.GetConfig()
	v := ac.context.GetVersion()

	resp := dto.Response[dto.AppInfoResponse]{
		Results: dto.AppInfoResponse{
			Version:              v.PrettyVersion(),
			VersionSha:           v.Sha,
			RegistrationDisabled: cfg.RegistrationDisabled,
			SocialsDisabled:      cfg.SocialsDisabled,
		},
	}

	return c.JSON(http.StatusOK, resp)
}
