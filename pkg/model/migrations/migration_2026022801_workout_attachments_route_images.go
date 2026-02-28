package migrations

import (
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"gorm.io/gorm"
)

func init() {
	model.RegisterMigration(2026022801, "create workout attachments and backfill route images",
		func(*gorm.DB) error {
			return nil
		},
		func(db *gorm.DB) error {
			return backfillWorkoutRouteImageAttachments(db)
		},
		func(*gorm.DB) error {
			return nil
		},
		func(*gorm.DB) error {
			return nil
		},
	)
}

func backfillWorkoutRouteImageAttachments(db *gorm.DB) error {
	const batchSize = 100

	for {
		ids := make([]uint64, 0, batchSize)
		err := db.Table("workouts").
			Select("workouts.id").
			Joins("JOIN map_data ON map_data.workout_id = workouts.id").
			Joins("JOIN map_data_details ON map_data_details.map_data_id = map_data.id").
			Joins("JOIN map_data_details_points ON map_data_details_points.map_data_details_id = map_data_details.id").
			Joins("LEFT JOIN workout_attachments ON workout_attachments.workout_id = workouts.id AND workout_attachments.kind = ?", model.WorkoutAttachmentKindRouteImage).
			Where("workout_attachments.id IS NULL").
			Group("workouts.id").
			Having("COUNT(map_data_details_points.sort_order) > 2").
			Order("workouts.id ASC").
			Limit(batchSize).
			Find(&ids).Error
		if err != nil {
			return err
		}

		if len(ids) == 0 {
			break
		}

		for _, workoutID := range ids {
			workout, getErr := model.GetWorkoutDetails(db, workoutID)
			if getErr != nil {
				return getErr
			}

			routeImageContent, generateErr := model.GenerateWorkoutAttachmentImage(workout)
			if generateErr != nil {
				return generateErr
			}

			if _, upsertErr := model.UpsertRouteImageAttachment(
				db,
				workout.ID,
				model.WorkoutRouteImageFilename(workout),
				model.RouteImageMIMEType,
				routeImageContent,
			); upsertErr != nil {
				return upsertErr
			}
		}
	}

	return nil
}
