package model

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/fsouza/slognil"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/peterldowns/pgtestdb"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// GormMigrator implements pgtestdb.Migrator to set up template databases
// using GORM AutoMigrate and model migrations.
type GormMigrator struct{}

func (GormMigrator) Hash() (string, error) {
	return "aepyornis-postgis-schema-v2", nil
}

func (GormMigrator) Migrate(_ context.Context, _ *sql.DB, conf pgtestdb.Config) error {
	gdb, err := Connect("postgres", conf.URL(), false, slognil.NewLogger())
	if err != nil {
		return err
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// TestDBURL provisions a test database via pgtestdb and returns its connection URL.
func TestDBURL(t *testing.T) string {
	t.Helper()

	user := os.Getenv("PGUSER")
	if user == "" {
		user = "postgres"
	}
	password := os.Getenv("PGPASSWORD")
	if password == "" {
		password = "password"
	}
	host := os.Getenv("PGHOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("PGPORT")
	if port == "" {
		port = "5433"
	}
	database := os.Getenv("PGDATABASE")
	if database == "" {
		database = "postgres"
	}

	role := pgtestdb.Role{
		Username:     "pgtdbuser",
		Password:     "pgtdbpass",
		Capabilities: "SUPERUSER",
	}

	conf := pgtestdb.Config{
		DriverName:                "pgx",
		User:                      user,
		Password:                  password,
		Host:                      host,
		Port:                      port,
		Database:                  database,
		Options:                   "sslmode=disable",
		TestRole:                  &role,
		ForceTerminateConnections: true,
	}

	dbConf := pgtestdb.Custom(t, conf, GormMigrator{})
	return dbConf.URL()
}

// TestDB is a test helper that returns a fresh, isolated, fully-migrated
// GORM DB connected to a test database provisioned by pgtestdb.
func TestDB(t *testing.T) *gorm.DB {
	t.Helper()

	url := TestDBURL(t)
	db, err := Connect("postgres", url, false, slognil.NewLogger())
	require.NoError(t, err)
	return db
}
