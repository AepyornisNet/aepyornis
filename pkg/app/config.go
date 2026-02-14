package app

import "github.com/jovandeginste/workout-tracker/v2/pkg/container"

func (a *App) ResetConfiguration() error {
	if a.Config == nil {
		cfg, err := container.NewConfig()
		if err != nil {
			return err
		}

		a.Config = cfg
	}

	return a.Config.Reset(a.db)
}
