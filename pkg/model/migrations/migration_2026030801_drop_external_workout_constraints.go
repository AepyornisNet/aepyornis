package migrations

import (
	"fmt"
	"log/slog"
	"regexp"

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
		// SQLite doesn't enforce NOT NULL changes via ALTER TABLE and
		// typically has no FK enforcement enabled. The updated GORM tags
		// take effect on new columns via AutoMigrate. No action needed.
		return nil
	case "postgres":
		stmts := []string{
			"ALTER TABLE workout_attachments ALTER COLUMN content DROP NOT NULL",
			"ALTER TABLE workout_attachments ALTER COLUMN checksum DROP NOT NULL",
			"ALTER TABLE workouts ALTER COLUMN user_id DROP NOT NULL",
		}
		for _, stmt := range stmts {
			if err := db.Exec(stmt).Error; err != nil {
				return err
			}
		}

		// Drop the foreign key constraint on workouts.user_id so that
		// external workouts (user_id = 0) can be inserted.
		dropPostgresFKOnWorkoutsUserID(db)
	case "mysql":
		stmts := []string{
			"ALTER TABLE workout_attachments MODIFY content LONGBLOB NULL",
			"ALTER TABLE workout_attachments MODIFY checksum VARBINARY(255) NULL",
			"ALTER TABLE workouts MODIFY user_id BIGINT UNSIGNED NULL",
		}
		for _, stmt := range stmts {
			if err := db.Exec(stmt).Error; err != nil {
				return err
			}
		}

		dropMySQLFKOnWorkoutsUserID(db)
	}

	return nil
}

// dropPostgresFKOnWorkoutsUserID finds and drops any FK constraint on
// workouts.user_id for PostgreSQL.
func dropPostgresFKOnWorkoutsUserID(db *gorm.DB) {
	var constraintName string
	err := db.Raw(`
		SELECT constraint_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON tc.constraint_name = kcu.constraint_name
		 AND tc.table_schema = kcu.table_schema
		WHERE tc.table_name = 'workouts'
		  AND kcu.column_name = 'user_id'
		  AND tc.constraint_type = 'FOREIGN KEY'
		LIMIT 1
	`).Scan(&constraintName).Error
	if err != nil || constraintName == "" {
		return
	}

	if !isValidSQLIdentifier(constraintName) {
		slog.Warn("Skipping FK drop: invalid constraint name", "constraint", constraintName)
		return
	}

	//nolint:gosec // constraintName is validated by isValidSQLIdentifier above
	stmt := fmt.Sprintf(`ALTER TABLE workouts DROP CONSTRAINT "%s"`, constraintName)
	if err := db.Exec(stmt).Error; err != nil {
		slog.Warn("Failed to drop FK constraint on workouts.user_id", "constraint", constraintName, "error", err)
	}
}

// dropMySQLFKOnWorkoutsUserID finds and drops any FK constraint on
// workouts.user_id for MySQL.
func dropMySQLFKOnWorkoutsUserID(db *gorm.DB) {
	var constraintName string
	err := db.Raw(`
		SELECT CONSTRAINT_NAME
		FROM information_schema.KEY_COLUMN_USAGE
		WHERE TABLE_NAME = 'workouts'
		  AND COLUMN_NAME = 'user_id'
		  AND REFERENCED_TABLE_NAME IS NOT NULL
		LIMIT 1
	`).Scan(&constraintName).Error
	if err != nil || constraintName == "" {
		return
	}

	if !isValidSQLIdentifier(constraintName) {
		slog.Warn("Skipping FK drop: invalid constraint name", "constraint", constraintName)
		return
	}

	//nolint:gosec // constraintName is validated by isValidSQLIdentifier above
	stmt := fmt.Sprintf("ALTER TABLE workouts DROP FOREIGN KEY `%s`", constraintName)
	if err := db.Exec(stmt).Error; err != nil {
		slog.Warn("Failed to drop FK constraint on workouts.user_id", "constraint", constraintName, "error", err)
	}
}

// validSQLIdentifier matches standard SQL identifier characters only.
var validSQLIdentifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// isValidSQLIdentifier ensures the name only contains safe identifier characters.
func isValidSQLIdentifier(name string) bool {
	return validSQLIdentifier.MatchString(name)
}
