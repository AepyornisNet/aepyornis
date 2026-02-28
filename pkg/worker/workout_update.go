package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jovandeginste/workout-tracker/v2/pkg/container"
	"github.com/vgarvardt/gue/v6"
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

		if !w.Dirty {
			return nil
		}

		l.Info("Updating workout")

		if err := w.UpdateData(db); err != nil {
			return err
		}

		// If geocoding didn't produce an address, enqueue a dedicated retry on the geo queue.
		if w.Data != nil && !w.Data.Center.IsZero() && w.Data.AddressString == "" {
			if err := EnqueueAddressUpdate(ctx, c, w.Data.ID); err != nil {
				l.Error("Failed to enqueue address update after workout processing", "error", err)
			}
		}

		return nil
	}
}
