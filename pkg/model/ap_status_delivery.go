package model

import (
	"time"

	"gorm.io/gorm"
)

type APStatusDelivery struct {
	Model

	APStatusID uint64    `gorm:"column:ap_status_id;uniqueIndex:idx_ap_outbox_delivery_entry_profile;not null" json:"ap_status_id"`
	APStatus   *APStatus `gorm:"foreignKey:APStatusID;references:ID;constraint:OnDelete:CASCADE"`

	ProfileID   *uint64   `gorm:"uniqueIndex:idx_ap_outbox_delivery_entry_profile" json:"profile_id,omitempty"`
	Profile     *Profile  `gorm:"foreignKey:ProfileID;constraint:OnDelete:CASCADE" json:"-"`
	DeliveredAt time.Time `gorm:"index;not null" json:"delivered_at"`
}

type APPendingStatusDelivery struct {
	EntryID    uint64 `json:"entry_id"`
	UserID     uint64 `json:"user_id"`
	Activity   []byte `json:"activity"`
	ActorIRI   string `json:"actor_iri"`
	ActorInbox string `json:"actor_inbox"`
}

func (APStatusDelivery) TableName() string {
	return "ap_outbox_delivery"
}

func (d *APStatusDelivery) BeforeCreate(_ *gorm.DB) error {
	if d.DeliveredAt.IsZero() {
		d.DeliveredAt = time.Now().UTC()
	}

	return nil
}
