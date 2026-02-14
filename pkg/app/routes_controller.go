package app

import (
	"github.com/jovandeginste/workout-tracker/v2/pkg/controller"
	"github.com/labstack/echo/v4"
)

func (a *App) registerUserController(apiGroup *echo.Group) {
	uc := controller.NewUserController(a.getContainer())

	apiGroup.GET("/whoami", uc.GetWhoami).Name = "api-v2-whoami"
	apiGroup.GET("/totals", uc.GetTotals).Name = "api-v2-totals"
	apiGroup.GET("/records", uc.GetRecords).Name = "api-v2-records"
	apiGroup.GET("/records/climbs/ranking", uc.GetClimbRecordsRanking).Name = "api-v2-records-climbs-ranking"
	apiGroup.GET("/records/ranking", uc.GetRecordsRanking).Name = "api-v2-records-ranking"
	apiGroup.GET("/:id", uc.GetUserByID).Name = "api-v2-user-show"
}

func (a *App) registerStatisticsController(apiGroup *echo.Group) {
	sc := controller.NewStatisticsController(a.getContainer())

	apiGroup.GET("/statistics", sc.GetStatistics).Name = "api-v2-statistics"
}

func (a *App) registerProfileController(apiGroup *echo.Group) {
	pc := controller.NewProfileController(a.getContainer())

	profileGroup := apiGroup.Group("/profile")
	profileGroup.GET("", pc.GetProfile).Name = "api-v2-profile"
	profileGroup.PUT("", pc.UpdateProfile).Name = "api-v2-profile-update"
	profileGroup.POST("/reset-api-key", pc.ResetAPIKey).Name = "api-v2-profile-reset-api-key"
	profileGroup.POST("/refresh-workouts", pc.RefreshWorkouts).Name = "api-v2-profile-refresh-workouts"
	profileGroup.POST("/update-version", pc.UpdateVersion).Name = "api-v2-user-update-version"
}

func (a *App) registerAdminController(apiGroup *echo.Group) {
	ac := controller.NewAdminController(
		a.getContainer(),
		a.ResetConfiguration,
	)

	adminGroup := apiGroup.Group("/admin")
	adminGroup.Use(a.ValidateAdminMiddleware)

	adminGroup.GET("/users", ac.GetUsers).Name = "api-v2-admin-users"
	adminGroup.GET("/users/:id", ac.GetUser).Name = "api-v2-admin-user"
	adminGroup.PUT("/users/:id", ac.UpdateUser).Name = "api-v2-admin-user-update"
	adminGroup.DELETE("/users/:id", ac.DeleteUser).Name = "api-v2-admin-user-delete"
	adminGroup.PUT("/config", ac.UpdateConfig).Name = "api-v2-admin-config-update"
}

func (a *App) registerEquipmentController(apiGroup *echo.Group) {
	ec := controller.NewEquipmentController(a.getContainer())

	apiGroup.GET("/equipment", ec.GetEquipmentList).Name = "equipment-list"
	apiGroup.GET("/equipment/:id", ec.GetEquipment).Name = "equipment-get"
	apiGroup.POST("/equipment", ec.CreateEquipment).Name = "equipment-create"
	apiGroup.PUT("/equipment/:id", ec.UpdateEquipment).Name = "equipment-update"
	apiGroup.DELETE("/equipment/:id", ec.DeleteEquipment).Name = "equipment-delete"
}

func (a *App) registerWorkoutController(apiGroup *echo.Group, apiGroupPublic *echo.Group) {
	wc := controller.NewWorkoutController(a.getContainer())

	workoutGroup := apiGroup.Group("/workouts")
	workoutGroup.GET("", wc.GetWorkouts).Name = "workouts-list"
	workoutGroup.POST("", wc.CreateWorkout).Name = "workouts-create"
	workoutGroup.GET("/recent", wc.GetRecentWorkouts).Name = "workouts-recent"
	workoutGroup.GET("/calendar", wc.GetWorkoutCalendar).Name = "workouts-calendar"
	workoutGroup.GET("/:id", wc.GetWorkout).Name = "workout-get"
	workoutGroup.GET("/:id/breakdown", wc.GetWorkoutBreakdown).Name = "workout-breakdown"
	workoutGroup.GET("/:id/stats-range", wc.GetWorkoutRangeStats).Name = "workout-range-stats"
	workoutGroup.GET("/:id/download", wc.DownloadWorkout).Name = "workout-download"
	workoutGroup.PUT("/:id", wc.UpdateWorkout).Name = "workout-update"
	workoutGroup.POST("/:id/toggle-lock", wc.ToggleWorkoutLock).Name = "workout-toggle-lock"
	workoutGroup.POST("/:id/refresh", wc.RefreshWorkout).Name = "workout-refresh"
	workoutGroup.POST("/:id/share", wc.CreateWorkoutShare).Name = "workout-share"
	workoutGroup.DELETE("/:id", wc.DeleteWorkout).Name = "workout-delete"
	workoutGroup.DELETE("/:id/share", wc.DeleteWorkoutShare).Name = "workout-share-delete"

	apiGroupPublic.GET("/workouts/public/:uuid", wc.GetPublicWorkout).Name = "workout-public"
	apiGroupPublic.GET("/workouts/public/:uuid/breakdown", wc.GetPublicWorkoutBreakdown).Name = "workout-public-breakdown"
	apiGroupPublic.GET("/workouts/public/:uuid/stats-range", wc.GetPublicWorkoutRangeStats).Name = "workout-public-range-stats"
}

func (a *App) registerHeatmapController(apiGroup *echo.Group) {
	hc := controller.NewHeatmapController(a.getContainer())

	apiGroup.GET("/workouts/coordinates", hc.GetWorkoutCoordinates).Name = "workouts-coordinates"
	apiGroup.GET("/workouts/centers", hc.GetWorkoutCenters).Name = "workouts-centers"
}

func (a *App) registerMeasurementController(apiGroup *echo.Group) {
	mc := controller.NewMeasurementController(a.getContainer())

	apiGroup.GET("/measurements", mc.GetMeasurements).Name = "measurements-list"
	apiGroup.POST("/measurements", mc.CreateMeasurement).Name = "measurements-create"
	apiGroup.DELETE("/measurements/:date", mc.DeleteMeasurement).Name = "measurements-delete"
}

func (a *App) registerRouteSegmentController(apiGroup *echo.Group) {
	rsc := controller.NewRouteSegmentController(a.getContainer())

	routeSegmentsGroup := apiGroup.Group("/route-segments")
	routeSegmentsGroup.GET("", rsc.GetRouteSegments).Name = "route-segments-list"
	routeSegmentsGroup.POST("", rsc.CreateRouteSegment).Name = "route-segment-create"
	routeSegmentsGroup.GET("/:id", rsc.GetRouteSegment).Name = "route-segment-get"
	routeSegmentsGroup.PUT("/:id", rsc.UpdateRouteSegment).Name = "route-segment-update"
	routeSegmentsGroup.DELETE("/:id", rsc.DeleteRouteSegment).Name = "route-segment-delete"
	routeSegmentsGroup.POST("/:id/refresh", rsc.RefreshRouteSegment).Name = "route-segment-refresh"
	routeSegmentsGroup.POST("/:id/matches", rsc.FindRouteSegmentMatches).Name = "route-segment-matches"
	routeSegmentsGroup.GET("/:id/download", rsc.DownloadRouteSegment).Name = "route-segment-download"
	apiGroup.POST("/workouts/:id/route-segment", rsc.CreateRouteSegmentFromWorkout).Name = "workout-route-segment-create"
}
