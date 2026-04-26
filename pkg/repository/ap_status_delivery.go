package repository

import (
	"errors"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/samber/do/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type APStatusDelivery interface {
	RecordDelivery(outboxEntryID uint64, actorIRI string) error
	ListPendingDeliveriesForEntry(entryID uint64) ([]model.APPendingStatusDelivery, error)
}

type apOutboxDeliveryRepository struct {
	db *gorm.DB
}

func NewAPStatusDelivery(injector do.Injector) (APStatusDelivery, error) {
	return &apOutboxDeliveryRepository{db: do.MustInvoke[*gorm.DB](injector)}, nil
}

func (r *apOutboxDeliveryRepository) RecordDelivery(outboxEntryID uint64, actorIRI string) error {
	if outboxEntryID == 0 {
		return errors.New("outbox entry id is required")
	}
	if actorIRI == "" {
		return errors.New("actor IRI is required")
	}

	d := &model.APStatusDelivery{
		APStatusID: outboxEntryID,
		ActorIRI:   actorIRI,
	}

	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(d).Error
}

func (r *apOutboxDeliveryRepository) ListPendingDeliveriesForEntry(entryID uint64) ([]model.APPendingStatusDelivery, error) {
	rows := make([]model.APPendingStatusDelivery, 0)
	err := r.db.Table("ap_statuses").
		Select("ap_statuses.id AS entry_id, ap_statuses.user_id AS user_id, ap_statuses.activity AS activity, followers.actor_iri AS actor_iri, followers.actor_inbox AS actor_inbox").
		Joins("JOIN followers ON followers.user_id = ap_statuses.user_id AND followers.approved = ?", true).
		Joins("LEFT JOIN ap_outbox_delivery ON ap_outbox_delivery.ap_status_id = ap_statuses.id AND ap_outbox_delivery.actor_iri = followers.actor_iri").
		Where("ap_statuses.id = ?", entryID).
		Where("ap_outbox_delivery.id IS NULL").
		Where("followers.actor_iri <> ''").
		Where("followers.actor_inbox <> ''").
		Find(&rows).
		Error

	return rows, err
}
