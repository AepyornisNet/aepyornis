package app

import (
	"net/http"

	"github.com/AepyornisNet/aepyornis/pkg/aputil"
	"github.com/AepyornisNet/aepyornis/pkg/model/dto"
	"github.com/AepyornisNet/aepyornis/pkg/service"
	"github.com/labstack/echo/v4"
	"github.com/samber/do/v2"
)

func (a *App) RequestingActorMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		actor, err := do.MustInvoke[service.ActivityPubRequestService](a.injector).VerifyRequest(c.Request())
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
