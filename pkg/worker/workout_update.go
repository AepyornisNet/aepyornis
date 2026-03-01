package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jovandeginste/workout-tracker/v2/pkg/container"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/vgarvardt/gue/v6"
	"gorm.io/gorm"
)

const JobUpdateWorkout = "update_workout"

// EnqueueWorkoutUpdate enqueues a job to reprocess the given workout.
// Call this wherever a workout is created or marked dirty.
func EnqueueWorkoutUpdate(ctx context.Context, c *container.Container, workoutID uint64) error {
	raw, err := json.Marshal(idArgs{ID: workoutID})
	if err != nil {
		return err
	}

	return c.Enqueue(ctx, &gue.Job{Queue: MainQueue, Type: JobUpdateWorkout, Args: raw})
}

func makeUpdateWorkoutHandler(c *container.Container, logger *slog.Logger) gue.WorkFunc {
	return func(ctx context.Context, j *gue.Job) error {
		db := c.GetDB()

		var args idArgs
		if err := json.Unmarshal(j.Args, &args); err != nil {
			return fmt.Errorf("update_workout: unmarshal args: %w", err)
		}

		l := logger.With("workout_id", args.ID)

		w, err := c.WorkoutRepo().GetDetailsByID(args.ID)
		if err != nil {
			return fmt.Errorf("update_workout: get workout %d: %w", args.ID, err)
		}

		if w.Dirty {
			l.Info("Updating workout")

			if err := w.UpdateData(db); err != nil {
				return err
			}

			storeWorkoutAttachmentImage(db, l, w)

			if w.Data != nil && !w.Data.Center.IsZero() && w.Data.AddressString == "" {
				if err := EnqueueAddressUpdate(ctx, c, w.Data.ID); err != nil {
					l.Error("Failed to enqueue address update after workout processing", "error", err)
				}
			}
		}

		if _, err := model.GetRouteImageAttachment(db, w.ID); errors.Is(err, gorm.ErrRecordNotFound) {
			storeWorkoutAttachmentImage(db, l, w)
		} else if err != nil {
			l.Warn("Failed to check existing workout attachment image", "error", err)
		}

		user, err := c.UserRepo().GetByID(w.UserID)
		if err != nil {
			return fmt.Errorf("update_workout: get user %d: %w", w.UserID, err)
		}

		if err := SyncWorkoutActivityPub(ctx, c, user, w, nil); err != nil {
			l.Warn("Failed to sync workout ActivityPub state", "error", err)
		}

		return nil
	}
}

func storeWorkoutAttachmentImage(db *gorm.DB, logger *slog.Logger, workout *model.Workout) {
	routeImageContent, routeImageErr := model.GenerateWorkoutAttachmentImage(workout)
	if routeImageErr != nil {
		logger.Warn("Failed to generate workout attachment image", "error", routeImageErr)
		return
	}

	if _, err := model.UpsertRouteImageAttachment(
		db,
		workout.ID,
		model.WorkoutRouteImageFilename(workout),
		model.RouteImageMIMEType,
		routeImageContent,
	); err != nil {
		logger.Error("Failed to store workout route image attachment", "error", err)
	}
}
