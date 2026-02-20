package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/container"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/vgarvardt/gue/v6"
	"gorm.io/gorm"
)

const (
	// MainQueue is the default queue used for most background tasks.
	MainQueue = ""
	// GeoQueue is the queue for geocoding tasks; limited to 1 worker to respect rate limits.
	GeoQueue = "geo"

	mainWorkerCount = 10
	geoWorkerCount  = 1

	gueJobsSchema = `
CREATE TABLE IF NOT EXISTS gue_jobs (
  job_id      TEXT        NOT NULL PRIMARY KEY,
  priority    SMALLINT    NOT NULL,
  run_at      TIMESTAMPTZ NOT NULL,
  job_type    TEXT        NOT NULL,
  args        BYTEA       NOT NULL,
  error_count INTEGER     NOT NULL DEFAULT 0,
  last_error  TEXT,
  queue       TEXT        NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL,
  updated_at  TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_gue_jobs_selector ON gue_jobs (queue, run_at, priority);
`
)

// Worker wraps gue worker pools and the scheduler that enqueues periodic jobs.
type Worker struct {
	client   *gue.Client
	mainPool *gue.WorkerPool
	geoPool  *gue.WorkerPool
	logger   *slog.Logger
	db       *gorm.DB
	cfg      *container.Config
	delay    time.Duration
}

// New creates a Worker using dependencies from the provided Container.
// It migrates the gue_jobs schema and builds the work maps.
func New(c *container.Container) (*Worker, error) {
	db := c.GetDB()
	cfg := c.GetConfig()
	logger := c.Logger().With("module", "worker")

	if err := db.Exec(gueJobsSchema).Error; err != nil {
		return nil, fmt.Errorf("worker: migrating gue_jobs schema: %w", err)
	}

	gc := c.GetGueClient()
	if gc == nil {
		return nil, fmt.Errorf("worker: missing gue client on container")
	}

	wm := gue.WorkMap{
		JobUpdateWorkout:      makeUpdateWorkoutHandler(c, logger),
		JobUpdateRouteSegment: makeUpdateRouteSegmentHandler(c, logger),
		JobAutoImport:         makeAutoImportHandler(c, logger),
		JobDeliverActivityPub: makeDeliverActivityPubHandler(c, logger),
	}

	geoWM := gue.WorkMap{
		JobUpdateAddress: makeUpdateAddressHandler(c, logger),
	}

	mainPool, err := gue.NewWorkerPool(gc, wm, mainWorkerCount)
	if err != nil {
		return nil, fmt.Errorf("worker: creating main worker pool: %w", err)
	}

	geoPool, err := gue.NewWorkerPool(gc, geoWM, geoWorkerCount, gue.WithPoolQueue(GeoQueue))
	if err != nil {
		return nil, fmt.Errorf("worker: creating geo worker pool: %w", err)
	}

	w := &Worker{
		client:   gc,
		mainPool: mainPool,
		geoPool:  geoPool,
		logger:   logger,
		db:       db,
		cfg:      cfg,
		delay:    time.Duration(cfg.WorkerDelaySeconds) * time.Second,
	}

	return w, nil
}

// Client returns the underlying gue.Client so callers can enqueue jobs directly.
func (w *Worker) Client() *gue.Client {
	return w.client
}

// Start runs the worker pools and the scheduler. It blocks until ctx is cancelled.
func (w *Worker) Start(ctx context.Context) {
	w.logger.Info("Background worker starting", "delay_seconds", w.delay.Seconds())

	go func() {
		if err := w.mainPool.Run(ctx); err != nil && ctx.Err() == nil {
			w.logger.Error("Main worker pool stopped unexpectedly", "error", err)
		}
	}()

	go func() {
		if err := w.geoPool.Run(ctx); err != nil && ctx.Err() == nil {
			w.logger.Error("Geo worker pool stopped unexpectedly", "error", err)
		}
	}()

	if !w.cfg.AutoImportEnabled {
		w.logger.Info("Auto-import scheduler disabled", "auto_import_enabled", false)
		<-ctx.Done()
		return
	}

	w.runScheduler(ctx)
}

// runScheduler periodically scans for work that has no direct event source and enqueues gue jobs.
// Currently only auto-imports require polling (filesystem changes have no push notification).
func (w *Worker) runScheduler(ctx context.Context) {
	for {
		w.scheduleOnce(ctx)

		select {
		case <-ctx.Done():
			return
		case <-time.After(w.delay):
		}
	}
}

func (w *Worker) scheduleOnce(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error(fmt.Sprintf("panic in scheduler: %#v", r))
			fmt.Println("stacktrace from panic:\n" + string(debug.Stack()))
		}
	}()

	w.logger.Info("Auto import check started")
	w.enqueueAutoImports(ctx)
	w.logger.Info("Auto import check finished")
}

func (w *Worker) enqueueAutoImports(ctx context.Context) {
	var ids []uint64
	if err := w.db.Model(&model.User{}).Pluck("id", &ids).Error; err != nil {
		w.logger.Error("enqueueAutoImports: query failed", "error", err)
		return
	}

	for _, id := range ids {
		w.enqueueJob(ctx, MainQueue, JobAutoImport, idArgs{ID: id})
	}
}

func (w *Worker) enqueueJob(ctx context.Context, queue, jobType string, args any) {
	raw, err := json.Marshal(args)
	if err != nil {
		w.logger.Error("Failed to marshal job args", "job_type", jobType, "error", err)
		return
	}

	j := &gue.Job{Queue: queue, Type: jobType, Args: raw}
	if err := w.client.Enqueue(ctx, j); err != nil {
		w.logger.Error("Failed to enqueue job", "job_type", jobType, "error", err)
	}
}

// idArgs is the shared JSON payload for all single-entity jobs.
type idArgs struct {
	ID uint64 `json:"id"`
}
