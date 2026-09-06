package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/repository"
	"github.com/vgarvardt/gue/v6"
	"gorm.io/gorm"
)

const JobUpdateRouteSegment = "update_route_segment"

// EnqueueRouteSegmentUpdate enqueues a job to re-match the given route segment.
// Call this wherever a route segment is created or marked dirty.
func EnqueueRouteSegmentUpdate(ctx context.Context, client *gue.Client, segmentID uint64) error {
	return enqueueJob(ctx, client, MainQueue, JobUpdateRouteSegment, idArgs{ID: segmentID})
}

func makeUpdateRouteSegmentHandler(db *gorm.DB, logger *slog.Logger, routeSegmentRepo repository.RouteSegment) gue.WorkFunc {
	return func(ctx context.Context, j *gue.Job) error {
		var args idArgs
		if err := json.Unmarshal(j.Args, &args); err != nil {
			return fmt.Errorf("update_route_segment: unmarshal args: %w", err)
		}

		l := logger.With("route_segment_id", args.ID)

		rs, err := routeSegmentRepo.GetByID(args.ID)
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
	l.Debug("rematchRouteSegmentToWorkouts start")

	if err := model.RematchRouteSegment(db, rs.ID); err != nil {
		return fmt.Errorf("error rematching route segment %d: %w", rs.ID, err)
	}

	l.Debug("rematchRouteSegmentToWorkouts done")
	return nil
}
