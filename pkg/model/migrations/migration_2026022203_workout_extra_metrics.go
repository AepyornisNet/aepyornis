package migrations

import (
	"encoding/json"
	"sort"

	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"gorm.io/gorm"
)

func init() {
	model.RegisterMigration(2026022203, "backfill workout extra metrics after automigrate",
		func(*gorm.DB) error {
			return nil
		},
		func(db *gorm.DB) error {
			// All table/column name variables below are set to one of two hardcoded
			// string literals (never user input), so no SQL injection risk exists.

			// Determine which table name to use (may have been renamed already)
			trackDataTable := "map_data"
			if db.Migrator().HasTable("track_data") {
				trackDataTable = "track_data"
			}

			dataPointsTable := "map_data_details_points"
			if db.Migrator().HasTable("data_points") {
				dataPointsTable = "data_points"
			}

			trackDataIDColumn := "map_data_id"
			if db.Migrator().HasColumn(dataPointsTable, "track_data_id") {
				trackDataIDColumn = "track_data_id"
			}

			// Find track_data rows where extra_metrics is null or empty
			rows, err := db.Raw("SELECT id FROM " + trackDataTable + " WHERE extra_metrics IS NULL OR extra_metrics = '[]' OR extra_metrics = 'null' OR extra_metrics = ''").Rows()
			if err != nil {
				return err
			}
			defer rows.Close()

			for rows.Next() {
				var trackDataID uint64
				if err := rows.Scan(&trackDataID); err != nil {
					return err
				}

				// Collect unique metric keys from data points
				pointRows, err := db.Raw("SELECT extra_metrics FROM "+dataPointsTable+" WHERE "+trackDataIDColumn+" = ?", trackDataID).Rows()
				if err != nil {
					return err
				}

				metricsSet := make(map[string]bool)
				for pointRows.Next() {
					var metricsRaw []byte
					if err := pointRows.Scan(&metricsRaw); err != nil {
						pointRows.Close()
						return err
					}
					if len(metricsRaw) == 0 {
						continue
					}
					var metrics map[string]interface{}
					if err := json.Unmarshal(metricsRaw, &metrics); err == nil {
						for k := range metrics {
							metricsSet[k] = true
						}
					}
				}
				pointRows.Close()

				if len(metricsSet) == 0 {
					continue
				}

				metricsList := make([]string, 0, len(metricsSet))
				for k := range metricsSet {
					metricsList = append(metricsList, k)
				}
				sort.Strings(metricsList)

				metricsJSON, _ := json.Marshal(metricsList)
				if err := db.Exec("UPDATE "+trackDataTable+" SET extra_metrics = ? WHERE id = ?", string(metricsJSON), trackDataID).Error; err != nil {
					return err
				}
			}

			return rows.Err()
		},
		func(*gorm.DB) error {
			return nil
		},
		func(*gorm.DB) error {
			return nil
		},
	)
}
