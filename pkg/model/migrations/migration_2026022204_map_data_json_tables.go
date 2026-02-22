package migrations

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"gorm.io/gorm"
)

func init() {
	model.RegisterMigration(2026022204, "migrate map_data climbs and points JSON into relational tables",
		func(*gorm.DB) error {
			return nil
		},
		func(db *gorm.DB) error {
			if db.Migrator().HasColumn("map_data", "climbs") {
				if err := migrateMapDataClimbs(db); err != nil {
					return err
				}
			}

			if db.Migrator().HasColumn("map_data_details", "points") {
				if err := migrateMapDataDetailPoints(db); err != nil {
					return err
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

func migrateMapDataClimbs(db *gorm.DB) error {
	if !db.Migrator().HasTable(&model.Segment{}) {
		return nil
	}

	if err := db.Where("1 = 1").Delete(&model.Segment{}).Error; err != nil {
		return err
	}

	rows, err := db.Table("map_data").Select("id, climbs").Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	batch := make([]model.Segment, 0, 500)

	for rows.Next() {
		var (
			id  uint64
			raw any
		)

		if err := rows.Scan(&id, &raw); err != nil {
			return err
		}

		payload, ok := payloadBytes(raw)
		if !ok {
			continue
		}

		var climbs []model.Segment
		if err := json.Unmarshal(payload, &climbs); err != nil {
			return fmt.Errorf("unmarshal map_data.climbs for map_data_id=%d: %w", id, err)
		}

		for i := range climbs {
			climbs[i].MapDataID = id
			climbs[i].SortOrder = i

			batch = append(batch, climbs[i])

			if len(batch) == cap(batch) {
				if err := db.Create(&batch).Error; err != nil {
					return err
				}
				batch = batch[:0]
			}
		}
	}

	if len(batch) > 0 {
		if err := db.Create(&batch).Error; err != nil {
			return err
		}
	}

	return rows.Err()
}

func migrateMapDataDetailPoints(db *gorm.DB) error {
	if !db.Migrator().HasTable(&model.MapPoint{}) {
		return nil
	}

	if err := db.Where("1 = 1").Delete(&model.MapPoint{}).Error; err != nil {
		return err
	}

	rows, err := db.Table("map_data_details").Select("id, points").Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	batch := make([]model.MapPoint, 0, 500)

	for rows.Next() {
		var (
			id  uint64
			raw any
		)

		if err := rows.Scan(&id, &raw); err != nil {
			return err
		}

		payload, ok := payloadBytes(raw)
		if !ok {
			continue
		}

		var points []model.MapPoint
		if err := json.Unmarshal(payload, &points); err != nil {
			return fmt.Errorf("unmarshal map_data_details.points for map_data_details_id=%d: %w", id, err)
		}

		for i := range points {
			points[i].MapDataDetailsID = id
			points[i].SortOrder = i

			batch = append(batch, points[i])

			if len(batch) == cap(batch) {
				if err := db.Create(&batch).Error; err != nil {
					return err
				}
				batch = batch[:0]
			}
		}
	}

	if len(batch) > 0 {
		if err := db.Create(&batch).Error; err != nil {
			return err
		}
	}

	return rows.Err()
}

func payloadBytes(raw any) ([]byte, bool) {
	if raw == nil {
		return nil, false
	}

	var payload []byte

	switch v := raw.(type) {
	case []byte:
		payload = v
	case string:
		payload = []byte(v)
	default:
		payload = []byte(fmt.Sprint(v))
	}

	trimmed := strings.TrimSpace(string(payload))
	if trimmed == "" || trimmed == "null" || trimmed == "[]" {
		return nil, false
	}

	return payload, true
}
