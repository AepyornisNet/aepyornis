package controller

import (
	"github.com/AepyornisNet/aepyornis/pkg/aputil"
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

func currentAPUser(c echo.Context) *aputil.UserActor {
	d := c.Get("user_ap_actor")

	a, ok := d.(*aputil.UserActor)
	if !ok {
		return nil
	}

	return a
}
