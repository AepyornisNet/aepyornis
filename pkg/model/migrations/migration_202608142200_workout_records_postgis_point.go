package migrations

import (
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"gorm.io/gorm"
)

func init() {
	model.RegisterMigration(
		202608142200,
		"migrate workout_records lat and lng columns to PostGIS geometry point",
		preWorkoutRecordsPointMigrate,
		postWorkoutRecordsPointMigrate,
		nil,
		nil,
	)
}

func preWorkoutRecordsPointMigrate(db *gorm.DB) error {
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS postgis").Error; err != nil {
		return err
	}

	if db.Migrator().HasTable("workout_records") {
		if !db.Migrator().HasColumn("workout_records", "point") {
			if err := db.Exec("ALTER TABLE workout_records ADD COLUMN IF NOT EXISTS point geometry(Point, 4326)").Error; err != nil {
				return err
			}
		}

		if db.Migrator().HasColumn("workout_records", "lat") && db.Migrator().HasColumn("workout_records", "lng") {
			if err := db.Exec("UPDATE workout_records SET point = ST_SetSRID(ST_MakePoint(lng, lat), 4326) WHERE point IS NULL").Error; err != nil {
				return err
			}
			if err := db.Exec("ALTER TABLE workout_records DROP COLUMN IF EXISTS lat").Error; err != nil {
				return err
			}
			if err := db.Exec("ALTER TABLE workout_records DROP COLUMN IF EXISTS lng").Error; err != nil {
				return err
			}
		}
	}

	return nil
}

func postWorkoutRecordsPointMigrate(db *gorm.DB) error {
	if db.Migrator().HasTable("workout_records") {
		_ = db.Exec("CREATE INDEX IF NOT EXISTS idx_workout_records_point ON workout_records USING GIST (point)").Error
	}
	return nil
}
