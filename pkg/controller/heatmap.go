package controller

import (
	"net/http"

	"github.com/jovandeginste/workout-tracker/v2/pkg/model/dto"
	"github.com/jovandeginste/workout-tracker/v2/pkg/container"
	"github.com/labstack/echo/v4"
	geojson "github.com/paulmach/orb/geojson"
)

type HeatmapController interface {
	GetWorkoutCoordinates(c echo.Context) error
	GetWorkoutCenters(c echo.Context) error
}

type heatmapController struct {
	context *container.Container
}

func NewHeatmapController(c *container.Container) HeatmapController {
	return &heatmapController{context: c}
}

// GetWorkoutCoordinates returns all coordinates of all workouts of the current user
// @Summary      Get workout coordinates
// @Tags         heatmap
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Success      200  {object}  api.Response[[][]float64]
// @Failure      500  {object}  api.Response[any]
// @Router       /workouts/coordinates [get]
func (hc *heatmapController) GetWorkoutCoordinates(c echo.Context) error {
	coords := [][]float64{}

	db := hc.context.GetDB().Preload("Data").Preload("Data.Details")
	u := hc.context.GetUser(c)

	wos, err := u.GetWorkouts(db)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	for _, w := range wos {
		if !w.HasTracks() {
			continue
		}

		for _, p := range w.Data.Details.Points {
			coords = append(coords, []float64{p.Lat, p.Lng, 1})
		}
	}

	resp := dto.Response[[][]float64]{
		Results: coords,
	}

	return c.JSON(http.StatusOK, resp)
}

// GetWorkoutCenters returns the center of all workouts of the current user
// @Summary      Get workout centers
// @Tags         heatmap
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Success      200  {object}  api.Response[geojson.FeatureCollection]
// @Failure      500  {object}  api.Response[any]
// @Router       /workouts/centers [get]
func (hc *heatmapController) GetWorkoutCenters(c echo.Context) error {
	coords := geojson.NewFeatureCollection()
	u := hc.context.GetUser(c)
	db := hc.context.GetDB().Preload("Data").Preload("Data.Details")

	wos, err := u.GetWorkouts(db)
	if err != nil {
		return renderApiError(c, http.StatusInternalServerError, err)
	}

	for _, w := range wos {
		if w.Data == nil {
			continue
		}

		p := w.Data.Center
		if p.IsZero() {
			continue
		}

		f := geojson.NewFeature(p.ToOrbPoint())
		f.Properties["popup_data"] = dto.NewWorkoutPopupData(w)

		coords.Append(f)
	}

	resp := dto.Response[*geojson.FeatureCollection]{
		Results: coords,
	}

	return c.JSON(http.StatusOK, resp)
}
