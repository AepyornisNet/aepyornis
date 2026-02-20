package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jovandeginste/workout-tracker/v2/pkg/container"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/vgarvardt/gue/v6"
)

const JobUpdateAddress = "update_address"

// EnqueueAddressUpdate enqueues a geocoding job for the given map data ID on the rate-limited geo queue.
func EnqueueAddressUpdate(ctx context.Context, c *container.Container, mapDataID uint64) error {
	raw, err := json.Marshal(idArgs{ID: mapDataID})
	if err != nil {
		return err
	}

	return c.Enqueue(ctx, &gue.Job{Queue: GeoQueue, Type: JobUpdateAddress, Args: raw})
}

func makeUpdateAddressHandler(c *container.Container, logger *slog.Logger) gue.WorkFunc {
	return func(ctx context.Context, j *gue.Job) error {
		db := c.GetDB()

		var args idArgs
		if err := json.Unmarshal(j.Args, &args); err != nil {
			return fmt.Errorf("update_address: unmarshal args: %w", err)
		}

		md, err := model.GetMapData(db, args.ID)
		if err != nil {
			return fmt.Errorf("update_address: get map data %d: %w", args.ID, err)
		}

		logger.With("map_data_id", md.ID).With("workout_id", md.WorkoutID).Info("Updating address")
		md.UpdateAddress()

		return md.Save(db)
	}
}
