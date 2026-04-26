package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/AepyornisNet/aepyornis/pkg/config"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/repository"
	"github.com/vgarvardt/gue/v6"
	"gorm.io/gorm"
)

const JobUpdateWorkout = "update_workout"

// EnqueueWorkoutUpdate enqueues a job to reprocess the given workout.
// Call this wherever a workout is created or marked dirty.
func EnqueueWorkoutUpdate(ctx context.Context, client *gue.Client, workoutID uint64) error {
	return enqueueJob(ctx, client, MainQueue, JobUpdateWorkout, idArgs{ID: workoutID})
}

func makeUpdateWorkoutHandler(
	db *gorm.DB,
	cfg *config.Config,
	client *gue.Client,
	logger *slog.Logger,
	apOutboxRepo repository.APOutbox,
	deliveryRepo repository.APStatusDelivery,
	userRepo repository.User,
	workoutRepo repository.Workout,
) gue.WorkFunc {
	return func(ctx context.Context, j *gue.Job) error {
		var args idArgs
		if err := json.Unmarshal(j.Args, &args); err != nil {
			return fmt.Errorf("update_workout: unmarshal args: %w", err)
		}

		l := logger.With("workout_id", args.ID)

		w, err := workoutRepo.GetDetailsByID(args.ID)
		if err != nil {
			return fmt.Errorf("update_workout: get workout %d: %w", args.ID, err)
		}

		if w.Dirty {
			l.Info("Updating workout")

			if err := w.UpdateData(db); err != nil {
				return err
			}

			if w.Data != nil && !w.Data.Center.IsZero() && w.Data.AddressString == "" {
				if err := EnqueueAddressUpdate(ctx, client, w.Data.ID); err != nil {
					l.Error("Failed to enqueue address update after workout processing", "error", err)
				}
			}
		}

		syncWorkoutAttachmentImage(db, l, w)

		if w.Profile == nil || w.Profile.UserID == nil {
			return nil
		}

		user, err := userRepo.GetByID(*w.Profile.UserID)
		if err != nil {
			return fmt.Errorf("update_workout: get user %d: %w", *w.Profile.UserID, err)
		}

		if err := SyncWorkoutActivityPub(ctx, client, db, cfg, apOutboxRepo, deliveryRepo, user, w, nil); err != nil {
			l.Warn("Failed to sync workout ActivityPub state", "error", err)
		}

		return nil
	}
}

func storeWorkoutAttachmentImage(db *gorm.DB, logger *slog.Logger, workout *model.Workout) {
	routeImageContent, routeImageErr := model.GenerateWorkoutRouteImage(workout)
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

func syncWorkoutAttachmentImage(db *gorm.DB, logger *slog.Logger, workout *model.Workout) {
	if model.WorkoutRoutePointCount(workout) < 2 {
		if err := model.DeleteRouteImageAttachment(db, workout.ID); err != nil {
			logger.Warn("Failed to remove workout route image attachment", "error", err)
		}
		return
	}

	storeWorkoutAttachmentImage(db, logger, workout)
}
