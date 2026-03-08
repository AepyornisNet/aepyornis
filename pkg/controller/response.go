package controller

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model/dto"
	"github.com/labstack/echo/v4"
)

const activityPubContentType = `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`

const errorCodeWorkoutAlreadyExists = "workout_already_exists"

func apiErrorCode(err error) string {
	if errors.Is(err, model.ErrWorkoutAlreadyExists) {
		return errorCodeWorkoutAlreadyExists
	}

	return ""
}

func renderApiError(c echo.Context, status int, err error) error {
	resp := dto.Response[any]{}
	resp.AddError(err)

	if code := apiErrorCode(err); code != "" {
		resp.ErrorCodes = append(resp.ErrorCodes, code)
	}

	return c.JSON(status, resp)
}

func renderActivityPubResponse(c echo.Context, payload []byte) error {
	return c.Blob(http.StatusOK, activityPubContentType, payload)
}

// wantsActivityPub returns true when the request Accept header indicates
// that the client expects an ActivityPub response (application/activity+json
// or application/ld+json).  Browsers typically send text/html or */*.
func wantsActivityPub(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	if accept == "" {
		return false
	}

	for _, part := range strings.Split(accept, ",") {
		mediaType := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
		switch mediaType {
		case "application/activity+json", "application/ld+json":
			return true
		}
	}

	return false
}
