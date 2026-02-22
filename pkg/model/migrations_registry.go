package model

import (
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"gorm.io/gorm"
)

type MigrationFn func(*gorm.DB) error

type migration struct {
	Version          uint64
	Description      string
	PreAutoMigrate   MigrationFn
	PostAutoMigrate  MigrationFn
	PreAutoRollback  MigrationFn
	PostAutoRollback MigrationFn
}

type migrationRegistry struct {
	migrations []migration
}

type appliedMigration struct {
	Version     uint64    `gorm:"primaryKey;autoIncrement:false"`
	Description string    `gorm:"not null"`
	AppliedAt   time.Time `gorm:"not null"`
}

func (appliedMigration) TableName() string {
	return "schema_migrations"
}

var migrations = &migrationRegistry{}

func RegisterMigration(version uint64, description string, preAutoMigrate, postAutoMigrate, preAutoRollback, postAutoRollback MigrationFn) {
	migrations.Register(version, description, preAutoMigrate, postAutoMigrate, preAutoRollback, postAutoRollback)
}

func RunMigrations(db *gorm.DB, autoMigrateFn func(*gorm.DB) error) error {
	return migrations.Run(db, autoMigrateFn)
}

func (mr *migrationRegistry) Register(version uint64, description string, preAutoMigrate, postAutoMigrate, preAutoRollback, postAutoRollback MigrationFn) {
	if description == "" {
		panic("migration description cannot be empty")
	}

	for _, existing := range mr.migrations {
		if existing.Version == version {
			panic(fmt.Sprintf("duplicate migration version %d", version))
		}
	}

	mr.migrations = append(mr.migrations, migration{
		Version:          version,
		Description:      description,
		PreAutoMigrate:   toNoop(preAutoMigrate),
		PostAutoMigrate:  toNoop(postAutoMigrate),
		PreAutoRollback:  toNoop(preAutoRollback),
		PostAutoRollback: toNoop(postAutoRollback),
	})
}

func (mr *migrationRegistry) Run(db *gorm.DB, autoMigrateFn func(*gorm.DB) error) error {
	if err := db.AutoMigrate(&appliedMigration{}); err != nil {
		return err
	}

	pending, err := mr.pendingMigrations(db)
	if err != nil {
		return err
	}

	runPre := make([]migration, 0, len(pending))
	for _, m := range pending {
		if err := m.PreAutoMigrate(db); err != nil {
			return errors.Join(err, mr.rollbackPre(runPre, db))
		}

		runPre = append(runPre, m)
	}

	if err := autoMigrateFn(db); err != nil {
		return errors.Join(err, mr.rollbackPre(runPre, db))
	}

	runPost := make([]migration, 0, len(pending))
	for _, m := range pending {
		if err := m.PostAutoMigrate(db); err != nil {
			runPost = append(runPost, m)
			return errors.Join(err, mr.rollbackPost(runPost, db))
		}

		applied := appliedMigration{
			Version:     m.Version,
			Description: m.Description,
			AppliedAt:   time.Now(),
		}

		if err := db.Create(&applied).Error; err != nil {
			runPost = append(runPost, m)
			return errors.Join(err, mr.rollbackPost(runPost, db))
		}

		slog.Info("Applied database migration", "version", m.Version, "description", m.Description, "applied_at", applied.AppliedAt)

		runPost = append(runPost, m)
	}

	return nil
}

func (mr *migrationRegistry) pendingMigrations(db *gorm.DB) ([]migration, error) {
	versions := make(map[uint64]struct{})
	var applied []appliedMigration
	if err := db.Find(&applied).Error; err != nil {
		return nil, err
	}

	for _, a := range applied {
		versions[a.Version] = struct{}{}
	}

	all := mr.sortedMigrations()
	pending := make([]migration, 0, len(all))
	for _, m := range all {
		if _, exists := versions[m.Version]; exists {
			continue
		}

		pending = append(pending, m)
	}

	return pending, nil
}

func (mr *migrationRegistry) sortedMigrations() []migration {
	sorted := make([]migration, len(mr.migrations))
	copy(sorted, mr.migrations)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Version < sorted[j].Version
	})

	return sorted
}

func (mr *migrationRegistry) rollbackPre(run []migration, db *gorm.DB) error {
	var rollbackErr error
	for i := len(run) - 1; i >= 0; i-- {
		rollbackErr = errors.Join(rollbackErr, run[i].PreAutoRollback(db))
	}

	return rollbackErr
}

func (mr *migrationRegistry) rollbackPost(run []migration, db *gorm.DB) error {
	var rollbackErr error
	for i := len(run) - 1; i >= 0; i-- {
		rollbackErr = errors.Join(rollbackErr, run[i].PostAutoRollback(db))
	}

	return rollbackErr
}

func toNoop(fn MigrationFn) MigrationFn {
	if fn == nil {
		return func(*gorm.DB) error { return nil }
	}

	return fn
}
