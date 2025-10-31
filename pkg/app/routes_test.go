package app

import (
	"testing"

	"github.com/jovandeginste/workout-tracker/v2/pkg/database"
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

func defaultUser(db *gorm.DB) *database.User {
	u := &database.User{
		UserData: database.UserData{
			Username: "my-username",
			Name:     "my-name",
			Active:   true,
		},
		UserSecrets: database.UserSecrets{
			Password: "my-password",
		},
	}

	u.SetDB(db)

	return u
}
