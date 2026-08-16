package migrations

import (
	"errors"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"gorm.io/gorm"
)

func init() {
	model.RegisterMigration(
		202608091410,
		"backfill owner profile_id and default visibility for route segments",
		preAutoMigrateRouteSegments,
		nil,
		nil,
		nil,
	)
}

func preAutoMigrateRouteSegments(db *gorm.DB) error {
	if !db.Migrator().HasTable("route_segments") {
		return nil
	}

	if !db.Migrator().HasColumn("route_segments", "profile_id") {
		// Add profile_id column without NOT NULL constraint first so existing rows can be backfilled
		_ = db.Exec("ALTER TABLE route_segments ADD COLUMN profile_id bigint").Error
	}

	var adminProfileID uint64

	var adminUser model.User
	err := db.Preload("Profile").Where("admin = ?", true).Order("id ASC").First(&adminUser).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			var firstUser model.User
			if errFirst := db.Preload("Profile").Order("id ASC").First(&firstUser).Error; errFirst == nil {
				adminProfileID = firstUser.Profile.ID
			}
		} else {
			return err
		}
	} else {
		adminProfileID = adminUser.Profile.ID
	}

	if adminProfileID > 0 {
		if err := db.Exec("UPDATE route_segments SET profile_id = ? WHERE profile_id IS NULL OR profile_id = 0", adminProfileID).Error; err != nil {
			return err
		}
	}

	if db.Migrator().HasColumn("route_segments", "visibility") {
		if err := db.Exec("UPDATE route_segments SET visibility = ? WHERE visibility IS NULL OR visibility = ''", model.WorkoutVisibilityPublic).Error; err != nil {
			return err
		}
	}

	return nil
}
