package app

import (
	"log/slog"

	"github.com/AepyornisNet/aepyornis/pkg/config"
	"github.com/AepyornisNet/aepyornis/pkg/repository"
	"github.com/AepyornisNet/aepyornis/pkg/service"
	"github.com/AepyornisNet/aepyornis/pkg/version"
	"github.com/alexedwards/scs/v2"
	"github.com/samber/do/v2"
	"github.com/vgarvardt/gue/v6"
	"gorm.io/gorm"
)

func newInjector(
	db *gorm.DB,
	cfg *config.Config,
	v *version.Version,
	sessionManager *scs.SessionManager,
	logger *slog.Logger,
	gueClient *gue.Client,
) do.Injector {
	injector := do.New(repository.Package, service.Package)
	do.ProvideValue(injector, db)
	do.ProvideValue(injector, cfg)
	do.ProvideValue(injector, v)
	do.ProvideValue(injector, sessionManager)
	do.ProvideValue(injector, logger)
	do.ProvideValue(injector, gueClient)

	return injector
}
