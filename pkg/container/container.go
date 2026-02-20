package container

import (
	"context"
	"log/slog"

	"github.com/alexedwards/scs/v2"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/jovandeginste/workout-tracker/v2/pkg/version"
	"github.com/labstack/echo/v4"
	"github.com/vgarvardt/gue/v6"
	"gorm.io/gorm"
)

type Container struct {
	db             *gorm.DB
	config         *Config
	version        *version.Version
	sessionManager *scs.SessionManager
	logger         *slog.Logger
	gueClient      *gue.Client
}

func NewContainer(
	db *gorm.DB,
	config *Config,
	v *version.Version,
	sessionManager *scs.SessionManager,
	logger *slog.Logger,
	gueClient *gue.Client,
) *Container {
	return &Container{
		db:             db,
		config:         config,
		version:        v,
		sessionManager: sessionManager,
		logger:         logger,
		gueClient:      gueClient,
	}
}

func (c *Container) GetDB() *gorm.DB {
	return c.db
}

func (c *Container) Logger() *slog.Logger {
	return c.logger
}

func (c *Container) GetConfig() *Config {
	return c.config
}

func (c *Container) GetVersion() *version.Version {
	return c.version
}

func (c *Container) GetSessionManager() *scs.SessionManager {
	return c.sessionManager
}

func (c *Container) GetGueClient() *gue.Client {
	return c.gueClient
}

func (c *Container) Enqueue(ctx context.Context, j *gue.Job) error {
	return c.gueClient.Enqueue(ctx, j)
}

func (c *Container) GetUser(e echo.Context) *model.User {
	d := e.Get("user_info")

	u, ok := d.(*model.User)
	if !ok {
		u = model.AnonymousUser()
	}

	u.SetContext(e.Request().Context())

	return u
}
