package model

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/glebarez/sqlite"
	slogGorm "github.com/orandin/slog-gorm"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Threshold at which point queries are logged as slow
const thresholdSlowQueries = 100 * time.Millisecond

var ErrUnsuportedDriver = errors.New("unsupported driver")

type Model struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	ID        uint64 `gorm:"primaryKey"`
}

func Connect(driver, dsn string, debug bool, logger *slog.Logger) (*gorm.DB, error) {
	loggerOptions := []slogGorm.Option{
		slogGorm.WithHandler(logger.With("module", "database").Handler()),
		slogGorm.WithSlowThreshold(thresholdSlowQueries),
	}

	if debug {
		loggerOptions = append(loggerOptions, slogGorm.WithTraceAll())
	}

	gormLogger := slogGorm.New(
		loggerOptions...,
	)

	d, err := dialectorFor(driver, dsn)
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(d, &gorm.Config{
		Logger:         gormLogger,
		TranslateError: true,
	})
	if err != nil {
		return nil, err
	}

	if err := RunMigrations(db, func(db *gorm.DB) error {
		return db.AutoMigrate(
			&User{}, &Profile{}, &Config{}, &Equipment{}, &WorkoutEquipment{}, &Measurement{},
			&Workout{}, &GPXData{}, &MapData{}, &Segment{}, &MapDataDetails{}, &MapPoint{}, &RouteSegment{}, &RouteSegmentMatch{},
			&WorkoutIntervalRecord{}, &Follower{}, &APOutboxWorkout{}, &APOutboxEntry{}, &APOutboxDelivery{},
		)
	}); err != nil {
		return nil, err
	}

	if err := setUserAPIKeys(db); err != nil {
		return nil, err
	}

	return db, nil
}

func setUserAPIKeys(db *gorm.DB) error {
	users := make([]*User, 0)
	if err := db.Find(&users).Error; err != nil {
		return err
	}

	for _, u := range users {
		if u.APIKey != "" {
			continue
		}

		if err := u.Save(db); err != nil {
			return err
		}
	}

	return nil
}

func dialectorFor(driver, dsn string) (gorm.Dialector, error) {
	switch driver {
	case "sqlite":
		return sqlite.Open(dsn), nil
	case "memory":
		return sqlite.Open(":memory:"), nil
	case "mysql":
		return mysql.Open(dsn), nil
	case "postgres":
		return postgres.Open(dsn), nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsuportedDriver, driver)
	}
}
