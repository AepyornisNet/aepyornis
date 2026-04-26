package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/aputil"
	"github.com/AepyornisNet/aepyornis/pkg/config"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/repository"
	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
	"github.com/google/uuid"
	"github.com/vgarvardt/gue/v6"
	"gorm.io/gorm"
)

func SyncWorkoutActivityPub(
	ctx context.Context,
	client *gue.Client,
	db *gorm.DB,
	cfg *config.Config,
	apOutboxRepo repository.APOutbox,
	deliveryRepo repository.APStatusDelivery,
	user *model.User,
	workout *model.Workout,
	previousVisibility *model.WorkoutVisibility,
) error {
	if user == nil || workout == nil {
		return nil
	}

	if previousVisibility != nil && *previousVisibility == workout.Visibility {
		return nil
	}

	entry, err := apOutboxRepo.GetEntryForWorkout(user.ID, workout.ID)
	hasOutboxEntry := err == nil
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	shouldPublish := user.ActivityPubEnabled() &&
		(workout.Visibility == model.WorkoutVisibilityPublic || workout.Visibility == model.WorkoutVisibilityFollowers)

	if !shouldPublish {
		if hasOutboxEntry {
			if err := apOutboxRepo.DeleteEntryForWorkout(user.ID, workout.ID); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}

		return nil
	}

	if hasOutboxEntry {
		return updateWorkoutActivityPubAudience(db, cfg, user, entry, workout)
	}

	return publishWorkoutToActivityPub(ctx, client, db, cfg, apOutboxRepo, deliveryRepo, user, workout)
}

func publishWorkoutToActivityPub(
	ctx context.Context,
	client *gue.Client,
	db *gorm.DB,
	cfg *config.Config,
	apOutboxRepo repository.APOutbox,
	deliveryRepo repository.APStatusDelivery,
	user *model.User,
	workout *model.Workout,
) error {
	fitContent, err := aputil.GenerateWorkoutFIT(workout)
	if err != nil {
		return err
	}

	actorURL, err := localActorURL(cfg, user)
	if err != nil {
		return err
	}

	entryUUID := uuid.New()
	entryURL := fmt.Sprintf("%s/outbox/%s", actorURL, entryUUID.String())
	objectURL := entryURL + "#object"
	fitURL := entryURL + "/fit"
	routeImageURL := entryURL + "/route-image"
	publishedAt := time.Now().UTC()
	noteContent := aputil.WorkoutNoteContent(workout)

	attachments := vocab.ItemCollection{}
	routeImageAttachment, routeImageErr := model.GetRouteImageAttachment(db, workout.ID)
	if routeImageErr == nil {
		attachments = append(attachments, &vocab.Object{
			Type:      vocab.ImageType,
			Name:      vocab.DefaultNaturalLanguage(routeImageAttachment.Filename),
			MediaType: vocab.MimeType(routeImageAttachment.ContentType),
			URL:       vocab.IRI(routeImageURL),
		})
	} else if !errors.Is(routeImageErr, gorm.ErrRecordNotFound) {
		return routeImageErr
	}

	note := aputil.NewWorkoutNote()
	note.ID = vocab.ID(objectURL)
	note.AttributedTo = vocab.IRI(actorURL)
	note.Published = publishedAt
	note.Content = vocab.DefaultNaturalLanguage(noteContent)
	note.Attachment = attachments
	note.PopulateFromWorkout(workout, vocab.IRI(fitURL))

	to := vocab.ItemCollection{vocab.IRI(actorURL + "/followers")}
	cc := vocab.ItemCollection{}
	if workout.Visibility == model.WorkoutVisibilityPublic {
		to = vocab.ItemCollection{vocab.IRI("https://www.w3.org/ns/activitystreams#Public")}
		cc = vocab.ItemCollection{vocab.IRI(actorURL + "/followers")}
	}

	activity := vocab.Activity{
		ID:        vocab.ID(entryURL),
		Type:      vocab.CreateType,
		Actor:     vocab.IRI(actorURL),
		Published: publishedAt,
		To:        to,
		CC:        cc,
		Object:    note,
	}

	activityJSON, err := jsonld.WithContext(aputil.WorkoutJSONLDContext()).Marshal(activity)
	if err != nil {
		return err
	}

	noteJSON, err := jsonld.WithContext(aputil.WorkoutJSONLDContext()).Marshal(note)
	if err != nil {
		return err
	}

	outboxWorkout := &model.APStatusWorkout{
		UserID:         user.ID,
		WorkoutID:      workout.ID,
		FitFilename:    aputil.WorkoutFITFilename(workout),
		FitContent:     fitContent,
		FitContentType: aputil.FitMIMEType,
	}

	if err := apOutboxRepo.CreateWorkout(outboxWorkout); err != nil {
		return err
	}

	entry := &model.APStatus{
		PublicUUID:        entryUUID,
		UserID:            &user.ID,
		APStatusWorkoutID: &outboxWorkout.ID,
		StatusType:        model.APStatusTypeWorkout,
		Origin:            "local",
		ActivityID:        entryURL,
		ObjectID:          objectURL,
		Activity:          activityJSON,
		Payload:           noteJSON,
		Content:           noteContent,
		PublishedAt:       &publishedAt,
	}

	if err := apOutboxRepo.CreateEntry(entry); err != nil {
		return err
	}

	return EnqueueAPDeliveriesForEntry(ctx, client, deliveryRepo, entry.ID)
}

func updateWorkoutActivityPubAudience(db *gorm.DB, cfg *config.Config, user *model.User, entry *model.APStatus, workout *model.Workout) error {
	if entry == nil {
		return errors.New("outbox entry is nil")
	}

	actorURL, err := localActorURL(cfg, user)
	if err != nil {
		return err
	}

	activity := vocab.Activity{}
	if err := jsonld.Unmarshal(entry.Activity, &activity); err != nil {
		return err
	}

	note := aputil.NewWorkoutNote()
	if len(entry.Payload) > 0 {
		if err := jsonld.Unmarshal(entry.Payload, note); err != nil {
			return err
		}
	}

	activity.To = vocab.ItemCollection{vocab.IRI(actorURL + "/followers")}
	activity.CC = vocab.ItemCollection{}
	activity.Object = note
	if workout.Visibility == model.WorkoutVisibilityPublic {
		activity.To = vocab.ItemCollection{vocab.IRI("https://www.w3.org/ns/activitystreams#Public")}
		activity.CC = vocab.ItemCollection{vocab.IRI(actorURL + "/followers")}
	}

	activityJSON, err := jsonld.WithContext(aputil.WorkoutJSONLDContext()).Marshal(activity)
	if err != nil {
		return err
	}

	return db.Model(&model.APStatus{}).
		Where("id = ?", entry.ID).
		Update("activity", activityJSON).Error
}

func localActorURL(cfg *config.Config, user *model.User) (string, error) {
	actorURL := aputil.LocalActorURL(aputil.LocalActorURLConfig{
		Host:           cfg.Host,
		WebRoot:        cfg.WebRoot,
		FallbackHost:   cfg.Host,
		FallbackScheme: "https",
	}, user.Profile.Username)

	if actorURL == "" {
		return "", errors.New("could not determine local actor URL")
	}

	return actorURL, nil
}
