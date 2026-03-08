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
// instead of local Content/Checksum, and external workouts have a NULL UserID.
//
// This runs as a pre-auto-migrate step so AutoMigrate can pick up the
// updated GORM tags without constraint violations.
func dropOldWorkoutConstraints(db *gorm.DB) error {
	if !db.Migrator().HasTable("workout_attachments") {
		return nil
	}

	stmts := []string{
		"ALTER TABLE workout_attachments ALTER COLUMN content DROP NOT NULL",
		"ALTER TABLE workout_attachments ALTER COLUMN checksum DROP NOT NULL",
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}

	// Drop the foreign key constraint on workouts.user_id so that
	// external workouts (user_id IS NULL) can be inserted.
	dropPostgresFKOnWorkoutsUserID(db)

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

	stmt := fmt.Sprintf(`ALTER TABLE workouts DROP CONSTRAINT "%s"`, constraintName)
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
