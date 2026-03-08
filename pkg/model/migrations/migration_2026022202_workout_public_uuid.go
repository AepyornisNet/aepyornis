package migrations

import (
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"gorm.io/gorm"
)

func init() {
	model.RegisterMigration(2026022202, "migrate workout public_uuid into visibility",
		func(*gorm.DB) error {
			return nil
		},
		func(db *gorm.DB) error {
			if !db.Migrator().HasColumn("workouts", "public_uuid") {
				return nil
			}

			if err := db.Exec("UPDATE workouts SET visibility = 'public' WHERE public_uuid IS NOT NULL").Error; err != nil {
				return err
			}

			return db.Migrator().DropColumn("workouts", "public_uuid")
		},
		func(*gorm.DB) error {
			return nil
		},
		func(*gorm.DB) error {
			return nil
		},
	)
}
