package migrations

import (
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"gorm.io/gorm"
)

func init() {
	model.RegisterMigration(
		202610041200,
		"remove speed interval records for virtual and indoor workouts",
		nil,
		postRemoveVirtualWorkoutSpeedRecordsMigrate,
		nil,
		nil,
	)
}

func postRemoveVirtualWorkoutSpeedRecordsMigrate(db *gorm.DB) error {
	if !db.Migrator().HasTable("workout_interval_records") || !db.Migrator().HasTable("workouts") {
		return nil
	}

	excludedWorkoutTypes := model.ExcludedRecordWorkoutTypes()
	excludedSubTypes := model.ExcludedRecordSubTypes()

	condition := db.Where("1 = 0")
	if len(excludedWorkoutTypes) > 0 {
		condition = condition.Or("workouts.type IN (?)", excludedWorkoutTypes)
	}
	if len(excludedSubTypes) > 0 {
		condition = condition.Or("coalesce(workouts.sub_type, '') IN (?)", excludedSubTypes)
	}
	condition = condition.Or("lower(coalesce(workouts.sub_type, '')) LIKE '%virtual%'")

	subQuery := db.Table("workouts").Select("workouts.id").Where(condition)

	return db.Where("type = ? AND workout_id IN (?)", model.WorkoutIntervalBestTypeSpeed, subQuery).
		Delete(&model.WorkoutIntervalBest{}).Error
}
