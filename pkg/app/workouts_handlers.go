package app

import (
	"net/http"

	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
	"github.com/jovandeginste/workout-tracker/v2/views/workouts"
	"github.com/labstack/echo/v4"
	"github.com/spf13/cast"
)

func (a *App) addRoutesWorkouts(e *echo.Group) {
	workoutsGroup := e.Group("/workouts")
	workoutsGroup.POST("", a.addWorkout).Name = "workouts-create"
	workoutsGroup.POST("/:id", a.workoutsUpdateHandler).Name = "workout-update"
	workoutsGroup.POST("/:id/route-segment", a.workoutsCreateRouteSegmentFromWorkoutHandler).Name = "workout-route-segment-create"
}

func (a *App) workoutsCreateRouteSegmentFromWorkoutHandler(c echo.Context) error {
	id, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return a.redirectWithError(c, a.echo.Reverse("workouts"), err)
	}

	workout, err := database.GetWorkoutDetails(a.db, id)
	if err != nil {
		return a.redirectWithError(c, a.echo.Reverse("workouts"), err)
	}

	var params database.RoutSegmentCreationParams

	if err := c.Bind(&params); err != nil {
		return a.redirectWithError(c, a.echo.Reverse("workouts"), err)
	}

	content, err := database.RouteSegmentFromPoints(workout, &params)
	if err != nil {
		return a.redirectWithError(c, a.echo.Reverse("workouts"), err)
	}

	rs, err := database.AddRouteSegment(a.db, "", params.Filename(), content)
	if err != nil {
		return a.redirectWithError(c, a.echo.Reverse("workouts"), err)
	}

	a.addNoticeT(c, "translation.The_route_segment_s_has_been_created_we_search_for_matches_in_the_background", rs.Name)

	return c.Redirect(http.StatusFound, a.echo.Reverse("route-segment-show", rs.ID))
}

func (a *App) workoutsCreateRouteSegmentHandler(c echo.Context) error {
	id, err := cast.ToUint64E(c.Param("id"))
	if err != nil {
		return a.redirectWithError(c, a.echo.Reverse("workouts"), err)
	}

	w, err := database.GetWorkoutDetails(a.db, id)
	if err != nil {
		return a.redirectWithError(c, a.echo.Reverse("workouts"), err)
	}

	return Render(c, http.StatusOK, workouts.CreateRouteSegment(w))
}

func (a *App) workoutsDeleteConfirmHandler(c echo.Context) error {
	w, err := a.getWorkout(c)
	if err != nil {
		return a.redirectWithError(c, a.echo.Reverse("workouts"), err)
	}

	return Render(c, http.StatusOK, workouts.DeleteModal(w))
}
