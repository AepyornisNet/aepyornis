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

func hasMultipartFormValue(form *multipart.Form, c *echo.Context, key string) bool {
	if form != nil && form.Value != nil {
		if _, ok := form.Value[key]; ok {
			return true
		}
	}
	if req := c.Request(); req != nil {
		if req.Form != nil && req.Form.Has(key) {
			return true
		}
		if req.PostForm != nil && req.PostForm.Has(key) {
			return true
		}
	}
	return false
}
