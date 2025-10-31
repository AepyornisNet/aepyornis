package app

import (
	"context"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/alexedwards/scs/gormstore"
	"github.com/alexedwards/scs/v2"
	"github.com/invopop/ctxi18n"
	"github.com/jovandeginste/workout-tracker/v2/pkg/api"
	"github.com/jovandeginste/workout-tracker/v2/pkg/geocoder"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"

	slogecho "github.com/samber/slog-echo"

	session "github.com/spazzymoto/echo-scs-session"
)

func (a *App) WebRoot() string {
	root := path.Join("/", a.Config.WebRoot)
	return strings.TrimSuffix(root, "/")
}

func newEcho(logger *slog.Logger) *echo.Echo {
	e := echo.New()

	e.HideBanner = true
	e.HidePort = true

	e.Use(slogecho.New(logger.With("module", "webserver")))
	e.Use(middleware.Recover())
	e.Use(middleware.Secure())
	e.Use(middleware.CORS())
	e.Use(middleware.Gzip())
	e.Pre(middleware.RemoveTrailingSlash())
	e.Pre(middleware.MethodOverrideWithConfig(middleware.MethodOverrideConfig{
		Getter: middleware.MethodFromHeader(echo.HeaderXHTTPMethodOverride),
	}))

	return e
}

func (a *App) ConfigureWebserver() error {
	var err error

	e := newEcho(a.rawLogger)
	e.Debug = a.Config.Debug

	a.sessionManager = scs.New()
	a.sessionManager.Cookie.Path = "/"
	a.sessionManager.Cookie.HttpOnly = true
	a.sessionManager.Lifetime = 24 * time.Hour

	if a.sessionManager.Store, err = gormstore.New(a.db); err != nil {
		return err
	}

	e.Use(session.LoadAndSave(a.sessionManager))
	e.Use(a.ContextValueMiddleware)
	e.Use(func(handlerFunc echo.HandlerFunc) echo.HandlerFunc {
		return func(context echo.Context) error {
			a.setContext(context)
			return handlerFunc(context)
		}
	})

	publicGroup := e.Group(a.WebRoot())
	a.apiRoutes(publicGroup)
	a.apiV2Routes(publicGroup)

	if a.AssetDir != "" {
		publicGroup.Static("/assets", a.AssetDir)
	} else {
		publicGroup.StaticFS("/assets", a.Assets)
	}

	publicGroup.GET("/assets", func(c echo.Context) error {
		return c.Redirect(http.StatusFound, a.echo.Reverse("dashboard"))
	}).Name = "assets"

	authGroup := publicGroup.Group("/auth")
	authGroup.POST("/signin", a.userSigninHandler).Name = "user-signin"
	authGroup.POST("/register", a.userRegisterHandler).Name = "user-register"
	authGroup.GET("/signout", a.userSignoutHandler).Name = "user-signout"

	a.echo = e

	return nil
}

func (a *App) ValidateAdminMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		u := a.getCurrentUser(ctx)
		if u.IsAnonymous() || !u.IsActive() {
			log.Warn("User is not found")
			return ctx.Redirect(http.StatusFound, a.echo.Reverse("user-signout"))
		}

		if !u.Admin {
			log.Warn("User is not an admin")
			return ctx.Redirect(http.StatusFound, a.echo.Reverse("dashboard"))
		}

		return next(ctx)
	}
}

func (a *App) ValidateUserMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		if err := a.setUserFromContext(ctx); err != nil {
			a.logger.Warn("error validating user", "error", err.Error())
			return ctx.Redirect(http.StatusFound, a.echo.Reverse("user-signout"))
		}

		lctx, _ := ctxi18n.WithLocale(ctx.Request().Context(), langFromContextString(ctx))
		if lctx != nil {
			ctx.SetRequest(ctx.Request().WithContext(lctx))
		}

		return next(ctx)
	}
}

// extend echo.Context
type contextValue struct {
	echo.Context
}

