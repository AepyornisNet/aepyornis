package controller

import (
	"errors"
	"net/http"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/model/dto"
	"github.com/labstack/echo/v5"
)

const activityPubContentType = `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`

const errorCodeWorkoutAlreadyExists = "workout_already_exists"

func apiErrorCode(err error) string {
	if errors.Is(err, model.ErrWorkoutAlreadyExists) {
		return errorCodeWorkoutAlreadyExists
	}

	return ""
}

func renderApiError(c *echo.Context, status int, err error) error {
	resp := dto.Response[any]{}
	resp.AddContextError(c.Request().Context(), err)

	if code := apiErrorCode(err); code != "" {
		resp.ErrorCodes = append(resp.ErrorCodes, code)
	}

	return c.JSON(status, resp)
}

func renderActivityPubResponse(c *echo.Context, payload []byte) error {
	return c.Blob(http.StatusOK, activityPubContentType, payload)
}
