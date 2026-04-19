package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	ap "github.com/AepyornisNet/aepyornis/pkg/activitypub"
	"github.com/AepyornisNet/aepyornis/pkg/container"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func PublishReplyToActivityPub(ctx context.Context, c *container.Container, author *model.User, workout *model.Workout, reply *model.APStatus) error {
	if c == nil || author == nil || workout == nil || reply == nil {
		return nil
	}

	if !author.ActivityPubEnabled() || author.PrivateKey == "" {
		return nil
	}

	parentEntry, err := c.APOutboxRepo().GetEntryForWorkout(workout.UserID, workout.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}

		return err
	}

	actorURL, err := localActorURL(c, author)
	if err != nil {
		return err
	}

	to, cc := parentActivityAudience(parentEntry.Activity, actorURL)
	publishedAt := time.Now().UTC()

	createUUID := uuid.New()
	createActivityID := fmt.Sprintf("%s/outbox/%s", actorURL, createUUID.String())
	createObjectID := createActivityID + "#object"

	note := ap.NewWorkoutNote()
	note.ID = vocab.ID(createObjectID)
	note.AttributedTo = vocab.IRI(actorURL)
	note.Published = publishedAt
	note.Content = vocab.DefaultNaturalLanguage(reply.Content)
	note.SetInReplyTo(vocab.IRI(parentEntry.ObjectID))

	createActivity := vocab.Activity{
		ID:        vocab.ID(createActivityID),
		Type:      vocab.CreateType,
		Actor:     vocab.IRI(actorURL),
		Published: publishedAt,
		To:        to,
		CC:        cc,
		Object:    note,
	}

	createActivityJSON, err := jsonld.WithContext(ap.WorkoutJSONLDContext()).Marshal(createActivity)
	if err != nil {
		return err
	}

	noteJSON, err := jsonld.WithContext(ap.WorkoutJSONLDContext()).Marshal(note)
	if err != nil {
		return err
	}

	if err := c.GetDB().Model(&model.APStatus{}).Where("id = ?", reply.ID).Updates(map[string]any{
		"activity_id":  createActivityID,
		"object_id":    createObjectID,
		"activity":     createActivityJSON,
		"payload":      noteJSON,
		"published_at": publishedAt,
		"origin":       "local",
		"status_type":  model.APStatusTypeReply,
		"user_id":      author.ID,
	}).Error; err != nil {
		return err
	}
	reply.ActivityID = createActivityID
	reply.ObjectID = createObjectID
	reply.Activity = createActivityJSON
	reply.Payload = noteJSON
	reply.PublishedAt = &publishedAt

	if err := EnqueueAPDeliveriesForEntry(ctx, c, reply.ID); err != nil {
		return err
	}

	return nil
}

func parentActivityAudience(raw []byte, actorURL string) (vocab.ItemCollection, vocab.ItemCollection) {
	activity := vocab.Activity{}
	if err := jsonld.Unmarshal(raw, &activity); err == nil {
		if len(activity.To) > 0 || len(activity.CC) > 0 {
			return activity.To, activity.CC
		}
	}

	return vocab.ItemCollection{vocab.IRI(actorURL + "/followers")}, vocab.ItemCollection{}
}
