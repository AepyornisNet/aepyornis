package controller

import (
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jovandeginste/workout-tracker/v2/pkg/container"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model/dto"
	"github.com/labstack/echo/v4"
)

var (
	ErrLoginFailed     = errors.New("username or password incorrect")
	ErrInvalidJWTToken = errors.New("invalid JWT token")
)

type AuthController interface {
	SignIn(c echo.Context) error
	SignOut(c echo.Context) error
	Register(c echo.Context) error
}

type authController struct {
	context *container.Container
}

func NewAuthController(c *container.Container) AuthController {
	return &authController{context: c}
}

// SignIn authenticates user credentials and sets auth cookies
// @Summary      Sign in
// @Tags         auth
// @Accept       json
// @Produce      json
// @Success      200  {object}  api.Response[dto.UserProfileResponse]
// @Failure      400  {object}  api.Response[any]
// @Failure      401  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /auth/signin [post]
func (ac *authController) SignIn(c echo.Context) error {
	var req dto.SigninRequest
	if err := c.Bind(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	if req.Username == "" || req.Password == "" {
		return renderApiError(c, http.StatusBadRequest, dto.ErrBadRequest)
	}

	storedUser, err := model.GetUser(ac.context.GetDB(), req.Username)
	if err != nil || !storedUser.ValidLogin(req.Password) {
		return renderApiError(c, http.StatusUnauthorized, ErrLoginFailed)
	}

	ac.context.GetSessionManager().Put(c.Request().Context(), "username", storedUser.Username)

	if err := ac.createToken(storedUser, c); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[dto.UserProfileResponse]{
		Results: dto.NewUserProfileResponse(storedUser),
	}

	return c.JSON(http.StatusOK, resp)
}

// SignOut removes authentication cookie and server-side session
// @Summary      Sign out
// @Tags         auth
// @Produce      json
// @Success      200  {object}  api.Response[map[string]string]
// @Failure      500  {object}  api.Response[any]
// @Router       /auth/signout [post]
func (ac *authController) SignOut(c echo.Context) error {
	ac.clearTokenCookie(c)

	if err := ac.context.GetSessionManager().Destroy(c.Request().Context()); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[map[string]string]{
		Results: map[string]string{"message": "signed out"},
	}

	return c.JSON(http.StatusOK, resp)
}

// Register creates a new inactive user account
// @Summary      Register account
// @Tags         auth
// @Accept       json
// @Produce      json
// @Success      201  {object}  api.Response[map[string]string]
// @Failure      400  {object}  api.Response[any]
// @Failure      403  {object}  api.Response[any]
// @Failure      500  {object}  api.Response[any]
// @Router       /auth/register [post]
func (ac *authController) Register(c echo.Context) error {
	if ac.context.GetConfig().RegistrationDisabled {
		return renderApiError(c, http.StatusForbidden, errors.New("registration is disabled"))
	}

	var req dto.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	if req.Username == "" || req.Password == "" {
		return renderApiError(c, http.StatusBadRequest, dto.ErrBadRequest)
	}

	if req.Name == "" {
		req.Name = req.Username
	}

	u := &model.User{
		UserData: model.UserData{
			Username: req.Username,
			Name:     req.Name,
			Admin:    false,
			Active:   false,
		},
	}

	if err := u.SetPassword(req.Password); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	u.Profile.Theme = "browser"
	u.Profile.TotalsShow = model.WorkoutTypeRunning
	u.Profile.Language = "browser"
	u.Profile.PreferredUnits.SpeedRaw = "km/h"
	u.Profile.PreferredUnits.DistanceRaw = "km"
	u.Profile.PreferredUnits.ElevationRaw = "m"
	u.Profile.PreferredUnits.WeightRaw = "kg"
	u.Profile.PreferredUnits.HeightRaw = "cm"

	if err := u.Create(ac.context.GetDB()); err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	resp := dto.Response[map[string]string]{
		Results: map[string]string{
			"message": "Your account has been created but needs to be activated",
		},
	}

	return c.JSON(http.StatusCreated, resp)
}

func (ac *authController) createToken(u *model.User, c echo.Context) error {
	token := jwt.New(jwt.SigningMethodHS256)

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return ErrInvalidJWTToken
	}

	exp := time.Now().Add(time.Hour * 24 * 10)

	claims["name"] = u.Username
	claims["exp"] = exp.Unix()

	t, err := token.SignedString(ac.context.GetConfig().JWTSecret())
	if err != nil {
		return err
	}

	ac.setTokenCookie(t, exp, c)

	return nil
}

func (ac *authController) setTokenCookie(t string, exp time.Time, c echo.Context) {
	cookie := new(http.Cookie)
	cookie.Path = "/"
	cookie.HttpOnly = true
	cookie.Name = "token"
	cookie.Value = t
	cookie.Expires = exp

	c.SetCookie(cookie)
}

func (ac *authController) clearTokenCookie(c echo.Context) {
	exp := time.Now()
	ac.setTokenCookie("", exp, c)
}
