package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	ap "github.com/AepyornisNet/aepyornis/pkg/activitypub"
	"github.com/AepyornisNet/aepyornis/pkg/container"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/vgarvardt/gue/v6"
)

const JobDeliverActivityPub = "deliver_activitypub"

// EnqueueAPDeliveriesForEntry queries all pending follower deliveries for the given outbox entry
// and enqueues one job per follower. Call this immediately after creating an AP outbox entry.
func EnqueueAPDeliveriesForEntry(ctx context.Context, c *container.Container, entryID uint64) error {
	pending, err := c.APStatusDeliveryRepo().ListPendingDeliveriesForEntry(entryID)
	if err != nil {
		return fmt.Errorf("EnqueueAPDeliveriesForEntry: list deliveries: %w", err)
	}

	for i := range pending {
		raw, err := json.Marshal(pending[i])
		if err != nil {
			return fmt.Errorf("EnqueueAPDeliveriesForEntry: marshal delivery: %w", err)
		}

		if err := c.Enqueue(ctx, &gue.Job{Queue: MainQueue, Type: JobDeliverActivityPub, Args: raw}); err != nil {
			return fmt.Errorf("EnqueueAPDeliveriesForEntry: enqueue delivery for %s: %w", pending[i].ActorIRI, err)
		}
	}

	return nil
}

func makeDeliverActivityPubHandler(c *container.Container, logger *slog.Logger) gue.WorkFunc {
	return func(ctx context.Context, j *gue.Job) error {
		cfg := c.GetConfig()

		var item model.APPendingStatusDelivery
		if err := json.Unmarshal(j.Args, &item); err != nil {
			return fmt.Errorf("deliver_activitypub: unmarshal args: %w", err)
		}

		l := logger.With("entry_id", item.EntryID, "actor", item.ActorIRI)

		u, err := c.UserRepo().GetByID(item.UserID)
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

		actorURL := ap.LocalActorURL(ap.LocalActorURLConfig{
			Host:           cfg.Host,
			WebRoot:        cfg.WebRoot,
			FallbackHost:   "",
			FallbackScheme: "https",
		}, u.Profile.Username)

		deliverCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()

		if err := ap.SendSignedActivity(deliverCtx, actorURL, u.Profile.PrivateKey, item.ActorInbox, item.Activity); err != nil {
			return fmt.Errorf("deliver_activitypub: send to %s: %w", item.ActorIRI, err)
		}

		if err := c.APStatusDeliveryRepo().RecordDelivery(item.EntryID, item.ActorIRI); err != nil {
			return fmt.Errorf("deliver_activitypub: record delivery: %w", err)
		}

		l.Info("ActivityPub delivery recorded")

		return nil
	}
}
