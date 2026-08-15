package migrations

import (
	"encoding/json"
	"strings"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/restayway/gogis"
	"gorm.io/gorm"
)

func init() {
	model.RegisterMigration(
		202608142300,
		"migrate route_segments points column to PostGIS geometry linestring",
		preRouteSegmentsLineStringMigrate,
		postRouteSegmentsLineStringMigrate,
		nil,
		nil,
	)
}

func preRouteSegmentsLineStringMigrate(db *gorm.DB) error {
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS postgis").Error; err != nil {
		return err
	}

	if db.Migrator().HasTable("route_segments") {
		if db.Migrator().HasColumn("route_segments", "points") {
			var udtName string
			_ = db.Raw(`
				SELECT udt_name 
				FROM information_schema.columns 
				WHERE table_name = 'route_segments' AND column_name = 'points'
			`).Scan(&udtName).Error

			if udtName != "geometry" {
				if err := db.Exec("ALTER TABLE route_segments RENAME COLUMN points TO points_legacy_json").Error; err != nil {
					return err
				}
				if err := db.Exec("ALTER TABLE route_segments ADD COLUMN IF NOT EXISTS points geometry(LineString, 4326)").Error; err != nil {
					return err
				}
			}
		} else {
			if err := db.Exec("ALTER TABLE route_segments ADD COLUMN IF NOT EXISTS points geometry(LineString, 4326)").Error; err != nil {
				return err
			}
		}
	}

	return nil
}

type legacyRouteSegmentRow struct {
	ID               uint64 `gorm:"column:id"`
	Filename         string `gorm:"column:filename"`
	Content          []byte `gorm:"column:content"`
	PointsLegacyJSON string `gorm:"column:points_legacy_json"`
}

type legacyPointRecord struct {
	Point struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"point"`
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

func postRouteSegmentsLineStringMigrate(db *gorm.DB) error { //nolint:gocyclo
	if !db.Migrator().HasTable("route_segments") {
		return nil
	}

	hasLegacyColumn := db.Migrator().HasColumn("route_segments", "points_legacy_json")

	var rows []legacyRouteSegmentRow
	if hasLegacyColumn {
		if err := db.Raw("SELECT id, filename, content, points_legacy_json FROM route_segments WHERE points IS NULL").Scan(&rows).Error; err != nil {
			return err
		}
	} else {
		if err := db.Raw("SELECT id, filename, content, '' as points_legacy_json FROM route_segments WHERE points IS NULL").Scan(&rows).Error; err != nil {
			return err
		}
	}

	for _, row := range rows {
		var points []gogis.Point

		// 1. Try parsing from content (GPX) if available and WorkoutParser is registered
		if len(row.Content) > 0 && model.WorkoutParser != nil {
			if parsed, err := model.WorkoutParser(row.Filename, row.Content); err == nil && len(parsed) > 0 && parsed[0] != nil {
				for _, r := range parsed[0].Records {
					if r.Point != nil && (r.Point.Lat != 0 || r.Point.Lng != 0) {
						points = append(points, *r.Point)
					}
				}
			}
		}

		// 2. Fallback: Parse from legacy JSON text
		if len(points) == 0 && strings.TrimSpace(row.PointsLegacyJSON) != "" {
			var legacyPoints []legacyPointRecord
			if err := json.Unmarshal([]byte(row.PointsLegacyJSON), &legacyPoints); err == nil {
				for _, lp := range legacyPoints {
					lat := lp.Point.Lat
					lng := lp.Point.Lng
					if lat == 0 && lng == 0 {
						lat = lp.Lat
						lng = lp.Lng
					}
					if lat != 0 || lng != 0 {
						points = append(points, gogis.Point{Lat: lat, Lng: lng})
					}
				}
			}
		}

		if len(points) >= 2 {
			ls := gogis.LineString{Points: points}
			if err := db.Exec("UPDATE route_segments SET points = ST_GeomFromEWKT(?) WHERE id = ?", ls.String(), row.ID).Error; err != nil {
				return err
			}
		}
	}

	if hasLegacyColumn {
		if err := db.Exec("ALTER TABLE route_segments DROP COLUMN IF EXISTS points_legacy_json").Error; err != nil {
			return err
		}
	}

	_ = db.Exec("CREATE INDEX IF NOT EXISTS idx_route_segments_points ON route_segments USING GIST (points)").Error

	return nil
}
