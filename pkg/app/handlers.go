package app

import (
	"net/http"
	"os"

	"github.com/invopop/ctxi18n/i18n"
	"github.com/labstack/echo/v4"
)

func (a *App) serveClientAppHandler(c echo.Context) error {
	requestPath := c.Request().URL.Path
	if requestPath == "" || requestPath == "/" {
		return c.File("assets/client/index.html")
	}
	
	filePath := "assets/client" + requestPath
	if _, err := os.Stat(filePath); err == nil {
		return c.File(filePath)
	}
	
	return c.File("assets/client/index.html")
}

func (a *App) redirectWithError(c echo.Context, target string, err error) error {
	if err != nil {
		a.addErrorT(c, "alerts.something_wrong", i18n.M{"message": err.Error()})
	}

	return c.Redirect(http.StatusFound, target)
}
