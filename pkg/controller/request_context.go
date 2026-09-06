package controller

import (
	"mime/multipart"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/labstack/echo/v5"
)

func currentUser(c *echo.Context) *model.User {
	d := c.Get("user_info")

	u, ok := d.(*model.User)
	if !ok {
		u = model.AnonymousUser()
	}

	u.SetContext(c.Request().Context())

	return u
}

func multipartFormValue(form *multipart.Form, c *echo.Context, key string) string {
	if form != nil && form.Value != nil {
		if vs, ok := form.Value[key]; ok && len(vs) > 0 {
			return vs[0]
		}
	}
	return c.FormValue(key)
}
