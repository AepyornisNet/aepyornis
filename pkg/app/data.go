package app

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/invopop/ctxi18n"

	"github.com/labstack/echo/v4"
	"github.com/samber/do/v2"
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

	email, ok := claims["name"].(string)
	if !ok || strings.TrimSpace(email) == "" {
		return ErrInvalidJWTToken
	}

	dbUser, err := do.MustInvoke[repository.User](a.injector).GetByEmail(email)
	if err != nil {
		return ErrInvalidJWTToken
	}

	if !dbUser.IsActive() {
		return ErrInvalidJWTToken
	}

	a.setContextUser(c, dbUser)
	return nil
}

func (a *App) setContextUser(c echo.Context, user *model.User) {
	c.Set("user_language", user.Language)
	c.Set("user_info", user)
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
