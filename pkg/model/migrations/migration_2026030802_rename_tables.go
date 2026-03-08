package migrations

import (
	"fmt"

	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"gorm.io/gorm"
)

func init() {
	model.RegisterMigration(2026030802, "rename map_data tables to track_data, data_points, segments",
		func(db *gorm.DB) error {
			// Only run if old table names exist
			if !db.Migrator().HasTable("map_data") {
				return nil
			}

			// Drop old named indexes before renaming tables
			_ = db.Migrator().DropIndex("map_data_climbs", "idx_map_data_climbs_parent_order")
			_ = db.Migrator().DropIndex("map_data_details_points", "idx_map_data_details_points_parent_order")

			// Rename FK columns in related tables
			if db.Migrator().HasColumn("track_locations", "map_data_id") {
				if err := db.Migrator().RenameColumn("track_locations", "map_data_id", "track_data_id"); err != nil {
					return fmt.Errorf("rename track_locations.map_data_id: %w", err)
				}
			}
			if db.Migrator().HasColumn("map_data_details_points", "map_data_id") {
				if err := db.Migrator().RenameColumn("map_data_details_points", "map_data_id", "track_data_id"); err != nil {
					return fmt.Errorf("rename map_data_details_points.map_data_id: %w", err)
				}
			}
			if db.Migrator().HasColumn("map_data_climbs", "map_data_id") {
				if err := db.Migrator().RenameColumn("map_data_climbs", "map_data_id", "track_data_id"); err != nil {
					return fmt.Errorf("rename map_data_climbs.map_data_id: %w", err)
				}
			}

			// Rename tables
			if err := db.Migrator().RenameTable("map_data", "track_data"); err != nil {
				return fmt.Errorf("rename map_data to track_data: %w", err)
			}
			if db.Migrator().HasTable("map_data_details_points") {
				if err := db.Migrator().RenameTable("map_data_details_points", "data_points"); err != nil {
					return fmt.Errorf("rename map_data_details_points to data_points: %w", err)
				}
			}
			if db.Migrator().HasTable("map_data_climbs") {
				if err := db.Migrator().RenameTable("map_data_climbs", "segments"); err != nil {
					return fmt.Errorf("rename map_data_climbs to segments: %w", err)
				}
			}

			return nil
		},
		func(*gorm.DB) error { return nil },
		func(*gorm.DB) error { return nil },
		func(*gorm.DB) error { return nil },
	)
}
