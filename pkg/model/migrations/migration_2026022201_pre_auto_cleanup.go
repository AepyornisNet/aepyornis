package migrations

import (
	"math"

	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"gorm.io/gorm"
)

func init() {
	model.RegisterMigration(2026022201, "normalize legacy workout and map data before automigrate",
		func(db *gorm.DB) error {
			if !db.Migrator().HasTable(&model.TrackData{}) {
				return nil
			}

			for _, column := range []string{"average_speed", "average_speed_no_pause"} {
				if !db.Migrator().HasColumn(&model.TrackData{}, column) {
					continue
				}

				if err := db.
					Model(&model.TrackData{}).
					Where(column+" in ?", []float64{math.Inf(1), math.Inf(-1), math.NaN()}).
					Update(column, 0).Error; err != nil {
					return err
				}
			}

			if err := db.
				Where("id < (select max(id) from map_data as m where m.workout_id = map_data.workout_id)").
				Delete(&model.TrackData{}).Error; err != nil {
				return err
			}

			if err := db.
				Where("id < (select max(id) from workouts as w where w.date = workouts.date and w.user_id = workouts.user_id)").
				Delete(&model.Workout{}).Error; err != nil {
				return err
			}

			if db.Migrator().HasTable("map_data_details") {
				if err := db.Exec("DELETE FROM map_data_details WHERE map_data_id IN (SELECT map_data_id FROM map_data_details as mdd where map_data_details.created_at < mdd.created_at)").Error; err != nil {
					return err
				}
			}

			return db.
				Model(&model.Workout{}).
				Where(&model.Workout{Type: "weight lifting"}).
				Update("type", model.WorkoutTypeWeightLifting).Error
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
