package controller

import (
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/labstack/echo/v4"
)

func currentUser(c echo.Context) *model.User {
	d := c.Get("user_info")

	u, ok := d.(*model.User)
	if !ok {
		u = model.AnonymousUser()
	}

	u.SetContext(c.Request().Context())

	return u
}
