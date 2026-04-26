package container

import (
	"context"
	"log/slog"

	"github.com/AepyornisNet/aepyornis/pkg/aputil"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/repository"
	"github.com/AepyornisNet/aepyornis/pkg/version"
	"github.com/alexedwards/scs/v2"
	"github.com/labstack/echo/v4"
	"github.com/samber/do/v2"
	"github.com/vgarvardt/gue/v6"
	"gorm.io/gorm"
)

type Container struct {
	injector do.Injector
}

func NewFromInjector(injector do.Injector) *Container {
	return &Container{
		injector: injector,
	}
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
	injector := do.New()
	do.ProvideValue(injector, db)
	do.ProvideValue(injector, config)
	do.ProvideValue(injector, v)
	do.ProvideValue(injector, sessionManager)
	do.ProvideValue(injector, logger)
	do.ProvideValue(injector, gueClient)
	do.ProvideValue(injector, repositories)

	return &Container{
		injector: injector,
	}
}

func (c *Container) Injector() do.Injector {
	return c.injector
}

func (c *Container) GetDB() *gorm.DB {
	return do.MustInvoke[*gorm.DB](c.injector)
}

func (c *Container) Logger() *slog.Logger {
	return do.MustInvoke[*slog.Logger](c.injector)
}

func (c *Container) GetConfig() *Config {
	return do.MustInvoke[*Config](c.injector)
}

func (c *Container) GetVersion() *version.Version {
	return do.MustInvoke[*version.Version](c.injector)
}

func (c *Container) GetSessionManager() *scs.SessionManager {
	return do.MustInvoke[*scs.SessionManager](c.injector)
}

func (c *Container) GetGueClient() *gue.Client {
	return do.MustInvoke[*gue.Client](c.injector)
}

func (c *Container) GetRepositories() *repository.Repositories {
	return do.MustInvoke[*repository.Repositories](c.injector)
}

func (c *Container) APOutboxRepo() repository.APOutbox {
	repositories := c.GetRepositories()
	if repositories == nil {
		return nil
	}

	return repositories.APOutbox
}

func (c *Container) APStatusRepo() repository.APStatus {
	repositories := c.GetRepositories()
	if repositories == nil {
		return nil
	}

	return repositories.APStatus
}

func (c *Container) APStatusDeliveryRepo() repository.APStatusDelivery {
	repositories := c.GetRepositories()
	if repositories == nil {
		return nil
	}

	return repositories.APStatusDelivery
}

func (c *Container) FollowerRepo() repository.Follower {
	repositories := c.GetRepositories()
	if repositories == nil {
		return nil
	}

	return repositories.Follower
}

func (c *Container) EquipmentRepo() repository.Equipment {
	repositories := c.GetRepositories()
	if repositories == nil {
		return nil
	}

	return repositories.Equipment
}

func (c *Container) RouteSegmentRepo() repository.RouteSegment {
	repositories := c.GetRepositories()
	if repositories == nil {
		return nil
	}

	return repositories.RouteSegment
}

func (c *Container) MeasurementRepo() repository.Measurement {
	repositories := c.GetRepositories()
	if repositories == nil {
		return nil
	}

	return repositories.Measurement
}

func (c *Container) WorkoutRepo() repository.Workout {
	repositories := c.GetRepositories()
	if repositories == nil {
		return nil
	}

	return repositories.Workout
}

func (c *Container) WorkoutLikeRepo() repository.WorkoutLike {
	repositories := c.GetRepositories()
	if repositories == nil {
		return nil
	}

	return repositories.WorkoutLike
}

func (c *Container) WorkoutReplyRepo() repository.WorkoutReply {
	repositories := c.GetRepositories()
	if repositories == nil {
		return nil
	}

	return repositories.WorkoutReply
}

func (c *Container) UserRepo() repository.User {
	repositories := c.GetRepositories()
	if repositories == nil {
		return nil
	}

	return repositories.User
}

func (c *Container) Enqueue(ctx context.Context, j *gue.Job) error {
	return c.GetGueClient().Enqueue(ctx, j)
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

func (c *Container) GetApUser(e echo.Context) *aputil.UserActor {
	d := e.Get("user_ap_actor")

	a, ok := d.(*aputil.UserActor)
	if !ok {
		return nil
	}

	return a
}
