package migrations

import (
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"gorm.io/gorm"
)

func init() {
	model.RegisterMigration(2026030801, "migrate location data to separate track_locations table",
		func(db *gorm.DB) error {
			// Pre-AutoMigrate: fix map_data_details_points to use map_data_id directly
			if db.Migrator().HasColumn("map_data_details_points", "map_data_details_id") {
				if err := db.Exec("ALTER TABLE map_data_details_points ADD COLUMN map_data_id bigint").Error; err != nil {
					return err
				}
				if err := db.Exec("UPDATE map_data_details_points SET map_data_id = (SELECT map_data_id FROM map_data_details WHERE id = map_data_details_id)").Error; err != nil {
					return err
				}
				if err := db.Migrator().DropColumn("map_data_details_points", "map_data_details_id"); err != nil {
					return err
				}
			}

			return nil
		},
		func(db *gorm.DB) error {
			// Post-AutoMigrate: migrate location data from map_data to track_locations
			if db.Migrator().HasColumn("map_data", "address") {
				if err := db.Exec(`INSERT INTO track_locations (created_at, updated_at, map_data_id, address, address_string, center)
					SELECT created_at, updated_at, id, address, address_string, center
					FROM map_data
					WHERE address IS NOT NULL OR address_string != '' OR center IS NOT NULL`).Error; err != nil {
					return err
				}
				if err := db.Migrator().DropColumn("map_data", "address"); err != nil {
					return err
				}
				if err := db.Migrator().DropColumn("map_data", "address_string"); err != nil {
					return err
				}
				if err := db.Migrator().DropColumn("map_data", "center"); err != nil {
					return err
				}
			}

			if db.Migrator().HasTable("map_data_details") {
				if err := db.Migrator().DropTable("map_data_details"); err != nil {
					return err
				}
			}

			return nil
		},
		func(*gorm.DB) error { return nil },
		func(*gorm.DB) error { return nil },
	)
}
