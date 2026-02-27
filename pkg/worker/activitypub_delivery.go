package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	ap "github.com/jovandeginste/workout-tracker/v2/pkg/activitypub"
	"github.com/jovandeginste/workout-tracker/v2/pkg/container"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/vgarvardt/gue/v6"
)

const JobDeliverActivityPub = "deliver_activitypub"

// EnqueueAPDeliveriesForEntry queries all pending follower deliveries for the given outbox entry
// and enqueues one job per follower. Call this immediately after creating an AP outbox entry.
func EnqueueAPDeliveriesForEntry(ctx context.Context, c *container.Container, entryID uint64) error {
	pending, err := c.APOutboxDeliveryRepo().ListPendingDeliveriesForEntry(entryID)
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
		db := c.GetDB()
		cfg := c.GetConfig()

		var item model.APPendingOutboxDelivery
		if err := json.Unmarshal(j.Args, &item); err != nil {
			return fmt.Errorf("deliver_activitypub: unmarshal args: %w", err)
		}

		l := logger.With("entry_id", item.EntryID, "actor", item.ActorIRI)

		u, err := model.GetUserByID(db, item.UserID)
		if err != nil {
			return fmt.Errorf("deliver_activitypub: get user %d: %w", item.UserID, err)
		}

		if u.PrivateKey == "" {
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
		}, u.Username)

		deliverCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()

		if err := ap.SendSignedActivity(deliverCtx, actorURL, u.PrivateKey, item.ActorInbox, item.Activity); err != nil {
			return fmt.Errorf("deliver_activitypub: send to %s: %w", item.ActorIRI, err)
		}

		if err := c.APOutboxDeliveryRepo().RecordDelivery(item.EntryID, item.ActorIRI); err != nil {
			return fmt.Errorf("deliver_activitypub: record delivery: %w", err)
		}

		l.Info("ActivityPub delivery recorded")

		return nil
	}
}
