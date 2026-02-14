package app

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	"github.com/invopop/ctxi18n"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"

	"github.com/labstack/echo/v4"
)

var ErrInvalidJWTToken = errors.New("invalid JWT token")

func (a *App) setContext(ctx echo.Context) {
	ctx.Set("version", &a.Version)
	ctx.Set("config", a.Config)
	ctx.Set("echo", a.echo)
	ctx.Set("sessionManager", a.sessionManager)

	lctx, _ := ctxi18n.WithLocale(ctx.Request().Context(), langFromContextString(ctx))
	if lctx == nil {
		lctx, _ = ctxi18n.WithLocale(ctx.Request().Context(), "en")
	}

	ctx.SetRequest(ctx.Request().WithContext(lctx))
}

func (a *App) setUserFromContext(ctx echo.Context) error {
	if err := a.setUser(ctx); err != nil {
		return fmt.Errorf("error validating user: %w", err)
	}

	u := a.getCurrentUser(ctx)
	if u.IsAnonymous() || !u.IsActive() {
		return errors.New("user not found or active")
	}

	return nil
}

func (a *App) setUser(c echo.Context) error {
	token, ok := c.Get("user").(*jwt.Token)
	if !ok {
		return ErrInvalidJWTToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return ErrInvalidJWTToken
	}

	dbUser, err := model.GetUser(a.db, claims["name"].(string))
	if err != nil {
		return err
	}

	if !dbUser.IsActive() {
		return ErrInvalidJWTToken
	}

	c.Set("user_language", dbUser.Profile.Language)
	c.Set("user_info", dbUser)

	return nil
}

func (a *App) getCurrentUser(c echo.Context) *model.User {
	d := c.Get("user_info")

	u, ok := d.(*model.User)
	if !ok {
		u = model.AnonymousUser()
	}

	u.SetContext(c.Request().Context())

	return u
}
