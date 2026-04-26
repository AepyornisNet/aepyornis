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
		resolver := activityPubProfileResolver{tx: tx}
		steps := []func(*gorm.DB, activityPubProfileResolver) error{
			backfillOutboxWorkoutProfiles,
			backfillStatusProfiles,
			backfillLikeProfiles,
			backfillDeliveryProfiles,
			dropLegacyActivityPubIndexes,
			dropLegacyActivityPubColumns,
			ensureActivityPubProfileIndexes,
		}

		for _, step := range steps {
			if err := step(tx, resolver); err != nil {
				return err
			}
		}

		return nil
	})
}

type activityPubProfileResolver struct {
	tx *gorm.DB
}

type statusProfileRow struct {
	ID                uint64  `gorm:"column:id"`
	ProfileID         *uint64 `gorm:"column:profile_id"`
	APStatusWorkoutID *uint64 `gorm:"column:ap_status_workout_id"`
	UserID            *uint64 `gorm:"column:user_id"`
	ActorIRI          *string `gorm:"column:actor_iri"`
	ActorName         *string `gorm:"column:actor_name"`
}

func (r activityPubProfileResolver) localProfileID(userID *uint64) (uint64, error) {
	if userID == nil || *userID == 0 {
		return 0, nil
	}

	profile := &model.Profile{}
	if err := r.tx.Where("user_id = ?", *userID).First(profile).Error; err != nil {
		return 0, err
	}

	return profile.ID, nil
}

func (r activityPubProfileResolver) remoteProfileID(actorIRI, actorName *string) (uint64, error) {
	if actorIRI == nil || strings.TrimSpace(*actorIRI) == "" {
		return 0, nil
	}

	actorURL := strings.TrimSpace(*actorIRI)
	displayName := ""
	if actorName != nil {
		displayName = strings.TrimSpace(*actorName)
	}

	profile := &model.Profile{DisplayName: displayName, URL: &actorURL}
	saved, err := profile.UpsertRemote(r.tx)
	if err != nil {
		return 0, err
	}

	return saved.ID, nil
}

func backfillOutboxWorkoutProfiles(tx *gorm.DB, _ activityPubProfileResolver) error {
	if !tx.Migrator().HasColumn("ap_outbox_workout", "user_id") {
		return nil
	}

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

	return nil
}

func backfillStatusProfiles(tx *gorm.DB, resolver activityPubProfileResolver) error {
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

	statusRows := make([]statusProfileRow, 0)
	if err := tx.Table("ap_statuses").Select(strings.Join(statusSelect, ", ")).Find(&statusRows).Error; err != nil {
		return err
	}

	for _, row := range statusRows {
		if row.ProfileID != nil && *row.ProfileID != 0 {
			continue
		}

		profileID, err := resolveStatusProfileID(tx, resolver, row)
		if err != nil {
			return err
		}
		if profileID == 0 {
			continue
		}
		if err := tx.Table("ap_statuses").Where("id = ?", row.ID).Update("profile_id", profileID).Error; err != nil {
			return err
		}
	}

	return nil
}

func resolveStatusProfileID(tx *gorm.DB, resolver activityPubProfileResolver, row statusProfileRow) (uint64, error) {
	if row.APStatusWorkoutID != nil && *row.APStatusWorkoutID != 0 {
		var profileID uint64
		if err := tx.Table("ap_outbox_workout").Select("profile_id").Where("id = ?", *row.APStatusWorkoutID).Take(&profileID).Error; err != nil && err != gorm.ErrRecordNotFound {
			return 0, err
		}
		if profileID != 0 {
			return profileID, nil
		}
	}

	profileID, err := resolver.localProfileID(row.UserID)
	if err != nil || profileID != 0 {
		return profileID, err
	}

	return resolver.remoteProfileID(row.ActorIRI, row.ActorName)
}

func backfillLikeProfiles(tx *gorm.DB, resolver activityPubProfileResolver) error {
	if !tx.Migrator().HasColumn("ap_status_likes", "user_id") && !tx.Migrator().HasColumn("ap_status_likes", "actor_iri") {
		return nil
	}

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

		profileID, err := resolver.localProfileID(row.UserID)
		if err != nil {
			return err
		}
		if profileID == 0 {
			profileID, err = resolver.remoteProfileID(row.ActorIRI, nil)
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

	return nil
}

func backfillDeliveryProfiles(tx *gorm.DB, resolver activityPubProfileResolver) error {
	if !tx.Migrator().HasColumn("ap_outbox_delivery", "actor_iri") {
		return nil
	}

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

		profileID, err := resolver.remoteProfileID(row.ActorIRI, nil)
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

	return nil
}

func dropLegacyActivityPubIndexes(tx *gorm.DB, _ activityPubProfileResolver) error {
	indexes := []struct {
		table string
		name  string
	}{
		{"ap_statuses", "idx_ap_statuses_user_published"},
		{"ap_status_likes", "idx_ap_status_like_status_user"},
		{"ap_status_likes", "idx_ap_status_like_status_actor"},
		{"ap_outbox_workout", "idx_ap_outbox_workout_user_workout"},
		{"ap_outbox_delivery", "idx_ap_outbox_delivery_entry_actor"},
	}

	for _, index := range indexes {
		if tx.Migrator().HasIndex(index.table, index.name) {
			if err := tx.Migrator().DropIndex(index.table, index.name); err != nil {
				return err
			}
		}
	}

	return nil
}

func dropLegacyActivityPubColumns(tx *gorm.DB, _ activityPubProfileResolver) error {
	columns := []struct {
		table  string
		column string
	}{
		{"ap_statuses", "user_id"},
		{"ap_statuses", "actor_iri"},
		{"ap_statuses", "actor_name"},
		{"ap_statuses", "inbox_url"},
		{"ap_status_likes", "user_id"},
		{"ap_status_likes", "actor_iri"},
		{"ap_outbox_workout", "user_id"},
		{"ap_outbox_delivery", "actor_iri"},
	}

	for _, column := range columns {
		if tx.Migrator().HasColumn(column.table, column.column) {
			if err := tx.Migrator().DropColumn(column.table, column.column); err != nil {
				return err
			}
		}
	}

	return nil
}

func ensureActivityPubProfileIndexes(tx *gorm.DB, _ activityPubProfileResolver) error {
	indexes := []struct {
		table string
		name  string
	}{
		{"ap_statuses", "idx_ap_statuses_profile_published"},
		{"ap_status_likes", "idx_ap_status_like_status_profile"},
		{"ap_outbox_workout", "idx_ap_outbox_workout_profile_workout"},
		{"ap_outbox_delivery", "idx_ap_outbox_delivery_entry_profile"},
	}

	for _, index := range indexes {
		if !tx.Migrator().HasIndex(index.table, index.name) {
			if err := tx.Migrator().CreateIndex(index.table, index.name); err != nil {
				return err
			}
		}
	}

	return nil
}
