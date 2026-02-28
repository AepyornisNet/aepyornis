package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/jovandeginste/workout-tracker/v2/pkg/container"
	"github.com/jovandeginste/workout-tracker/v2/pkg/converters"
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/vgarvardt/gue/v6"
)

const JobAutoImport = "auto_import"

const fileAddDelay = -1 * time.Minute

var ErrNothingImported = errors.New("nothing imported")

func makeAutoImportHandler(c *container.Container, logger *slog.Logger) gue.WorkFunc {
	return func(ctx context.Context, j *gue.Job) error {
		if !c.GetConfig().AutoImportEnabled {
			logger.Debug("Skipping auto-import job because auto import is disabled")
			return nil
		}

		var args idArgs
		if err := json.Unmarshal(j.Args, &args); err != nil {
			return fmt.Errorf("auto_import: unmarshal args: %w", err)
		}

		return autoImportForUser(ctx, c, logger.With("user_id", args.ID), args.ID)
	}
}

func autoImportForUser(ctx context.Context, c *container.Container, l *slog.Logger, userID uint64) error {
	u, err := c.UserRepo().GetByID(userID)
	if err != nil {
		return err
	}

	ok, err := u.Profile.CanImportFromDirectory()
	if err != nil {
		return fmt.Errorf("could not use auto-import dir %v for user %v: %w", u.Profile.AutoImportDirectory, u.Username, err)
	}

	if !ok {
		return nil
	}

	l = l.With("user", u.Username)
	l.Info("Importing from '" + u.Profile.AutoImportDirectory + "'")

	files, err := filepath.Glob(filepath.Join(u.Profile.AutoImportDirectory, "*"))
	if err != nil {
		return err
	}

	for _, path := range files {
		pl := l.With("path", path)
		if err := importForUser(ctx, c, pl, u, path); err != nil {
			pl.Error("Could not import: " + err.Error())
		}
	}

	return nil
}

func importForUser(ctx context.Context, c *container.Container, logger *slog.Logger, u *model.User, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !fileCanBeImported(path, info) {
		return nil
	}

	if importErr := importFile(ctx, c, logger, u, path); importErr != nil {
		logger.Error("Could not import: " + importErr.Error())
		return moveImportFile(logger, u.Profile.AutoImportDirectory, path, "failed")
	}

	return moveImportFile(logger, u.Profile.AutoImportDirectory, path, "done")
}

func importFile(ctx context.Context, c *container.Container, logger *slog.Logger, u *model.User, path string) error {
	db := c.GetDB()

	logger.Info("Importing path")

	dat, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	ws, addErr := u.AddWorkout(db, model.WorkoutTypeAutoDetect, "", path, dat)
	if len(addErr) > 0 {
		return addErr[0]
	}

	if len(ws) == 0 {
		return ErrNothingImported
	}

	for _, w := range ws {
		if err := EnqueueWorkoutUpdate(ctx, c, w.ID); err != nil {
			logger.Error("Failed to enqueue workout update after import", "workout_id", w.ID, "error", err)
		}
	}

	logger.Info("Finished import.")

	return nil
}

func moveImportFile(logger *slog.Logger, dir, path, statusDir string) error {
	destDir := filepath.Join(dir, statusDir)
	destPath := filepath.Join(destDir, filepath.Base(path))

	logger.Info("Moving file", "src", path, "dst", destPath)

	if _, err := os.Stat(destDir); errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir(destDir, 0o755); err != nil {
			return err
		}
	}

	if err := os.Rename(path, destPath); err != nil {
		return err
	}

	logger.Info("Files moved", "destination", destDir)

	return nil
}

func fileCanBeImported(p string, i os.FileInfo) bool {
	if i.IsDir() {
		return false
	}

	if i.ModTime().After(time.Now().Add(fileAddDelay)) {
		return false
	}

	return slices.Contains(converters.SupportedFileTypes, strings.ToLower(filepath.Ext(p)))
}
