package repository

import (
	"errors"
	"strings"

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

	profileURL := strings.TrimSpace(actorIRI)
	profile, err := (&model.Profile{URL: &profileURL}).UpsertRemote(r.db)
	if err != nil {
		return err
	}

	d := &model.APStatusDelivery{
		APStatusID: outboxEntryID,
		ProfileID:  &profile.ID,
	}

	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(d).Error
}

func (r *apOutboxDeliveryRepository) ListPendingDeliveriesForEntry(entryID uint64) ([]model.APPendingStatusDelivery, error) {
	rows := make([]model.APPendingStatusDelivery, 0)
	err := r.db.Table("ap_statuses").
		Select("ap_statuses.id AS entry_id, owner_profiles.user_id AS user_id, ap_statuses.activity AS activity, follower_profiles.url AS actor_iri, follower_profiles.inbox_url AS actor_inbox").
		Joins("JOIN profiles owner_profiles ON owner_profiles.id = ap_statuses.profile_id").
		Joins("JOIN followers ON followers.following_profile_id = owner_profiles.id AND followers.approved = ?", true).
		Joins("JOIN profiles follower_profiles ON follower_profiles.id = followers.profile_id").
		Joins("LEFT JOIN ap_outbox_delivery ON ap_outbox_delivery.ap_status_id = ap_statuses.id AND ap_outbox_delivery.profile_id = follower_profiles.id").
		Where("ap_statuses.id = ?", entryID).
		Where("ap_outbox_delivery.id IS NULL").
		Where("owner_profiles.user_id IS NOT NULL").
		Where("follower_profiles.url IS NOT NULL").
		Where("follower_profiles.inbox_url IS NOT NULL").
		Find(&rows).
		Error

	return rows, err
}
