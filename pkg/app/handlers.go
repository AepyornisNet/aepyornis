package app

import (
	"errors"
	"net/http"

	"github.com/a-h/templ"
	"github.com/invopop/ctxi18n/i18n"
	"github.com/jovandeginste/workout-tracker/v2/pkg/geocoder"
	"github.com/jovandeginste/workout-tracker/v2/views/partials"
	"github.com/labstack/echo/v4"
)

var ErrUserNotFound = errors.New("user not found")

func (a *App) redirectWithError(c echo.Context, target string, err error) error {
	if err != nil {
		a.addErrorT(c, "alerts.something_wrong", i18n.M{"message": err.Error()})
	}

	return c.Redirect(http.StatusFound, target)
}

func (a *App) lookupAddressHandler(c echo.Context) error {
	q := c.FormValue("location")

	results, err := geocoder.Search(q)
	if err != nil {
		a.addErrorT(c, "alerts.something_wrong", i18n.M{"message": err.Error()})
	}

	return Render(c, http.StatusOK, partials.AddressResults(results))
}

func Render(ctx echo.Context, statusCode int, t templ.Component) error {
	buf := templ.GetBuffer()
	defer templ.ReleaseBuffer(buf)

	if err := t.Render(ctx.Request().Context(), buf); err != nil {
		return err
	}

	return ctx.HTML(statusCode, buf.String())
}
