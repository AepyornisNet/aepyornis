package migrations

import (
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"gorm.io/gorm"
)

func init() {
	model.RegisterMigration(2026022201, "normalize legacy workout and map data before automigrate",
		func(db *gorm.DB) error {
			if !db.Migrator().HasTable("map_data") {
				return nil
			}

			// column is from a hardcoded allowlist above; no injection risk.
			for _, column := range []string{"average_speed", "average_speed_no_pause"} {
				if !db.Migrator().HasColumn("map_data", column) {
					continue
				}

				if err := db.Exec("UPDATE map_data SET " + column + " = 0 WHERE " + column + " > 1e308 OR " + column + " < -1e308").Error; err != nil {
					return err
				}
			}

			if err := db.Exec("DELETE FROM map_data WHERE id < (SELECT max(id) FROM map_data AS m WHERE m.workout_id = map_data.workout_id)").Error; err != nil {
				return err
			}

			if err := db.Exec("DELETE FROM workouts WHERE id < (SELECT max(id) FROM workouts AS w WHERE w.date = workouts.date AND w.user_id = workouts.user_id)").Error; err != nil {
				return err
			}

			if db.Migrator().HasTable("map_data_details") {
				if err := db.Exec("DELETE FROM map_data_details WHERE map_data_id IN (SELECT map_data_id FROM map_data_details AS mdd WHERE map_data_details.created_at < mdd.created_at)").Error; err != nil {
					return err
				}
			}

			return db.Exec("UPDATE workouts SET type = 'weight-lifting' WHERE type = 'weight lifting'").Error
		},
		func(*gorm.DB) error {
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
