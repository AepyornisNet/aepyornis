package container

import (
	"context"
	"log/slog"

	"github.com/alexedwards/scs/v2"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/jovandeginste/workout-tracker/v2/pkg/repository"
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
	repositories   *repository.Repositories
}

func NewContainer(
	db *gorm.DB,
	config *Config,
	v *version.Version,
	sessionManager *scs.SessionManager,
	logger *slog.Logger,
	gueClient *gue.Client,
	repositories *repository.Repositories,
) *Container {
	return &Container{
		db:             db,
		config:         config,
		version:        v,
		sessionManager: sessionManager,
		logger:         logger,
		gueClient:      gueClient,
		repositories:   repositories,
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

func (c *Container) GetRepositories() *repository.Repositories {
	return c.repositories
}

func (c *Container) APOutboxRepo() repository.APOutbox {
	if c.repositories == nil {
		return nil
	}

	return c.repositories.APOutbox
}

func (c *Container) APOutboxDeliveryRepo() repository.APOutboxDelivery {
	if c.repositories == nil {
		return nil
	}

	return c.repositories.APOutboxDelivery
}

func (c *Container) FollowerRepo() repository.Follower {
	if c.repositories == nil {
		return nil
	}

	return c.repositories.Follower
}

func (c *Container) EquipmentRepo() repository.Equipment {
	if c.repositories == nil {
		return nil
	}

	return c.repositories.Equipment
}

func (c *Container) RouteSegmentRepo() repository.RouteSegment {
	if c.repositories == nil {
		return nil
	}

	return c.repositories.RouteSegment
}

func (c *Container) MeasurementRepo() repository.Measurement {
	if c.repositories == nil {
		return nil
	}

	return c.repositories.Measurement
}

func (c *Container) WorkoutRepo() repository.Workout {
	if c.repositories == nil {
		return nil
	}

	return c.repositories.Workout
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
