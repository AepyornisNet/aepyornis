package app

import (
	"strings"

	"github.com/jovandeginste/workout-tracker/v2/pkg/model/dto"
	"github.com/labstack/echo/v4"
)

// ValidateAPIKeyMiddleware validates the API key and attaches user info to the context.
func (a *App) ValidateAPIKeyMiddleware(key string, c echo.Context) (bool, error) {
	token := strings.TrimSpace(key)
	if len(token) >= 7 && strings.EqualFold(token[:7], "bearer ") {
		token = strings.TrimSpace(token[7:])
	}

	u, err := a.container.UserRepo().GetByAPIKey(token)
	if err != nil {
		return false, dto.ErrInvalidAPIKey
	}

	if !u.IsActive() || !u.Profile.APIActive {
		return false, dto.ErrInvalidAPIKey
	}

	c.Set("user_info", u)
	c.Set("user_language", u.Profile.Language)

	return true, nil
}
