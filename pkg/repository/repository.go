package repository

import "gorm.io/gorm"

type Repositories struct {
	APOutbox         APOutbox
	APOutboxDelivery APOutboxDelivery
	Equipment        Equipment
	Follower         Follower
	Measurement      Measurement
	RouteSegment     RouteSegment
}

func New(db *gorm.DB) *Repositories {
	return &Repositories{
		APOutbox:         NewAPOutbox(db),
		APOutboxDelivery: NewAPOutboxDelivery(db),
		Equipment:        NewEquipment(db),
		Follower:         NewFollower(db),
		Measurement:      NewMeasurement(db),
		RouteSegment:     NewRouteSegment(db),
	}
}