func (c contextValue) Get(key string) any {
	if val := c.Context.Get(key); val != nil {
		return val
	}

	return c.Request().Context().Value(key)
}

func (c contextValue) Set(key string, val any) {
	// we're replacing the whole Request in echo.Context
	// with a copied request that has the updated context value
	c.SetRequest(
		c.Request().WithContext(
			context.WithValue(c.Request().Context(), key, val),
		),
	)
	c.Context.Set(key, val)
}

func (a *App) ContextValueMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// instead of passing next(c) as you usually would,
		// you return it with the extended version
		return next(contextValue{c})
	}
}

func (a *App) apiV2Routes(e *echo.Group) {
	// Public routes
	apiGroupPublic := e.Group("/api/v2")
	apiGroupPublic.GET("/app-info", a.apiV2AppInfoHandler).Name = "api-v2-app-info"

	apiGroup := e.Group("/api/v2")
	apiGroup.Use(echojwt.WithConfig(echojwt.Config{
		SigningKey:  a.jwtSecret(),
		TokenLookup: "cookie:token",
		ErrorHandler: func(c echo.Context, err error) error {
			log.Warn(err.Error())

			r := api.Response[any]{}
			r.AddError(err)
			r.AddError(api.ErrNotAuthorized)

			return c.JSON(http.StatusForbidden, r)
		},
		Skipper: func(ctx echo.Context) bool {
			if ctx.Request().Header.Get("Authorization") != "" {
				return true
			}

			if ctx.Request().URL.Query().Get("api-key") != "" {
				return true
			}

			return false
		},
		SuccessHandler: func(ctx echo.Context) {
			if err := a.setUserFromContext(ctx); err != nil {
				a.logger.Warn("error validating user", "error", err.Error())
				return
			}
		},
	}))

	apiGroup.Use(middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
		Validator: a.ValidateAPIKeyMiddleware,
		KeyLookup: "query:api-key",
		Skipper: func(ctx echo.Context) bool {
			return ctx.Request().URL.Query().Get("api-key") == ""
		},
	}))
	apiGroup.Use(middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
		Validator: a.ValidateAPIKeyMiddleware,
		Skipper: func(ctx echo.Context) bool {
			return ctx.Request().Header.Get("Authorization") == ""
		},
	}))

	a.registerAPIV2UserRoutes(apiGroup)
	a.registerAPIV2WorkoutRoutes(apiGroup, apiGroupPublic)
	a.registerAPIV2HeatmapRoutes(apiGroup)
	a.registerAPIV2RouteSegmentRoutes(apiGroup)
	a.registerAPIV2MeasurementRoutes(apiGroup)
	a.registerAPIV2EquipmentRoutes(apiGroup)
	a.registerAPIV2StatisticsRoutes(apiGroup)
	a.registerAPIV2ProfileRoutes(apiGroup)
	a.registerAPIV2AdminRoutes(apiGroup)

	apiGroup.POST("/lookup-address", a.apiV2LookupAddressHandler).Name = "lookup-address"
}

func (a *App) apiV2LookupAddressHandler(c echo.Context) error {
	q := c.Param("location")

	results, err := geocoder.Search(q)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	return c.JSON(http.StatusOK, api.Response[[]string]{
		Results: results,
	})
}

// apiV2AppInfoHandler returns application information
func (a *App) apiV2AppInfoHandler(c echo.Context) error {
	resp := api.Response[api.AppInfoResponse]{
		Results: api.AppInfoResponse{
			Version:              a.Version.PrettyVersion(),
			VersionSha:           a.Version.Sha,
			RegistrationDisabled: a.Config.RegistrationDisabled,
			SocialsDisabled:      a.Config.SocialsDisabled,
		},
	}

	return c.JSON(http.StatusOK, resp)
}

// renderAPIV2Error renders an API v2 error response
func (a *App) renderAPIV2Error(c echo.Context, status int, err error) error {
	resp := api.Response[any]{}
	resp.AddError(err)
	return c.JSON(status, resp)
}
