package app

import (
	"strings"

	"github.com/AepyornisNet/aepyornis/pkg/model/dto"
	"github.com/AepyornisNet/aepyornis/pkg/repository"
	"github.com/labstack/echo/v4"
	"github.com/samber/do/v2"
)

// ValidateAPIKeyMiddleware validates the API key and attaches user info to the context.
func (a *App) ValidateAPIKeyMiddleware(key string, c echo.Context) (bool, error) {
	token := strings.TrimSpace(key)
	if len(token) >= 7 && strings.EqualFold(token[:7], "bearer ") {
		token = strings.TrimSpace(token[7:])
	}

	u, err := do.MustInvoke[repository.User](a.injector).GetByAPIKey(token)
	if err != nil {
		return false, dto.ErrInvalidAPIKey
	}

	if !u.IsActive() || !u.APIActive {
		return false, dto.ErrInvalidAPIKey
	}

	a.setContextUser(c, u)

	return true, nil
}
