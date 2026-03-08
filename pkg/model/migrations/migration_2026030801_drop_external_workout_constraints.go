package migrations

import (
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"gorm.io/gorm"
)

func init() {
	model.RegisterMigration(2026030801, "drop old constraints for external workout support",
		dropOldWorkoutConstraints,
		nil,
		nil,
		nil,
	)
}

// dropOldWorkoutConstraints relaxes constraints that block external
// (federated) workouts. External workout attachments use ExternalURL
// instead of local Content/Checksum, and external workouts have UserID = 0.
//
// This runs as a pre-auto-migrate step so AutoMigrate can pick up the
// updated GORM tags without constraint violations.
func dropOldWorkoutConstraints(db *gorm.DB) error {
	dialect := db.Dialector.Name()

	switch dialect {
	case "sqlite":
		// SQLite doesn't enforce NOT NULL changes via ALTER TABLE.
		// AutoMigrate handles new columns; existing NOT NULL columns
		// won't block inserts of NULL because the GORM tags have already
		// been changed. No action needed.
		return nil
	case "postgres":
		stmts := []string{
			"ALTER TABLE workout_attachments ALTER COLUMN content DROP NOT NULL",
			"ALTER TABLE workout_attachments ALTER COLUMN checksum DROP NOT NULL",
		}
		for _, stmt := range stmts {
			if err := db.Exec(stmt).Error; err != nil {
				return err
			}
		}
	case "mysql":
		// MySQL requires the full column definition for ALTER.
		// Content is LONGBLOB, checksum is VARBINARY.
		stmts := []string{
			"ALTER TABLE workout_attachments MODIFY content LONGBLOB NULL",
			"ALTER TABLE workout_attachments MODIFY checksum VARBINARY(255) NULL",
		}
		for _, stmt := range stmts {
			if err := db.Exec(stmt).Error; err != nil {
				return err
			}
		}
	}

	return nil
}
