package repository

import (
	"errors"

	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type APOutboxDelivery interface {
	RecordDelivery(outboxEntryID uint64, actorIRI string) error
	ListPendingDeliveriesForEntry(entryID uint64) ([]model.APPendingOutboxDelivery, error)
}

type apOutboxDeliveryRepository struct {
	db *gorm.DB
}

func NewAPOutboxDelivery(db *gorm.DB) APOutboxDelivery {
	return &apOutboxDeliveryRepository{db: db}
}

func (r *apOutboxDeliveryRepository) RecordDelivery(outboxEntryID uint64, actorIRI string) error {
	if outboxEntryID == 0 {
		return errors.New("outbox entry id is required")
	}
	if actorIRI == "" {
		return errors.New("actor IRI is required")
	}

	d := &model.APOutboxDelivery{
		APOutboxEntryID: outboxEntryID,
		ActorIRI:        actorIRI,
	}

	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(d).Error
}

func (r *apOutboxDeliveryRepository) ListPendingDeliveriesForEntry(entryID uint64) ([]model.APPendingOutboxDelivery, error) {
	rows := make([]model.APPendingOutboxDelivery, 0)
	err := r.db.Table("ap_outbox").
		Select("ap_outbox.id AS entry_id, ap_outbox.user_id AS user_id, ap_outbox.activity AS activity, followers.actor_iri AS actor_iri, followers.actor_inbox AS actor_inbox").
		Joins("JOIN followers ON followers.user_id = ap_outbox.user_id AND followers.approved = ?", true).
		Joins("LEFT JOIN ap_outbox_delivery ON ap_outbox_delivery.ap_outbox_entry_id = ap_outbox.id AND ap_outbox_delivery.actor_iri = followers.actor_iri").
		Where("ap_outbox.id = ?", entryID).
		Where("ap_outbox_delivery.id IS NULL").
		Where("followers.actor_iri <> ''").
		Where("followers.actor_inbox <> ''").
		Find(&rows).
		Error

	return rows, err
}
