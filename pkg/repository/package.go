package repository

import (
	"github.com/samber/do/v2"
)

var Package = do.Package(
	do.Lazy(NewAPStatus),
	do.Lazy(NewAPOutbox),
	do.Lazy(NewAPStatusDelivery),
	do.Lazy(NewEquipment),
	do.Lazy(NewFollower),
	do.Lazy(NewMeasurement),
	do.Lazy(NewRouteSegment),
	do.Lazy(NewUser),
	do.Lazy(NewWorkout),
	do.Lazy(NewWorkoutLike),
	do.Lazy(NewWorkoutReply),
)
