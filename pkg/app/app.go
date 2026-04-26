package app

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"time"

	"github.com/AepyornisNet/aepyornis/pkg/config"
	"github.com/AepyornisNet/aepyornis/pkg/geocoder"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	_ "github.com/AepyornisNet/aepyornis/pkg/model/migrations"
	"github.com/AepyornisNet/aepyornis/pkg/version"
	"github.com/AepyornisNet/aepyornis/pkg/worker"
	"github.com/alexedwards/scs/v2"
	"github.com/fsouza/slognil"
	"github.com/invopop/ctxi18n/i18n"
	"github.com/labstack/echo/v4"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	"github.com/samber/do/v2"
	"gorm.io/gorm"
)

type App struct {
	Assets       fs.FS
	AssetDir     string
	Translations fs.FS

	echo           *echo.Echo
	logger         *slog.Logger
	rawLogger      *slog.Logger
	db             *gorm.DB
	sessionManager *scs.SessionManager
	translator     *i18n.Locale
	Version        version.Version
	Config         *config.Config
	injector       do.Injector
}

func (a *App) Serve() error {
	w, err := worker.New(a.injector)
	if err != nil {
		return err
	}

	go w.Start(context.Background())

	a.logger.Info("Starting web server on " + a.Config.Bind)

	return a.echo.Start(a.Config.Bind)
}

func (a *App) Configure() error {
	cfg, err := config.NewConfig()
	if err != nil {
		return err
	}

	a.Config = cfg

	if err := a.ConfigureLocalizer(); err != nil {
		return err
	}

	a.ConfigureLogger()

	if err := a.ConfigureDatabase(); err != nil {
		return err
	}

	a.ConfigureGeocoder()

	if err := model.InitTZFinder(); err != nil {
		return err
	}

	if err := a.Config.UpdateFromDatabase(a.db); err != nil {
		return err
	}

	if err := a.ConfigureWebserver(); err != nil {
		return err
	}

	return nil
}

func (a *App) ConfigureGeocoder() {
	if a.Config.Offline {
		geocoder.ForceOffline()
		return
	}

	geocoder.SetClient(a.logger, a.Version.UserAgent())
}

func (a *App) ConfigureDatabase() error {
	a.Config.SetDSN(a.logger)

	a.logger.Info("Connecting to the database '" + a.Config.DatabaseDriver + "': " + a.Config.DSN)

	db, err := model.Connect(a.Config.DatabaseDriver, a.Config.DSN, a.Config.Debug, a.rawLogger)
	if err != nil {
		return err
	}

	if a.Config.Debug {
		db = db.Debug()
	}

	a.db = db

	err = db.First(&model.User{}).Error
	if err == nil {
		return nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return a.createAdminUser()
}

func newLogger(enabled bool) *slog.Logger {
	if !enabled {
		return slognil.NewLogger()
	}

	return slog.New(newLogHandler())
}

func newLogHandler() slog.Handler {
	w := os.Stderr
	if isatty.IsTerminal(w.Fd()) {
		return tint.NewHandler(os.Stderr, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.Kitchen,
		})
	}

	return slog.NewJSONHandler(w, nil)
}

func (a *App) ConfigureLogger() {
	logger := newLogger(a.Config.Logging).
		With("app", "workout-tracker").
		With("version", a.Version.RefName).
		With("sha", a.Version.Sha)

	a.rawLogger = logger
	a.logger = logger.With("module", "app")
}

func NewApp(v version.Version) *App {
	return &App{
		Version:   v,
		Config:    &config.Config{},
		logger:    newLogger(false),
		rawLogger: newLogger(false),
	}
}

func (a *App) createAdminUser() error {
	u := &model.User{
		UserData: model.UserData{
			Active: true,
			Admin:  true,
		},
		UserSecrets: model.UserSecrets{Email: "admin@localhost"},
	}
	u.Profile.Username = "admin"
	u.Profile.DisplayName = "Administrator"
	u.Profile.Local = true

	if err := u.SetPassword("admin"); err != nil {
		return err
	}

	a.logger.Warn("Creating admin user '" + u.Email + "', with password 'admin'")

	u.ResetDefaults()
	if a.Config.ActivityPubActive {
		u.ActivityPub = true
		u.DefaultWorkoutVisibility = model.WorkoutVisibilityFollowers
		u.Profile.User = u

		if err := u.Profile.GenerateActivityPubKeys(false); err != nil {
			return err
		}
	}
	u.Profile.User = u

	return u.Create(a.db)
}

func (a *App) DB() *gorm.DB {
	return a.db
}

func (a *App) Logger() *slog.Logger {
	return a.logger
}
