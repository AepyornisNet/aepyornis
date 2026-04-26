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
) *Container {
	injector := do.New()
	repository.Package(injector)
	do.ProvideValue(injector, db)
	do.ProvideValue(injector, config)
	do.ProvideValue(injector, v)
	do.ProvideValue(injector, sessionManager)
	do.ProvideValue(injector, logger)
	do.ProvideValue(injector, gueClient)

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

func (c *Container) APOutboxRepo() repository.APOutbox {
	return do.MustInvoke[repository.APOutbox](c.injector)
}

func (c *Container) APStatusRepo() repository.APStatus {
	return do.MustInvoke[repository.APStatus](c.injector)
}

func (c *Container) APStatusDeliveryRepo() repository.APStatusDelivery {
	return do.MustInvoke[repository.APStatusDelivery](c.injector)
}

func (c *Container) FollowerRepo() repository.Follower {
	return do.MustInvoke[repository.Follower](c.injector)
}

func (c *Container) EquipmentRepo() repository.Equipment {
	return do.MustInvoke[repository.Equipment](c.injector)
}

func (c *Container) RouteSegmentRepo() repository.RouteSegment {
	return do.MustInvoke[repository.RouteSegment](c.injector)
}

func (c *Container) MeasurementRepo() repository.Measurement {
	return do.MustInvoke[repository.Measurement](c.injector)
}

func (c *Container) WorkoutRepo() repository.Workout {
	return do.MustInvoke[repository.Workout](c.injector)
}

func (c *Container) WorkoutLikeRepo() repository.WorkoutLike {
	return do.MustInvoke[repository.WorkoutLike](c.injector)
}

func (c *Container) WorkoutReplyRepo() repository.WorkoutReply {
	return do.MustInvoke[repository.WorkoutReply](c.injector)
}

func (c *Container) UserRepo() repository.User {
	return do.MustInvoke[repository.User](c.injector)
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
