package container

import (
	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
	"github.com/jovandeginste/workout-tracker/v2/pkg/version"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type Container struct {
	db      *gorm.DB
	config  *database.Config
	version *version.Version
}

func NewContainer(db *gorm.DB, config *database.Config, version *version.Version) *Container {
	return &Container{db: db, config: config, version: version}
}

func (c *Container) GetDB() *gorm.DB {
	return c.db
}

func (c *Container) GetConfig() *database.Config {
	return c.config
}

func (c *Container) GetVersion() *version.Version {
	return c.version
}

func (c *Container) GetUser(e echo.Context) *database.User {
	d := e.Get("user_info")

	u, ok := d.(*database.User)
	if !ok {
		u = database.AnonymousUser()
	}

	u.SetContext(e.Request().Context())

	return u
}
