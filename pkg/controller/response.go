package controller

import (
	"github.com/jovandeginste/workout-tracker/v2/pkg/model/dto"
	"github.com/labstack/echo/v4"
)

const activityPubContentType = `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`

func renderApiError(c echo.Context, status int, err error) error {
	resp := dto.Response[any]{}
	resp.AddError(err)

	return c.JSON(status, resp)
}

func renderActivityPubResponse(c echo.Context, status int, payload []byte) error {
	return c.Blob(status, activityPubContentType, payload)
}
