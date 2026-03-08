package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	vocab "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
	"github.com/google/uuid"
	ap "github.com/jovandeginste/workout-tracker/v2/pkg/activitypub"
	"github.com/jovandeginste/workout-tracker/v2/pkg/container"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"gorm.io/gorm"
)

func SyncWorkoutActivityPub(ctx context.Context, c *container.Container, user *model.User, workout *model.Workout, previousVisibility *model.WorkoutVisibility) error {
	if user == nil || workout == nil {
		return nil
	}

	entry, err := c.APOutboxRepo().GetEntryForWorkout(user.ID, workout.ID)
	hasOutboxEntry := err == nil
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	shouldPublish := user.ActivityPubEnabled() &&
		(workout.Visibility == model.WorkoutVisibilityPublic || workout.Visibility == model.WorkoutVisibilityFollowers)

	if !shouldPublish {
		if hasOutboxEntry {
			if err := c.APOutboxRepo().DeleteEntryForWorkout(user.ID, workout.ID); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}

		return nil
	}

	if hasOutboxEntry {
		visibilityChanged := previousVisibility != nil && *previousVisibility != workout.Visibility
		if visibilityChanged {
			return updateWorkoutActivityPubAudience(c, user, entry, workout)
		}

		// Content may have changed – send an Update activity
		return updateWorkoutActivityPub(ctx, c, user, entry, workout)
	}

	return publishWorkoutToActivityPub(ctx, c, user, workout)
}

func publishWorkoutToActivityPub(ctx context.Context, c *container.Container, user *model.User, workout *model.Workout) error {
	fitContent, err := ap.GenerateWorkoutFIT(workout)
	if err != nil {
		return err
	}

	actorURL, err := localActorURL(c, user)
	if err != nil {
		return err
	}

	entryUUID := uuid.New()
	entryURL := fmt.Sprintf("%s/outbox/%s", actorURL, entryUUID.String())
	objectURL := entryURL + "#object"
	fitURL := entryURL + "/fit"
	routeImageURL := entryURL + "/route-image"
	publishedAt := time.Now().UTC()
	noteContent := ap.WorkoutNoteContent(workout)

	attachments := vocab.ItemCollection{}
	routeImageAttachment, routeImageErr := model.GetRouteImageAttachment(c.GetDB(), workout.ID)
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

	note := ap.NewWorkoutNote()
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

	activityJSON, err := jsonld.WithContext(ap.WorkoutJSONLDContext()).Marshal(activity)
	if err != nil {
		return err
	}

	noteJSON, err := jsonld.WithContext(ap.WorkoutJSONLDContext()).Marshal(note)
	if err != nil {
		return err
	}

	outboxWorkout := &model.APOutboxWorkout{
		UserID:         user.ID,
		WorkoutID:      workout.ID,
		FitFilename:    ap.WorkoutFITFilename(workout),
		FitContent:     fitContent,
		FitContentType: ap.FitMIMEType,
	}

	if err := c.APOutboxRepo().CreateWorkout(outboxWorkout); err != nil {
		return err
	}

	entry := &model.APOutboxEntry{
		PublicUUID:        entryUUID,
		UserID:            user.ID,
		APOutboxWorkoutID: &outboxWorkout.ID,
		Kind:              model.APOutboxWorkoutKind,
		ActivityID:        entryURL,
		ObjectID:          objectURL,
		Activity:          activityJSON,
		Payload:           noteJSON,
		NoteText:          noteContent,
		PublishedAt:       publishedAt,
	}

	if err := c.APOutboxRepo().CreateEntry(entry); err != nil {
		return err
	}

	return EnqueueAPDeliveriesForEntry(ctx, c, entry.ID)
}

func updateWorkoutActivityPubAudience(c *container.Container, user *model.User, entry *model.APOutboxEntry, workout *model.Workout) error {
	if entry == nil {
		return errors.New("outbox entry is nil")
	}

	actorURL, err := localActorURL(c, user)
	if err != nil {
		return err
	}

	activity := vocab.Activity{}
	if err := jsonld.Unmarshal(entry.Activity, &activity); err != nil {
		return err
	}

	note := ap.NewWorkoutNote()
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

	activityJSON, err := jsonld.WithContext(ap.WorkoutJSONLDContext()).Marshal(activity)
	if err != nil {
		return err
	}

	return c.GetDB().Model(&model.APOutboxEntry{}).
		Where("id = ?", entry.ID).
		Update("activity", activityJSON).Error
}

