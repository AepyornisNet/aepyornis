package app

import (
	"github.com/AepyornisNet/aepyornis/pkg/controller"
	"github.com/labstack/echo/v5"
)

func (a *App) registerActivityPubController(e *echo.Group) {
	wfc := controller.NewWellKnownController(a.injector)
	wellKnownGroup := e.Group("/.well-known")
	wellKnownGroup.GET("/webfinger", wfc.WebFinger)
	wellKnownGroup.GET("/host-meta", wfc.HostMeta)

	auc := controller.NewApUserController(a.injector)
	aic := controller.NewApInboxController(a.injector)
	aoc := controller.NewApOutboxController(a.injector)
	apGroup := e.Group("/ap")
	apGroup.Use(a.RequestingActorMiddleware)
	apGroup.GET("/users/:username", auc.GetUser)
	apGroup.POST("/users/:username/inbox", aic.Inbox)
	apGroup.GET("/users/:username/outbox", aoc.Outbox)
	apGroup.GET("/users/:username/outbox/:id", aoc.OutboxItem)
	apGroup.GET("/users/:username/outbox/:id/fit", aoc.OutboxFit)
	apGroup.GET("/users/:username/outbox/:id/route-image", aoc.OutboxRouteImage)
	apGroup.GET("/users/:username/outbox/:id/replies", aoc.OutboxReplies)
	apGroup.GET("/users/:username/following", auc.Following)
	apGroup.GET("/users/:username/followers", auc.Followers)
}

func (a *App) registerUserController(apiGroup *echo.Group) {
	uc := controller.NewUserController(a.injector)

	apiGroup.GET("/whoami", uc.GetWhoami)
	apiGroup.GET("/user-profile", uc.GetUserProfileByHandle)
	apiGroup.GET("/user-profile/search", uc.SearchProfiles)
	apiGroup.POST("/user-profile/follow", uc.FollowUserByHandle)
	apiGroup.POST("/user-profile/unfollow", uc.UnfollowUserByHandle)
	apiGroup.GET("/totals", uc.GetTotals)
	apiGroup.GET("/records", uc.GetRecords)
	apiGroup.GET("/records/climbs/ranking", uc.GetClimbRecordsRanking)
	apiGroup.GET("/records/ranking", uc.GetRecordsRanking)
	apiGroup.GET("/:id", uc.GetUserByID)
}

func (a *App) registerNotificationController(apiGroup *echo.Group) {
	hc := controller.NewNotificationController(a.injector)

	apiGroup.GET("/notifications", hc.GetNotifications)
	apiGroup.POST("/notifications/read", hc.MarkAsRead)
	apiGroup.GET("/notifications/settings", hc.GetConfig)
	apiGroup.GET("/notifications/webpush/subscriptions", hc.GetWebpushSubscriptions)
	apiGroup.POST("/notifications/webpush/subscribe", hc.SubscribeWebpush)
	apiGroup.POST("/notifications/webpush/unsubscribe", hc.UnsubscribeWebpush)
	apiGroup.POST("/notifications/:type", hc.UpdateConfig)
}

func (a *App) registerAuthController(apiGroupPublic *echo.Group) {
	ac := controller.NewAuthController(a.injector)

	authGroup := apiGroupPublic.Group("/auth")
	authGroup.POST("/signin", ac.SignIn)
	authGroup.POST("/register", ac.Register)
	authGroup.POST("/signout", ac.SignOut)
}

func (a *App) registerHammerheadPublicController(apiGroupPublic *echo.Group) {
	hc := controller.NewHammerheadController(a.injector)

	apiGroupPublic.POST("/webhooks/hammerhead/activity", hc.Webhook)
}

func (a *App) registerStatisticsController(apiGroup *echo.Group) {
	sc := controller.NewStatisticsController(a.injector)

	apiGroup.GET("/statistics", sc.GetStatistics)
}

func (a *App) registerProfileController(apiGroup *echo.Group) {
	pc := controller.NewProfileController(a.injector)
	hc := controller.NewHammerheadController(a.injector)

	profileGroup := apiGroup.Group("/profile")
	profileGroup.GET("", pc.GetProfile)
	profileGroup.PUT("", pc.UpdateProfile)
	profileGroup.POST("/change-password", pc.ChangePassword)
	profileGroup.POST("/reset-api-key", pc.ResetAPIKey)
	profileGroup.POST("/enable-activity-pub", pc.EnableActivityPub)
	profileGroup.GET("/follow-requests", pc.ListFollowRequests)
	profileGroup.POST("/follow-requests/:id/accept", pc.AcceptFollowRequest)
	profileGroup.POST("/refresh-workouts", pc.RefreshWorkouts)
	profileGroup.POST("/update-version", pc.UpdateVersion)
	profileGroup.GET("/apps/hammerhead", hc.GetConnection)
	profileGroup.POST("/apps/hammerhead/connect", hc.Connect)
	profileGroup.GET("/apps/hammerhead/callback", hc.Callback)
	profileGroup.DELETE("/apps/hammerhead", hc.Disconnect)
}

