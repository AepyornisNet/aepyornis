package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/alexedwards/scs/gormstore"
	"github.com/alexedwards/scs/v2"
	"github.com/jovandeginste/workout-tracker/v2/pkg/container"
	"github.com/jovandeginste/workout-tracker/v2/pkg/geocoder"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model/dto"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/vgarvardt/gue/v6"

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

	sqlDB, err := a.db.DB()
	if err != nil {
		return fmt.Errorf("configure webserver: getting sql.DB from gorm: %w", err)
	}

	gc, err := gue.NewClient(sqlDB, gue.WithClientLogger(a.logger.With("module", "worker")))
	if err != nil {
		return fmt.Errorf("configure webserver: creating gue client: %w", err)
	}

	a.container = *container.NewContainer(a.db, a.Config, &a.Version, a.sessionManager, a.logger, gc)

	e.Use(session.LoadAndSave(a.sessionManager))
	e.Use(a.ContextValueMiddleware)
	e.Use(func(handlerFunc echo.HandlerFunc) echo.HandlerFunc {
		return func(context echo.Context) error {
			a.setContext(context)
			return handlerFunc(context)
		}
	})

	publicGroup := e.Group(a.WebRoot())
	publicGroup.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	}).Name = "health"
	a.apiV2Routes(publicGroup)
	a.registerActivityPubController(publicGroup)

	publicGroup.GET("/*", a.serveClientAppHandler).Name = "client-app"

	a.echo = e

	return nil
}

func (a *App) ValidateAdminMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		u := a.getCurrentUser(ctx)
		if u.IsAnonymous() || !u.IsActive() {
			log.Warn("User is not found")
			return a.renderAPIV2Error(ctx, http.StatusForbidden, dto.ErrNotAuthorized)
		}

		if !u.Admin {
			log.Warn("User is not an admin")
			return a.renderAPIV2Error(ctx, http.StatusForbidden, dto.ErrNotAuthorized)
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

// @title           Workout Tracker API
// @version         2.0
// @description     Workout Tracker HTTP API (v2).
// @BasePath        /api/v2
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @securityDefinitions.apikey ApiKeyQuery
// @in query
// @name api-key
// @securityDefinitions.apikey CookieAuth
// @in header
// @name Cookie
func (a *App) apiV2Routes(e *echo.Group) {
	// Public routes
	apiGroupPublic := e.Group("/api/v2")
	apiGroupPublic.GET("/app-info", a.apiV2AppInfoHandler).Name = "api-v2-app-info"
	a.registerAuthController(apiGroupPublic)

	apiGroup := e.Group("/api/v2")
	apiGroup.Use(echojwt.WithConfig(echojwt.Config{
		SigningKey:  a.Config.JWTSecret(),
		TokenLookup: "cookie:token",
		ErrorHandler: func(c echo.Context, err error) error {
			log.Warn(err.Error())

			r := dto.Response[any]{}
			r.AddError(err)
			r.AddError(dto.ErrNotAuthorized)

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

	a.registerUserController(apiGroup)
	a.registerWorkoutController(apiGroup)
	a.registerHeatmapController(apiGroup)
	a.registerRouteSegmentController(apiGroup)
	a.registerMeasurementController(apiGroup)
	a.registerEquipmentController(apiGroup)
	a.registerStatisticsController(apiGroup)
	a.registerProfileController(apiGroup)
	a.registerAdminController(apiGroup)

	apiGroup.POST("/lookup-address", a.apiV2LookupAddressHandler).Name = "lookup-address"
}

// apiV2LookupAddressHandler searches an address
// @Summary      Lookup address
// @Tags         lookup
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Param        location  query  string true "Free text address"
// @Success      200  {object}  dto.Response[[]string]
// @Failure      400  {object}  dto.Response[any]
// @Router       /lookup-address [post]
func (a *App) apiV2LookupAddressHandler(c echo.Context) error {
	q := c.Param("location")

	results, err := geocoder.Search(q)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusBadRequest, err)
	}

	return c.JSON(http.StatusOK, dto.Response[[]string]{
		Results: results,
	})
}

// apiV2AppInfoHandler returns application information
// @Summary      Get application info
// @Tags         meta
// @Produce      json
// @Success      200  {object}  dto.Response[dto.AppInfoResponse]
// @Router       /app-info [get]
func (a *App) apiV2AppInfoHandler(c echo.Context) error {
	resp := dto.Response[dto.AppInfoResponse]{
		Results: dto.AppInfoResponse{
			Version:              a.Version.PrettyVersion(),
			VersionSha:           a.Version.Sha,
			RegistrationDisabled: a.Config.RegistrationDisabled,
			SocialsDisabled:      a.Config.SocialsDisabled,
			AutoImportEnabled:    a.Config.AutoImportEnabled,
		},
	}

	return c.JSON(http.StatusOK, resp)
}

// renderAPIV2Error renders an API v2 error response
func (a *App) renderAPIV2Error(c echo.Context, status int, err error) error {
	resp := dto.Response[any]{}
	resp.AddError(err)
	return c.JSON(status, resp)
}
