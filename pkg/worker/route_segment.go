package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jovandeginste/workout-tracker/v2/pkg/container"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/vgarvardt/gue/v6"
	"gorm.io/gorm"
)

const JobUpdateRouteSegment = "update_route_segment"

const workerWorkoutsBatchSize = 10

// EnqueueRouteSegmentUpdate enqueues a job to re-match the given route segment.
// Call this wherever a route segment is created or marked dirty.
func EnqueueRouteSegmentUpdate(ctx context.Context, c *container.Container, segmentID uint64) error {
	raw, err := json.Marshal(idArgs{ID: segmentID})
	if err != nil {
		return err
	}

	return c.Enqueue(ctx, &gue.Job{Queue: MainQueue, Type: JobUpdateRouteSegment, Args: raw})
}

func makeUpdateRouteSegmentHandler(c *container.Container, logger *slog.Logger) gue.WorkFunc {
	return func(ctx context.Context, j *gue.Job) error {
		db := c.GetDB()

		var args idArgs
		if err := json.Unmarshal(j.Args, &args); err != nil {
			return fmt.Errorf("update_route_segment: unmarshal args: %w", err)
		}

		l := logger.With("route_segment_id", args.ID)

		rs, err := c.RouteSegmentRepo().GetByID(args.ID)
		if err != nil {
			return fmt.Errorf("update_route_segment: get route segment %d: %w", args.ID, err)
		}

		if !rs.Dirty {
			return nil
		}

		l.Info("Updating route segment")

		return rematchRouteSegmentToWorkouts(db, rs, l)
	}
}

func rematchRouteSegmentToWorkouts(db *gorm.DB, rs *model.RouteSegment, l *slog.Logger) error {
	rs.RouteSegmentMatches = []*model.RouteSegmentMatch{}

	var workoutsBatch []*model.Workout
	qw := model.PreloadWorkoutDetails(db).Preload("User").Model(&model.Workout{}).
		FindInBatches(&workoutsBatch, workerWorkoutsBatchSize, func(wtx *gorm.DB, batchNo int) error {
			l.With("batch_no", batchNo).
				With("workouts_batch_size", len(workoutsBatch)).
				Debug("rematchRouteSegmentsToWorkouts start")

			newMatches := rs.FindMatches(workoutsBatch)
			rs.RouteSegmentMatches = append(rs.RouteSegmentMatches, newMatches...)

			l.With("route_segment_id", rs.ID).
				With("new_matches", len(newMatches)).
				With("total_matches", len(rs.RouteSegmentMatches)).
				Debug("Updating route segments")

			l.With("batch_no", batchNo).
				With("workouts_batch_size", len(workoutsBatch)).
				Debug("rematchRouteSegmentsToWorkouts done")

			return nil
		})

	if qw.Error != nil {
		return qw.Error
	}

	rs.Dirty = false

	return rs.Save(db)
}
