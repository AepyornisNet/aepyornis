package app

import (
	"github.com/AepyornisNet/aepyornis/pkg/controller"
	"github.com/labstack/echo/v4"
)

func (a *App) registerActivityPubController(e *echo.Group) {
	wfc := controller.NewWellKnownController(a.injector)
	wellKnownGroup := e.Group("/.well-known")
	wellKnownGroup.GET("/webfinger", wfc.WebFinger).Name = "webfinger"
	wellKnownGroup.GET("/host-meta", wfc.HostMeta).Name = "host-meta"

	auc := controller.NewApUserController(a.injector)
	aic := controller.NewApInboxController(a.injector)
	aoc := controller.NewApOutboxController(a.injector)
	apGroup := e.Group("/ap")
	apGroup.Use(a.RequestingActorMiddleware)
	apGroup.GET("/users/:username", auc.GetUser).Name = "ap-user"
	apGroup.POST("/users/:username/inbox", aic.Inbox).Name = "ap-user-inbox"
	apGroup.GET("/users/:username/outbox", aoc.Outbox).Name = "ap-user-outbox"
	apGroup.GET("/users/:username/outbox/:id", aoc.OutboxItem).Name = "ap-user-outbox-item"
	apGroup.GET("/users/:username/outbox/:id/fit", aoc.OutboxFit).Name = "ap-user-outbox-fit"
	apGroup.GET("/users/:username/outbox/:id/route-image", aoc.OutboxRouteImage).Name = "ap-user-outbox-route-image"
	apGroup.GET("/users/:username/outbox/:id/replies", aoc.OutboxReplies).Name = "ap-user-outbox-replies"
	apGroup.GET("/users/:username/following", auc.Following).Name = "ap-user-following"
	apGroup.GET("/users/:username/followers", auc.Followers).Name = "ap-user-followers"
}

func (a *App) registerUserController(apiGroup *echo.Group) {
	uc := controller.NewUserController(a.injector)

	apiGroup.GET("/whoami", uc.GetWhoami).Name = "whoami"
	apiGroup.GET("/user-profile", uc.GetUserProfileByHandle).Name = "user-profile"
	apiGroup.POST("/user-profile/follow", uc.FollowUserByHandle).Name = "user-profile-follow"
	apiGroup.POST("/user-profile/unfollow", uc.UnfollowUserByHandle).Name = "user-profile-unfollow"
	apiGroup.GET("/totals", uc.GetTotals).Name = "totals"
	apiGroup.GET("/records", uc.GetRecords).Name = "records"
	apiGroup.GET("/records/climbs/ranking", uc.GetClimbRecordsRanking).Name = "records-climbs-ranking"
	apiGroup.GET("/records/ranking", uc.GetRecordsRanking).Name = "records-ranking"
	apiGroup.GET("/:id", uc.GetUserByID).Name = "user-show"
}

func (a *App) registerAuthController(apiGroupPublic *echo.Group) {
	ac := controller.NewAuthController(a.injector)

	authGroup := apiGroupPublic.Group("/auth")
	authGroup.POST("/signin", ac.SignIn).Name = "auth-signin"
	authGroup.POST("/register", ac.Register).Name = "auth-register"
	authGroup.POST("/signout", ac.SignOut).Name = "auth-signout"
}

func (a *App) registerHammerheadPublicController(apiGroupPublic *echo.Group) {
	hc := controller.NewHammerheadController(a.injector)

	apiGroupPublic.POST("/webhooks/hammerhead/activity", hc.Webhook).Name = "hammerhead-webhook-activity"
}

func (a *App) registerStatisticsController(apiGroup *echo.Group) {
	sc := controller.NewStatisticsController(a.injector)

	apiGroup.GET("/statistics", sc.GetStatistics).Name = "statistics"
}

func (a *App) registerProfileController(apiGroup *echo.Group) {
	pc := controller.NewProfileController(a.injector)
	hc := controller.NewHammerheadController(a.injector)

	profileGroup := apiGroup.Group("/profile")
	profileGroup.GET("", pc.GetProfile).Name = "profile"
	profileGroup.PUT("", pc.UpdateProfile).Name = "profile-update"
	profileGroup.POST("/change-password", pc.ChangePassword).Name = "profile-change-password"
	profileGroup.POST("/reset-api-key", pc.ResetAPIKey).Name = "profile-reset-api-key"
	profileGroup.POST("/enable-activity-pub", pc.EnableActivityPub).Name = "profile-enable-activity-pub"
	profileGroup.GET("/follow-requests", pc.ListFollowRequests).Name = "profile-follow-requests"
	profileGroup.POST("/follow-requests/:id/accept", pc.AcceptFollowRequest).Name = "profile-follow-request-accept"
	profileGroup.POST("/refresh-workouts", pc.RefreshWorkouts).Name = "profile-refresh-workouts"
	profileGroup.POST("/update-version", pc.UpdateVersion).Name = "user-update-version"
	profileGroup.GET("/apps/hammerhead", hc.GetConnection).Name = "profile-apps-hammerhead-status"
	profileGroup.POST("/apps/hammerhead/connect", hc.Connect).Name = "profile-apps-hammerhead-connect"
	profileGroup.GET("/apps/hammerhead/callback", hc.Callback).Name = "profile-apps-hammerhead-callback"
	profileGroup.DELETE("/apps/hammerhead", hc.Disconnect).Name = "profile-apps-hammerhead-disconnect"
}

