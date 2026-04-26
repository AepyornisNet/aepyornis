package app

import (
	"testing"

	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func configuredApp(t *testing.T) *App {
	t.Helper()
	t.Setenv("WT_DATABASE_DRIVER", "memory")

	a := defaultApp(t)

	t.Run("should self-configure", func(t *testing.T) {
		require.NoError(t, a.Configure())
	})

	return a
}

func defaultUser(db *gorm.DB) *model.User {
	u := &model.User{
		UserData: model.UserData{
			Active: true,
		},
		UserSecrets: model.UserSecrets{
			Email:    "my-username@example.com",
			Password: "my-password",
		},
		Profile: model.Profile{Username: "my-username", DisplayName: "my-name"},
	}

	u.SetDB(db)

	return u
}
