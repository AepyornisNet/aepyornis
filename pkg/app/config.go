package app

import "github.com/AepyornisNet/aepyornis/pkg/config"

func (a *App) ResetConfiguration() error {
	if a.Config == nil {
		cfg, err := config.NewConfig()
		if err != nil {
			return err
		}

		a.Config = cfg
	}

	return a.Config.Reset(a.db)
}
