package controller

import (
	"net/http"

	"github.com/jovandeginste/workout-tracker/v2/pkg/container"
	"github.com/labstack/echo/v4"
)

type ApUserController interface {
	GetUser(c echo.Context) error
	Inbox(c echo.Context) error
	Outbox(c echo.Context) error
	Following(c echo.Context) error
	Followers(c echo.Context) error
}

type apUserController struct {
	context *container.Container
}

func NewApUserController(c *container.Container) ApUserController {
	return &apUserController{context: c}
}

func (ac *apUserController) GetUser(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

func (ac *apUserController) Inbox(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

func (ac *apUserController) Outbox(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

func (ac *apUserController) Following(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

func (ac *apUserController) Followers(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}
