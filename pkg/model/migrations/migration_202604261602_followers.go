package migrations

import (
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"gorm.io/gorm"
)

func init() {
	model.RegisterMigration(
		202604261602,
		"drop followers table for profile-based follower schema",
		func(db *gorm.DB) error {
			return db.Migrator().DropTable(&model.Follower{})
		},
		nil,
		nil,
		nil,
	)
}
