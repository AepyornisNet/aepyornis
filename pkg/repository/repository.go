package repository

import "gorm.io/gorm"

type Repositories struct {
	APStatus         APStatus
	APOutbox         APOutbox
	APStatusDelivery APStatusDelivery
	Equipment        Equipment
	Follower         Follower
	Measurement      Measurement
	RouteSegment     RouteSegment
	User             User
	Workout          Workout
	WorkoutLike      WorkoutLike
	WorkoutReply     WorkoutReply
}

func New(db *gorm.DB) *Repositories {
	return &Repositories{
		APStatus:         NewAPStatus(db),
		APOutbox:         NewAPOutbox(db),
		APStatusDelivery: NewAPStatusDelivery(db),
		Equipment:        NewEquipment(db),
		Follower:         NewFollower(db),
		Measurement:      NewMeasurement(db),
		RouteSegment:     NewRouteSegment(db),
		User:             NewUser(db),
		Workout:          NewWorkout(db),
		WorkoutLike:      NewWorkoutLike(db),
		WorkoutReply:     NewWorkoutReply(db),
	}
}
