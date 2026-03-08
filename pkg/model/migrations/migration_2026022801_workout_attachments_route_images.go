package migrations

import (
	"errors"

	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"gorm.io/gorm"
)

func init() {
	model.RegisterMigration(2026022801, "create workout attachments and backfill route images",
		func(*gorm.DB) error {
			return nil
		},
		backfillWorkoutRouteImageAttachments,
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
			Joins("JOIN track_data ON track_data.workout_id = workouts.id").
			Joins("JOIN data_points ON data_points.track_data_id = track_data.id").
			Joins("LEFT JOIN workout_attachments ON workout_attachments.workout_id = workouts.id AND workout_attachments.kind = ?", model.WorkoutAttachmentKindRouteImage).
			Where("workout_attachments.id IS NULL").
			Group("workouts.id").
			Having("COUNT(data_points.sort_order) > 2").
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

			routeImageContent, generateErr := model.GenerateWorkoutRouteImage(workout)
			if generateErr != nil {
				if errors.Is(generateErr, model.ErrWorkoutMissingCoordinates) {
					continue
				}

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
