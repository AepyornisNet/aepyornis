package apptranslations

import (
	"embed"
	"io/fs"

	"github.com/labstack/echo/v5"
)

//go:embed *
var embedded embed.FS

func FS() fs.FS {
	return echo.MustSubFS(embedded, "")
}
