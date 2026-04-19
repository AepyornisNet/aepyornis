package controller

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/container"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/model/dto"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

var (
	ErrLoginFailed     = errors.New("email or password incorrect")
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
// @Success      200  {object}  dto.Response[dto.UserProfileResponse]
// @Failure      400  {object}  dto.Response[any]
// @Failure      401  {object}  dto.Response[any]
// @Failure      500  {object}  dto.Response[any]
// @Router       /auth/signin [post]
func (ac *authController) SignIn(c echo.Context) error {
	var req dto.SigninRequest
	if err := c.Bind(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	if req.Email == "" || req.Password == "" {
		return renderApiError(c, http.StatusBadRequest, dto.ErrBadRequest)
	}

	storedUser, err := ac.context.UserRepo().GetByEmail(req.Email)
	if err != nil || !storedUser.ValidLogin(req.Password) {
		return renderApiError(c, http.StatusUnauthorized, ErrLoginFailed)
	}

	ac.context.GetSessionManager().Put(c.Request().Context(), "email", storedUser.Email)

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
// @Success      200  {object}  dto.Response[map[string]string]
// @Failure      500  {object}  dto.Response[any]
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
// @Success      201  {object}  dto.Response[map[string]string]
// @Failure      400  {object}  dto.Response[any]
// @Failure      403  {object}  dto.Response[any]
// @Failure      500  {object}  dto.Response[any]
// @Router       /auth/register [post]
func (ac *authController) Register(c echo.Context) error {
	if ac.context.GetConfig().RegistrationDisabled {
		return renderApiError(c, http.StatusForbidden, errors.New("registration is disabled"))
	}

	var req dto.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	if req.Email == "" || req.Password == "" {
		return renderApiError(c, http.StatusBadRequest, dto.ErrBadRequest)
	}

	if req.Name == "" {
		req.Name = req.Email
	}

	language := req.Language
	if language == "" {
		language = model.DefaultProfileLanguage
	}

	u := &model.User{
		UserData: model.UserData{
			Admin:  false,
			Active: false,
		},
		UserSecrets: model.UserSecrets{Email: req.Email},
	}

	u.ResetDefaults()
	u.Language = language

	profileUsername := strings.TrimSpace(req.Username)
	if profileUsername == "" {
		at := strings.IndexByte(req.Email, '@')
		if at > 0 {
			profileUsername = req.Email[:at]
		} else {
			profileUsername = req.Email
		}
	}
	u.Profile.Username = profileUsername
	u.Profile.DisplayName = req.Name

	if err := u.SetPassword(req.Password); err != nil {
		return renderApiError(c, http.StatusBadRequest, err)
	}

	if ac.context.GetConfig().ActivityPubActive {
		u.ActivityPub = true
		u.DefaultWorkoutVisibility = model.WorkoutVisibilityFollowers
		u.Profile.User = u

		if err := u.Profile.GenerateActivityPubKeys(false); err != nil {
			return renderApiError(c, http.StatusInternalServerError, err)
		}
	}

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

	claims["name"] = u.Email
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