func (a *App) registerAdminController(apiGroup *echo.Group) {
	ac := controller.NewAdminController(
		a.injector,
		a.ResetConfiguration,
	)

	adminGroup := apiGroup.Group("/admin")
	adminGroup.Use(a.ValidateAdminMiddleware)

	adminGroup.GET("/users", ac.GetUsers).Name = "admin-users"
	adminGroup.GET("/users/:id", ac.GetUser).Name = "admin-user"
	adminGroup.PUT("/users/:id", ac.UpdateUser).Name = "admin-user-update"
	adminGroup.DELETE("/users/:id", ac.DeleteUser).Name = "admin-user-delete"
	adminGroup.PUT("/config", ac.UpdateConfig).Name = "admin-config-update"
}

func (a *App) registerEquipmentController(apiGroup *echo.Group) {
	ec := controller.NewEquipmentController(a.injector)

	apiGroup.GET("/equipment", ec.GetEquipmentList).Name = "equipment-list"
	apiGroup.GET("/equipment/:id", ec.GetEquipment).Name = "equipment-get"
	apiGroup.POST("/equipment", ec.CreateEquipment).Name = "equipment-create"
	apiGroup.PUT("/equipment/:id", ec.UpdateEquipment).Name = "equipment-update"
	apiGroup.DELETE("/equipment/:id", ec.DeleteEquipment).Name = "equipment-delete"
}

func (a *App) registerWorkoutController(apiGroup *echo.Group) {
	wc := controller.NewWorkoutController(a.injector)

	workoutGroup := apiGroup.Group("/workouts")
	workoutGroup.GET("", wc.GetWorkouts).Name = "workouts-list"
	workoutGroup.POST("", wc.CreateWorkout).Name = "workouts-create"
	workoutGroup.GET("/recent", wc.GetRecentWorkouts).Name = "workouts-recent"
	workoutGroup.GET("/calendar", wc.GetWorkoutCalendar).Name = "workouts-calendar"
	workoutGroup.GET("/:id", wc.GetWorkout).Name = "workout-get"
	workoutGroup.GET("/:id/likes", wc.GetWorkoutLikes).Name = "workout-likes"
	workoutGroup.GET("/:id/breakdown", wc.GetWorkoutBreakdown).Name = "workout-breakdown"
	workoutGroup.GET("/:id/stats-range", wc.GetWorkoutRangeStats).Name = "workout-range-stats"
	workoutGroup.GET("/:id/replies", wc.GetWorkoutReplies).Name = "workout-replies"
	workoutGroup.POST("/:id/like", wc.LikeWorkout).Name = "workout-like"
	workoutGroup.POST("/like", wc.LikeWorkoutByObject).Name = "workout-like-object"
	workoutGroup.POST("/:id/replies", wc.CreateReply).Name = "workout-create-reply"
	workoutGroup.GET("/:id/download", wc.DownloadWorkout).Name = "workout-download"
	workoutGroup.GET("/:id/attachments/:attachment_id", wc.DownloadWorkoutAttachment).Name = "workout-attachment-download"
	workoutGroup.PUT("/:id", wc.UpdateWorkout).Name = "workout-update"
	workoutGroup.POST("/:id/toggle-lock", wc.ToggleWorkoutLock).Name = "workout-toggle-lock"
	workoutGroup.POST("/:id/refresh", wc.RefreshWorkout).Name = "workout-refresh"
	workoutGroup.DELETE("/:id", wc.DeleteWorkout).Name = "workout-delete"
}

func (a *App) registerHeatmapController(apiGroup *echo.Group) {
	hc := controller.NewHeatmapController(a.injector)

	apiGroup.GET("/workouts/coordinates", hc.GetWorkoutCoordinates).Name = "workouts-coordinates"
	apiGroup.GET("/workouts/centers", hc.GetWorkoutCenters).Name = "workouts-centers"
}

func (a *App) registerMeasurementController(apiGroup *echo.Group) {
	mc := controller.NewMeasurementController(a.injector)

	apiGroup.GET("/measurements", mc.GetMeasurements).Name = "measurements-list"
	apiGroup.POST("/measurements", mc.CreateMeasurement).Name = "measurements-create"
	apiGroup.DELETE("/measurements/:date", mc.DeleteMeasurement).Name = "measurements-delete"
}

func (a *App) registerRouteSegmentController(apiGroup *echo.Group) {
	rsc := controller.NewRouteSegmentController(a.injector)

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
