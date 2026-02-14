package app

import (
	"errors"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/labstack/echo/v4"
)

func (a *App) serveClientAppHandler(c echo.Context) error {
	if a.Assets == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "assets filesystem not configured")
	}

	requestPath := strings.TrimPrefix(c.Request().URL.Path, a.WebRoot())
	if requestPath == "" || requestPath == "/" {
		return a.serveClientAsset(c, "client/index.html")
	}

	normalized := path.Clean(requestPath)
	normalized = strings.TrimPrefix(normalized, "/")
	if normalized == "" || normalized == "." {
		return a.serveClientAsset(c, "client/index.html")
	}

	assetPath := path.Join("client", normalized)
	if _, err := fs.Stat(a.Assets, assetPath); err == nil {
		return a.serveClientAsset(c, assetPath)
	}

	return a.serveClientAsset(c, "client/index.html")
}

func (a *App) serveClientAsset(c echo.Context, assetPath string) error {
	data, err := fs.ReadFile(a.Assets, assetPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return echo.NewHTTPError(http.StatusNotFound)
		}

		return err
	}

	contentType := mime.TypeByExtension(path.Ext(assetPath))
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	return c.Blob(http.StatusOK, contentType, data)
}
