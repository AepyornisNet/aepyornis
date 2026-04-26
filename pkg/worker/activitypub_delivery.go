package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/config"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/repository"
	"github.com/AepyornisNet/aepyornis/pkg/service"
	"github.com/vgarvardt/gue/v6"
)

const JobDeliverActivityPub = "deliver_activitypub"

// EnqueueAPDeliveriesForEntry queries all pending follower deliveries for the given outbox entry
// and enqueues one job per follower. Call this immediately after creating an AP outbox entry.
func EnqueueAPDeliveriesForEntry(ctx context.Context, client *gue.Client, deliveryRepo repository.APStatusDelivery, entryID uint64) error {
	pending, err := deliveryRepo.ListPendingDeliveriesForEntry(entryID)
	if err != nil {
		return fmt.Errorf("EnqueueAPDeliveriesForEntry: list deliveries: %w", err)
	}

	for i := range pending {
		if err := enqueueJob(ctx, client, MainQueue, JobDeliverActivityPub, pending[i]); err != nil {
			return fmt.Errorf("EnqueueAPDeliveriesForEntry: enqueue delivery for %s: %w", pending[i].ActorIRI, err)
		}
	}

	return nil
}

func makeDeliverActivityPubHandler(cfg *config.Config, logger *slog.Logger, deliveryRepo repository.APStatusDelivery, userRepo repository.User, actorService service.ActivityPubActorService) gue.WorkFunc {
	return func(ctx context.Context, j *gue.Job) error {
		var item model.APPendingStatusDelivery
		if err := json.Unmarshal(j.Args, &item); err != nil {
			return fmt.Errorf("deliver_activitypub: unmarshal args: %w", err)
		}

		l := logger.With("entry_id", item.EntryID, "actor", item.ActorIRI)

		u, err := userRepo.GetByID(item.UserID)
		if err != nil {
			return fmt.Errorf("deliver_activitypub: get user %d: %w", item.UserID, err)
		}

		if u.Profile.PrivateKey == "" {
			l.Warn("Skipping ActivityPub delivery due to missing private key", "user_id", u.ID)
			return nil
		}

		if cfg.Host == "" {
			l.Warn("Skipping ActivityPub delivery due to missing host configuration")
			return nil
		}

		deliverCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()

		if err := actorService.SendActivity(deliverCtx, &u.Profile, item.ActorInbox, item.Activity); err != nil {
			return fmt.Errorf("deliver_activitypub: send to %s: %w", item.ActorIRI, err)
		}

		if err := deliveryRepo.RecordDelivery(item.EntryID, item.ActorIRI); err != nil {
			return fmt.Errorf("deliver_activitypub: record delivery: %w", err)
		}

		l.Info("ActivityPub delivery recorded")

		return nil
	}
}
