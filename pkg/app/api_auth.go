package app

import (
	"fmt"
	"strings"

	"github.com/AepyornisNet/aepyornis/pkg/model/dto"
	"github.com/AepyornisNet/aepyornis/pkg/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/samber/do/v2"
)

// ValidateAPIKeyMiddleware validates the API key or JWT bearer token and attaches user info to the context.
func (a *App) ValidateAPIKeyMiddleware(c *echo.Context, key string, source middleware.ExtractorSource) (bool, error) {
	token := strings.TrimSpace(key)
	if len(token) >= 7 && strings.EqualFold(token[:7], "bearer ") {
		token = strings.TrimSpace(token[7:])
	}

	if token == "" {
		return false, dto.ErrInvalidAPIKey
	}

	// 1. Try validating as JWT token
	jwtToken, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return a.Config.JWTSecret(), nil
	})

	if err == nil && jwtToken != nil && jwtToken.Valid {
		if claims, ok := jwtToken.Claims.(jwt.MapClaims); ok {
			if email, ok := claims["name"].(string); ok && strings.TrimSpace(email) != "" {
				u, err := do.MustInvoke[repository.User](a.injector).GetByEmail(email)
				if err == nil && u.IsActive() {
					a.setContextUser(c, u)
					return true, nil
				}
			}
		}
	}

	// 2. Fallback: validate as API key
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
