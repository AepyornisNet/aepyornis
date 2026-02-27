package model

import (
	"time"

	"gorm.io/gorm"
)

type APOutboxDelivery struct {
	Model

	APOutboxEntryID uint64         `gorm:"uniqueIndex:idx_ap_outbox_delivery_entry_actor;not null" json:"ap_outbox_entry_id"`
	APOutboxEntry   *APOutboxEntry `gorm:"constraint:OnDelete:CASCADE"`

	ActorIRI    string    `gorm:"type:text;uniqueIndex:idx_ap_outbox_delivery_entry_actor;not null" json:"actor_iri"`
	DeliveredAt time.Time `gorm:"index;not null" json:"delivered_at"`
}

type APPendingOutboxDelivery struct {
	EntryID    uint64 `json:"entry_id"`
	UserID     uint64 `json:"user_id"`
	Activity   []byte `json:"activity"`
	ActorIRI   string `json:"actor_iri"`
	ActorInbox string `json:"actor_inbox"`
}

func (APOutboxDelivery) TableName() string {
	return "ap_outbox_delivery"
}

func (d *APOutboxDelivery) BeforeCreate(_ *gorm.DB) error {
	if d.DeliveredAt.IsZero() {
		d.DeliveredAt = time.Now().UTC()
	}

	return nil
}
