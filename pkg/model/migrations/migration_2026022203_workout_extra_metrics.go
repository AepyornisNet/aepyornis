package migrations

import (
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"github.com/labstack/gommon/log"
	"gorm.io/gorm"
)

func init() {
	model.RegisterMigration(2026022203, "backfill workout extra metrics after automigrate",
		func(*gorm.DB) error {
			return nil
		},
		func(db *gorm.DB) error {
			workouts, err := model.GetWorkouts(db)
			if err != nil {
				return err
			}

			for _, workout := range workouts {
				if !workout.HasTracks() || workout.Data.ExtraMetrics != nil {
					continue
				}

				workout.Data.UpdateExtraMetrics()

				if err := workout.Save(db); err != nil {
					log.Error("Failed to update extra metrics", "err", err)
				}
			}

			return nil
		},
		func(*gorm.DB) error {
			return nil
		},
		func(*gorm.DB) error {
			return nil
		},
	)
}
