package migrations

import (
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"gorm.io/gorm"
)

func init() {
	model.RegisterMigration(
		202608161300,
		"migrate route_segment_matches composite primary key to surrogate id primary key",
		preRouteSegmentMatchesSurrogatePKMigrate,
		nil,
		nil,
		nil,
	)
}

func preRouteSegmentMatchesSurrogatePKMigrate(db *gorm.DB) error {
	if db.Migrator().HasTable("route_segment_matches") {
		if !db.Migrator().HasColumn("route_segment_matches", "id") {
			_ = db.Exec("ALTER TABLE route_segment_matches DROP CONSTRAINT IF EXISTS route_segment_matches_pkey").Error
			if db.Dialector.Name() == "postgres" {
				if err := db.Exec("ALTER TABLE route_segment_matches ADD COLUMN id BIGSERIAL PRIMARY KEY").Error; err != nil {
					return err
				}
			} else {
				if err := db.Exec("ALTER TABLE route_segment_matches ADD COLUMN id INTEGER PRIMARY KEY AUTOINCREMENT").Error; err != nil {
					return err
				}
			}
		}
		_ = db.Exec("CREATE INDEX IF NOT EXISTS idx_route_segment_matches_route_segment_id ON route_segment_matches(route_segment_id)").Error
		_ = db.Exec("CREATE INDEX IF NOT EXISTS idx_route_segment_matches_workout_id ON route_segment_matches(workout_id)").Error
		_ = db.Exec("CREATE INDEX IF NOT EXISTS idx_route_segment_matches_segment_workout ON route_segment_matches(route_segment_id, workout_id)").Error
	}
	return nil
}
