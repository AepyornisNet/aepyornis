package app

import (
	"net/http"

	"github.com/AepyornisNet/aepyornis/pkg/aputil"
	"github.com/AepyornisNet/aepyornis/pkg/model/dto"
	"github.com/labstack/echo/v4"
)

func (a *App) RequestingActorMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		actor, err := aputil.VerifyRequest(c.Request(), http.DefaultClient)
		if err != nil {
			a.logger.Warn("invalid ActivityPub request signature", "error", err)
			return a.renderAPIV2Error(c, http.StatusBadRequest, dto.ErrBadRequest)
		}

		if actor != nil {
			c.Set(aputil.RequestingActorContextKey, actor)
		}

		return next(c)
	}
}
