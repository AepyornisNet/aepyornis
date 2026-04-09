# Migrations

This package registers database migrations through `init()` functions.

## Create a new migration

1. Create a new file in this folder:
   - `migration_<version>_<short_description>.go`
2. Use a monotonically increasing numeric version (recommended format: `YYYYMMDDNN`).
3. Register the migration in `init()` with `model.RegisterMigration(...)`.
4. Keep the migration idempotent and safe to run exactly once.

## Migration file template

```go
package migrations

import (
	"github.com/jovandeginste/workout-tracker/v2/pkg/model"
	"gorm.io/gorm"
)

func init() {
	model.RegisterMigration(
		2026040901,
		"describe what this migration does",
		preAutoMigrate,
		postAutoMigrate,
		preAutoRollback,
		postAutoRollback,
	)
}

func preAutoMigrate(db *gorm.DB) error {
	// Optional: data cleanup/preparation before AutoMigrate runs.
	return nil
}

func postAutoMigrate(db *gorm.DB) error {
	// Optional: backfill/transforms after AutoMigrate runs.
	return nil
}

func preAutoRollback(db *gorm.DB) error {
	// Optional rollback for preAutoMigrate phase.
	return nil
}

func postAutoRollback(db *gorm.DB) error {
	// Optional rollback for postAutoMigrate phase.
	return nil
}
```

## Notes

- Any `nil` function passed to `RegisterMigration` is treated as a no-op.
- Migrations run in version order and are recorded in `schema_migrations`.
- Keep migrations focused and small; one migration should solve one schema/data change.
