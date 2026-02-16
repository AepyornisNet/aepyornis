package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type APOutboxDelivery struct {
	Model

	APOutboxEntryID uint64 `gorm:"uniqueIndex:idx_ap_outbox_delivery_entry_actor;not null" json:"ap_outbox_entry_id"`
	APOutboxEntry   *APOutboxEntry

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

func RecordAPOutboxDelivery(db *gorm.DB, outboxEntryID uint64, actorIRI string) error {
	if outboxEntryID == 0 {
		return errors.New("outbox entry id is required")
	}
	if actorIRI == "" {
		return errors.New("actor IRI is required")
	}

	d := &APOutboxDelivery{
		APOutboxEntryID: outboxEntryID,
		ActorIRI:        actorIRI,
	}

	return db.Clauses(clause.OnConflict{DoNothing: true}).Create(d).Error
}

func ListPendingAPOutboxDeliveries(db *gorm.DB, limit int) ([]APPendingOutboxDelivery, error) {
	if limit <= 0 {
		limit = 25
	}

	rows := make([]APPendingOutboxDelivery, 0)
	err := db.Table("ap_outbox").
		Select("ap_outbox.id AS entry_id, ap_outbox.user_id AS user_id, ap_outbox.activity AS activity, followers.actor_iri AS actor_iri, followers.actor_inbox AS actor_inbox").
		Joins("JOIN followers ON followers.user_id = ap_outbox.user_id AND followers.approved = ?", true).
		Joins("LEFT JOIN ap_outbox_delivery ON ap_outbox_delivery.ap_outbox_entry_id = ap_outbox.id AND ap_outbox_delivery.actor_iri = followers.actor_iri").
		Where("ap_outbox_delivery.id IS NULL").
		Where("followers.actor_iri <> ''").
		Where("followers.actor_inbox <> ''").
		Order("ap_outbox.published_at ASC").
		Order("ap_outbox.id ASC").
		Limit(limit).
		Find(&rows).
		Error

	return rows, err
}
