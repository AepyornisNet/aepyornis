package app

import (
	"github.com/labstack/echo/v4"
)

func (a *App) addRoutesWorkouts(e *echo.Group) {
	workoutsGroup := e.Group("/workouts")
	workoutsGroup.POST("", a.addWorkout).Name = "workouts-create"
	workoutsGroup.POST("/:id", a.workoutsUpdateHandler).Name = "workout-update"
}
