package repository

import "gorm.io/gorm"

type Repositories struct {
	APOutbox         APOutbox
	APOutboxDelivery APOutboxDelivery
	Follower         Follower
}

func New(db *gorm.DB) *Repositories {
	return &Repositories{
		APOutbox:         NewAPOutbox(db),
		APOutboxDelivery: NewAPOutboxDelivery(db),
		Follower:         NewFollower(db),
	}
}