func (a *App) registerAdminController(apiGroup *echo.Group) {
	ac := controller.NewAdminController(
		a.injector,
		a.ResetConfiguration,
	)

	adminGroup := apiGroup.Group("/admin")
	adminGroup.Use(a.ValidateAdminMiddleware)

	adminGroup.GET("/users", ac.GetUsers)
	adminGroup.GET("/users/:id", ac.GetUser)
	adminGroup.PUT("/users/:id", ac.UpdateUser)
	adminGroup.DELETE("/users/:id", ac.DeleteUser)
	adminGroup.PUT("/config", ac.UpdateConfig)
}

func (a *App) registerEquipmentController(apiGroup *echo.Group) {
	ec := controller.NewEquipmentController(a.injector)

	apiGroup.GET("/equipment", ec.GetEquipmentList)
	apiGroup.GET("/equipment/:id", ec.GetEquipment)
	apiGroup.POST("/equipment", ec.CreateEquipment)
	apiGroup.PUT("/equipment/:id", ec.UpdateEquipment)
	apiGroup.DELETE("/equipment/:id", ec.DeleteEquipment)
}

func (a *App) registerWorkoutController(apiGroup *echo.Group) {
	wc := controller.NewWorkoutController(a.injector)

	workoutGroup := apiGroup.Group("/workouts")
	workoutGroup.GET("", wc.GetWorkouts)
	workoutGroup.GET("/filter-options", wc.GetWorkoutFilterOptions)
	workoutGroup.POST("", wc.CreateWorkout)
	workoutGroup.GET("/recent", wc.GetRecentWorkouts)
	workoutGroup.GET("/calendar", wc.GetWorkoutCalendar)
	workoutGroup.GET("/:id", wc.GetWorkout)
	workoutGroup.GET("/:id/likes", wc.GetWorkoutLikes)
	workoutGroup.GET("/:id/breakdown", wc.GetWorkoutBreakdown)
	workoutGroup.GET("/:id/stats-range", wc.GetWorkoutRangeStats)
	workoutGroup.GET("/:id/replies", wc.GetWorkoutReplies)
	workoutGroup.POST("/:id/like", wc.LikeWorkout)
	workoutGroup.POST("/like", wc.LikeWorkoutByObject)
	workoutGroup.POST("/:id/replies", wc.CreateReply)
	workoutGroup.GET("/:id/download", wc.DownloadWorkout)
	workoutGroup.GET("/download-zip", wc.DownloadWorkoutsZip)
	workoutGroup.POST("/add-equipment", wc.AddEquipmentToWorkouts)
	workoutGroup.GET("/:id/attachments/:attachment_id", wc.DownloadWorkoutAttachment)
	workoutGroup.PUT("/:id", wc.UpdateWorkout)
	workoutGroup.POST("/:id/toggle-lock", wc.ToggleWorkoutLock)
	workoutGroup.POST("/:id/refresh", wc.RefreshWorkout)
	workoutGroup.DELETE("/:id", wc.DeleteWorkout)
}

func (a *App) registerHeatmapController(apiGroup *echo.Group) {
	hc := controller.NewHeatmapController(a.injector)

	apiGroup.GET("/workouts/coordinates", hc.GetWorkoutCoordinates)
	apiGroup.GET("/workouts/centers", hc.GetWorkoutCenters)
}

func (a *App) registerMeasurementController(apiGroup *echo.Group) {
	mc := controller.NewMeasurementController(a.injector)

	apiGroup.GET("/measurements", mc.GetMeasurements)
	apiGroup.POST("/measurements", mc.CreateMeasurement)
	apiGroup.DELETE("/measurements/:date", mc.DeleteMeasurement)
}

func (a *App) registerRouteSegmentController(apiGroup *echo.Group) {
	rsc := controller.NewRouteSegmentController(a.injector)

	routeSegmentsGroup := apiGroup.Group("/route-segments")
	routeSegmentsGroup.GET("", rsc.GetRouteSegments)
	routeSegmentsGroup.POST("", rsc.CreateRouteSegment)
	routeSegmentsGroup.GET("/:id", rsc.GetRouteSegment)
	routeSegmentsGroup.PUT("/:id", rsc.UpdateRouteSegment)
	routeSegmentsGroup.DELETE("/:id", rsc.DeleteRouteSegment)
	routeSegmentsGroup.POST("/:id/refresh", rsc.RefreshRouteSegment)
	routeSegmentsGroup.POST("/:id/matches", rsc.FindRouteSegmentMatches)
	routeSegmentsGroup.GET("/:id/download", rsc.DownloadRouteSegment)
	apiGroup.POST("/workouts/:id/route-segment", rsc.CreateRouteSegmentFromWorkout)
}
