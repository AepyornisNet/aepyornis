package app

import (
	"net/http"

	"github.com/jovandeginste/workout-tracker/v2/pkg/api"
	"github.com/labstack/echo/v4"
	geojson "github.com/paulmach/orb/geojson"
)

func (a *App) registerAPIV2HeatmapRoutes(apiGroup *echo.Group) {
	apiGroup.GET("/workouts/coordinates", a.apiV2WorkoutsCoordinatesHandler).Name = "api-v2-workouts-coordinates"
	apiGroup.GET("/workouts/centers", a.apiV2WorkoutsCentersHandler).Name = "api-v2-workouts-centers"
}

// apiV2WorkoutsCoordinatesHandler returns all coordinates of all workouts of the current user
// @Summary      Get workout coordinates
// @Tags         heatmap
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Success      200  {object}  api.Response[geojson.FeatureCollection]
// @Failure      500  {object}  api.Response[any]
// @Router       /workouts/coordinates [get]
func (a *App) apiV2WorkoutsCoordinatesHandler(c echo.Context) error {
	coords := geojson.NewFeatureCollection()

	db := a.db.Preload("Data").Preload("Data.Details")
	u := a.getCurrentUser(c)

	wos, err := u.GetWorkouts(db)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
	}

	for _, w := range wos {
		if !w.HasTracks() {
			continue
		}

		for _, p := range w.Data.Details.Points {
			f := geojson.NewFeature(p.ToOrbPoint())
			coords.Append(f)
		}
	}

	resp := api.Response[*geojson.FeatureCollection]{
		Results: coords,
	}

	return c.JSON(http.StatusOK, resp)
}

// apiV2WorkoutsCentersHandler returns the center of all workouts of the current user
// @Summary      Get workout centers
// @Tags         heatmap
// @Security     ApiKeyAuth
// @Security     ApiKeyQuery
// @Security     CookieAuth
// @Produce      json
// @Success      200  {object}  api.Response[geojson.FeatureCollection]
// @Failure      500  {object}  api.Response[any]
// @Router       /workouts/centers [get]
func (a *App) apiV2WorkoutsCentersHandler(c echo.Context) error {
	coords := geojson.NewFeatureCollection()
	u := a.getCurrentUser(c)
	db := a.db.Preload("Data").Preload("Data.Details")

	wos, err := u.GetWorkouts(db)
	if err != nil {
		return a.renderAPIV2Error(c, http.StatusInternalServerError, err)
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

		// Add structured popup data
		popupData := api.NewWorkoutPopupData(w)
		f.Properties["popup_data"] = popupData

		coords.Append(f)
	}

	resp := api.Response[*geojson.FeatureCollection]{
		Results: coords,
	}

	return c.JSON(http.StatusOK, resp)
}
