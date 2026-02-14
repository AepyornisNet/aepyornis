package controller

import (
	"github.com/jovandeginste/workout-tracker/v2/pkg/model/dto"
	"github.com/labstack/echo/v4"
)

func renderApiError(c echo.Context, status int, err error) error {
	resp := dto.Response[any]{}
	resp.AddError(err)

	return c.JSON(status, resp)
}