// updateWorkoutActivityPub rebuilds the workout note, stores the updated
// payload, creates an Update activity, and delivers it to followers.
func updateWorkoutActivityPub(ctx context.Context, c *container.Container, user *model.User, entry *model.APOutboxEntry, workout *model.Workout) error {
	if entry == nil {
		return errors.New("outbox entry is nil")
	}

	actorURL, err := localActorURL(c, user)
	if err != nil {
		return err
	}

	// Regenerate FIT content
	fitContent, fitErr := ap.GenerateWorkoutFIT(workout)
	if fitErr != nil {
		return fitErr
	}

	routeImageURL := entry.ActivityID + "/route-image"
	fitURL := entry.ActivityID + "/fit"
	noteContent := ap.WorkoutNoteContent(workout)

	attachments := vocab.ItemCollection{}
	routeImageAttachment, routeImageErr := model.GetRouteImageAttachment(c.GetDB(), workout.ID)
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

	note := ap.NewWorkoutNote()
	note.ID = vocab.ID(entry.ObjectID)
	note.AttributedTo = vocab.IRI(actorURL)
	note.Published = entry.PublishedAt
	note.Updated = time.Now().UTC()
	note.Content = vocab.DefaultNaturalLanguage(noteContent)
	note.Attachment = attachments
	note.PopulateFromWorkout(workout, vocab.IRI(fitURL))

	to := vocab.ItemCollection{vocab.IRI(actorURL + "/followers")}
	cc := vocab.ItemCollection{}
	if workout.Visibility == model.WorkoutVisibilityPublic {
		to = vocab.ItemCollection{vocab.IRI("https://www.w3.org/ns/activitystreams#Public")}
		cc = vocab.ItemCollection{vocab.IRI(actorURL + "/followers")}
	}

	updateURL := fmt.Sprintf("%s#update-%d", entry.ActivityID, time.Now().Unix())
	activity := vocab.Activity{
		ID:        vocab.ID(updateURL),
		Type:      vocab.UpdateType,
		Actor:     vocab.IRI(actorURL),
		Published: time.Now().UTC(),
		To:        to,
		CC:        cc,
		Object:    note,
	}

	activityJSON, err := jsonld.WithContext(ap.WorkoutJSONLDContext()).Marshal(activity)
	if err != nil {
		return err
	}

	noteJSON, err := jsonld.WithContext(ap.WorkoutJSONLDContext()).Marshal(note)
	if err != nil {
		return err
	}

	// Update the existing outbox entry and workout data
	if err := c.GetDB().Model(&model.APOutboxEntry{}).
		Where("id = ?", entry.ID).
		Updates(map[string]any{
			"activity":  activityJSON,
			"payload":   noteJSON,
			"note_text": noteContent,
		}).Error; err != nil {
		return err
	}

	if err := c.GetDB().Model(&model.APOutboxWorkout{}).
		Where("id = ?", entry.APOutboxWorkoutID).
		Updates(map[string]any{
			"fit_content":  fitContent,
			"fit_filename": ap.WorkoutFITFilename(workout),
		}).Error; err != nil {
		return err
	}

	return EnqueueAPDeliveriesForEntry(ctx, c, entry.ID)
}

// DeleteWorkoutActivityPub sends a Delete activity to all followers of the user
// who previously received the workout via ActivityPub.
func DeleteWorkoutActivityPub(ctx context.Context, c *container.Container, user *model.User, workout *model.Workout) error {
	if user == nil || workout == nil {
		return nil
	}

	entry, err := c.APOutboxRepo().GetEntryForWorkout(user.ID, workout.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // never published, nothing to do
		}

		return err
	}

	actorURL, err := localActorURL(c, user)
	if err != nil {
		return err
	}

	to := vocab.ItemCollection{vocab.IRI(actorURL + "/followers")}
	cc := vocab.ItemCollection{}

	deleteURL := fmt.Sprintf("%s#delete-%d", entry.ActivityID, time.Now().Unix())
	activity := vocab.Activity{
		ID:        vocab.ID(deleteURL),
		Type:      vocab.DeleteType,
		Actor:     vocab.IRI(actorURL),
		Published: time.Now().UTC(),
		To:        to,
		CC:        cc,
		Object:    vocab.IRI(entry.ObjectID),
	}

	activityJSON, err := jsonld.WithContext(ap.WorkoutJSONLDContext()).Marshal(activity)
	if err != nil {
		return err
	}

	// Overwrite the activity in the outbox entry with the Delete activity
	// so the delivery mechanism picks it up.
	if err := c.GetDB().Model(&model.APOutboxEntry{}).
		Where("id = ?", entry.ID).
		Update("activity", activityJSON).Error; err != nil {
		return err
	}

	if err := EnqueueAPDeliveriesForEntry(ctx, c, entry.ID); err != nil {
		return err
	}

	// Now remove the outbox entry
	return c.APOutboxRepo().DeleteEntryForWorkout(user.ID, workout.ID)
}

func localActorURL(c *container.Container, user *model.User) (string, error) {
	actorURL := ap.LocalActorURL(ap.LocalActorURLConfig{
		Host:           c.GetConfig().Host,
		WebRoot:        c.GetConfig().WebRoot,
		FallbackHost:   c.GetConfig().Host,
		FallbackScheme: "https",
	}, user.Username)

	if actorURL == "" {
		return "", errors.New("could not determine local actor URL")
	}

	return actorURL, nil
}
