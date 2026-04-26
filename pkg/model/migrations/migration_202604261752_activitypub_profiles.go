package migrations

import (
	"strings"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"gorm.io/gorm"
)

func init() {
	model.RegisterMigration(
		202604261752,
		"backfill activitypub profile references and drop legacy actor columns",
		nil,
		backfillActivityPubProfileReferences,
		nil,
		nil,
	)
}

func backfillActivityPubProfileReferences(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		resolveLocalProfileID := func(userID *uint64) (uint64, error) {
			if userID == nil || *userID == 0 {
				return 0, nil
			}

			profile := &model.Profile{}
			if err := tx.Where("user_id = ?", *userID).First(profile).Error; err != nil {
				return 0, err
			}

			return profile.ID, nil
		}

		resolveRemoteProfileID := func(actorIRI, actorName *string) (uint64, error) {
			if actorIRI == nil || strings.TrimSpace(*actorIRI) == "" {
				return 0, nil
			}

			actorURL := strings.TrimSpace(*actorIRI)
			displayName := ""
			if actorName != nil {
				displayName = strings.TrimSpace(*actorName)
			}

			profile := &model.Profile{
				DisplayName: displayName,
				URL:         &actorURL,
			}
			saved, err := profile.UpsertRemote(tx)
			if err != nil {
				return 0, err
			}

			return saved.ID, nil
		}

		if tx.Migrator().HasColumn("ap_outbox_workout", "user_id") {
			type workoutRow struct {
				ID        uint64  `gorm:"column:id"`
				ProfileID *uint64 `gorm:"column:profile_id"`
				WorkoutID uint64  `gorm:"column:workout_id"`
			}

			rows := make([]workoutRow, 0)
			if err := tx.Table("ap_outbox_workout").Select("id, profile_id, workout_id").Find(&rows).Error; err != nil {
				return err
			}

			for _, row := range rows {
				if row.ProfileID != nil && *row.ProfileID != 0 {
					continue
				}

				var workoutProfileID uint64
				if err := tx.Table("workouts").Select("profile_id").Where("id = ?", row.WorkoutID).Take(&workoutProfileID).Error; err != nil {
					return err
				}

				if err := tx.Table("ap_outbox_workout").Where("id = ?", row.ID).Update("profile_id", workoutProfileID).Error; err != nil {
					return err
				}
			}
		}

		type statusRow struct {
			ID                uint64  `gorm:"column:id"`
			ProfileID         *uint64 `gorm:"column:profile_id"`
			APStatusWorkoutID *uint64 `gorm:"column:ap_status_workout_id"`
			UserID            *uint64 `gorm:"column:user_id"`
			ActorIRI          *string `gorm:"column:actor_iri"`
			ActorName         *string `gorm:"column:actor_name"`
		}

		statusSelect := []string{"id", "profile_id", "ap_status_workout_id"}
		if tx.Migrator().HasColumn("ap_statuses", "user_id") {
			statusSelect = append(statusSelect, "user_id")
		}
		if tx.Migrator().HasColumn("ap_statuses", "actor_iri") {
			statusSelect = append(statusSelect, "actor_iri")
		}
		if tx.Migrator().HasColumn("ap_statuses", "actor_name") {
			statusSelect = append(statusSelect, "actor_name")
		}

		statusRows := make([]statusRow, 0)
		if err := tx.Table("ap_statuses").Select(strings.Join(statusSelect, ", ")).Find(&statusRows).Error; err != nil {
			return err
		}

		for _, row := range statusRows {
			if row.ProfileID != nil && *row.ProfileID != 0 {
				continue
			}

			var profileID uint64
			if row.APStatusWorkoutID != nil && *row.APStatusWorkoutID != 0 {
				if err := tx.Table("ap_outbox_workout").Select("profile_id").Where("id = ?", *row.APStatusWorkoutID).Take(&profileID).Error; err != nil && err != gorm.ErrRecordNotFound {
					return err
				}
			}
			if profileID == 0 {
				var err error
				profileID, err = resolveLocalProfileID(row.UserID)
				if err != nil {
					return err
				}
			}
			if profileID == 0 {
				var err error
				profileID, err = resolveRemoteProfileID(row.ActorIRI, row.ActorName)
				if err != nil {
					return err
				}
			}
			if profileID == 0 {
				continue
			}

			if err := tx.Table("ap_statuses").Where("id = ?", row.ID).Update("profile_id", profileID).Error; err != nil {
				return err
			}
		}

		if tx.Migrator().HasColumn("ap_status_likes", "user_id") || tx.Migrator().HasColumn("ap_status_likes", "actor_iri") {
			type likeRow struct {
				ID        uint64  `gorm:"column:id"`
				ProfileID *uint64 `gorm:"column:profile_id"`
				UserID    *uint64 `gorm:"column:user_id"`
				ActorIRI  *string `gorm:"column:actor_iri"`
			}

			likeSelect := []string{"id", "profile_id"}
			if tx.Migrator().HasColumn("ap_status_likes", "user_id") {
				likeSelect = append(likeSelect, "user_id")
			}
			if tx.Migrator().HasColumn("ap_status_likes", "actor_iri") {
				likeSelect = append(likeSelect, "actor_iri")
			}

			likeRows := make([]likeRow, 0)
			if err := tx.Table("ap_status_likes").Select(strings.Join(likeSelect, ", ")).Find(&likeRows).Error; err != nil {
				return err
			}

			for _, row := range likeRows {
				if row.ProfileID != nil && *row.ProfileID != 0 {
					continue
				}

				profileID, err := resolveLocalProfileID(row.UserID)
				if err != nil {
					return err
				}
				if profileID == 0 {
					profileID, err = resolveRemoteProfileID(row.ActorIRI, nil)
					if err != nil {
						return err
					}
				}
				if profileID == 0 {
					continue
				}

				if err := tx.Table("ap_status_likes").Where("id = ?", row.ID).Update("profile_id", profileID).Error; err != nil {
					return err
				}
			}
		}

		if tx.Migrator().HasColumn("ap_outbox_delivery", "actor_iri") {
			type deliveryRow struct {
				ID        uint64  `gorm:"column:id"`
				ProfileID *uint64 `gorm:"column:profile_id"`
				ActorIRI  *string `gorm:"column:actor_iri"`
			}

			deliveryRows := make([]deliveryRow, 0)
			if err := tx.Table("ap_outbox_delivery").Select("id, profile_id, actor_iri").Find(&deliveryRows).Error; err != nil {
				return err
			}

			for _, row := range deliveryRows {
				if row.ProfileID != nil && *row.ProfileID != 0 {
					continue
				}

				profileID, err := resolveRemoteProfileID(row.ActorIRI, nil)
				if err != nil {
					return err
				}
				if profileID == 0 {
					continue
				}

				if err := tx.Table("ap_outbox_delivery").Where("id = ?", row.ID).Update("profile_id", profileID).Error; err != nil {
					return err
				}
			}
		}

		if tx.Migrator().HasIndex("ap_statuses", "idx_ap_statuses_user_published") {
			if err := tx.Migrator().DropIndex("ap_statuses", "idx_ap_statuses_user_published"); err != nil {
				return err
			}
		}
		if tx.Migrator().HasIndex("ap_status_likes", "idx_ap_status_like_status_user") {
			if err := tx.Migrator().DropIndex("ap_status_likes", "idx_ap_status_like_status_user"); err != nil {
				return err
			}
		}
		if tx.Migrator().HasIndex("ap_status_likes", "idx_ap_status_like_status_actor") {
			if err := tx.Migrator().DropIndex("ap_status_likes", "idx_ap_status_like_status_actor"); err != nil {
				return err
			}
		}
		if tx.Migrator().HasIndex("ap_outbox_workout", "idx_ap_outbox_workout_user_workout") {
			if err := tx.Migrator().DropIndex("ap_outbox_workout", "idx_ap_outbox_workout_user_workout"); err != nil {
				return err
			}
		}
		if tx.Migrator().HasIndex("ap_outbox_delivery", "idx_ap_outbox_delivery_entry_actor") {
			if err := tx.Migrator().DropIndex("ap_outbox_delivery", "idx_ap_outbox_delivery_entry_actor"); err != nil {
				return err
			}
		}

		if tx.Migrator().HasColumn("ap_statuses", "user_id") {
			if err := tx.Migrator().DropColumn("ap_statuses", "user_id"); err != nil {
				return err
			}
		}
		if tx.Migrator().HasColumn("ap_statuses", "actor_iri") {
			if err := tx.Migrator().DropColumn("ap_statuses", "actor_iri"); err != nil {
				return err
			}
		}
		if tx.Migrator().HasColumn("ap_statuses", "actor_name") {
			if err := tx.Migrator().DropColumn("ap_statuses", "actor_name"); err != nil {
				return err
			}
		}
		if tx.Migrator().HasColumn("ap_statuses", "inbox_url") {
			if err := tx.Migrator().DropColumn("ap_statuses", "inbox_url"); err != nil {
				return err
			}
		}
		if tx.Migrator().HasColumn("ap_status_likes", "user_id") {
			if err := tx.Migrator().DropColumn("ap_status_likes", "user_id"); err != nil {
				return err
			}
		}
		if tx.Migrator().HasColumn("ap_status_likes", "actor_iri") {
			if err := tx.Migrator().DropColumn("ap_status_likes", "actor_iri"); err != nil {
				return err
			}
		}
		if tx.Migrator().HasColumn("ap_outbox_workout", "user_id") {
			if err := tx.Migrator().DropColumn("ap_outbox_workout", "user_id"); err != nil {
				return err
			}
		}
		if tx.Migrator().HasColumn("ap_outbox_delivery", "actor_iri") {
			if err := tx.Migrator().DropColumn("ap_outbox_delivery", "actor_iri"); err != nil {
				return err
			}
		}

		if !tx.Migrator().HasIndex("ap_statuses", "idx_ap_statuses_profile_published") {
			if err := tx.Migrator().CreateIndex("ap_statuses", "idx_ap_statuses_profile_published"); err != nil {
				return err
			}
		}
		if !tx.Migrator().HasIndex("ap_status_likes", "idx_ap_status_like_status_profile") {
			if err := tx.Migrator().CreateIndex("ap_status_likes", "idx_ap_status_like_status_profile"); err != nil {
				return err
			}
		}
		if !tx.Migrator().HasIndex("ap_outbox_workout", "idx_ap_outbox_workout_profile_workout") {
			if err := tx.Migrator().CreateIndex("ap_outbox_workout", "idx_ap_outbox_workout_profile_workout"); err != nil {
				return err
			}
		}
		if !tx.Migrator().HasIndex("ap_outbox_delivery", "idx_ap_outbox_delivery_entry_profile") {
			if err := tx.Migrator().CreateIndex("ap_outbox_delivery", "idx_ap_outbox_delivery_entry_profile"); err != nil {
				return err
			}
		}

		return nil
	})
}
