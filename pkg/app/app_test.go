package app

import (
	"log/slog"
	"testing"

	appassets "github.com/AepyornisNet/aepyornis/assets"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/AepyornisNet/aepyornis/pkg/version"
	apptranslations "github.com/AepyornisNet/aepyornis/translations"
	"github.com/fsouza/slognil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func defaultApp(t *testing.T) *App {
	t.Helper()
	t.Setenv("WT_LOGGING", "false")

	a := NewApp(version.Version{RefName: "test"})

	a.Assets = appassets.FS()
	a.Translations = apptranslations.FS()

	return a
}

func TestApp_RandomJWTError(t *testing.T) {
	a1 := defaultApp(t)
	s1 := a1.Config.JWTSecret()
	assert.NotEmpty(t, s1)

	a2 := defaultApp(t)
	s2 := a2.Config.JWTSecret()
	assert.NotEqual(t, s1, s2)
}

func TestApp_NewApp(t *testing.T) {
	a := defaultApp(t)
	assert.NotNil(t, a.rawLogger)
	assert.NotNil(t, a.logger)
	assert.IsType(t, slognil.Handler{}, a.logger.Handler())
	assert.Equal(t, "test", a.Version.RefName)
}

func TestApp_Configure(t *testing.T) {
	a := defaultApp(t)
	assert.Nil(t, a.db)

	t.Setenv("WT_DATABASE_DRIVER", "memory")
	require.NoError(t, a.Configure())

	assert.Equal(t, "memory", a.Config.DatabaseDriver)
	assert.NotNil(t, a.db)
}

func TestApp_NewLogger(t *testing.T) {
	l := newLogger(false)
	assert.IsType(t, slognil.Handler{}, l.Handler())

	l = newLogger(true)
	assert.IsType(t, &slog.JSONHandler{}, l.Handler())
}

func TestApp_RandomJWTErrorIdemPotent(t *testing.T) {
	a := defaultApp(t)
	s1 := a.Config.JWTSecret()
	assert.NotEmpty(t, s1)

	s2 := a.Config.JWTSecret()
	assert.Equal(t, s1, s2)
}

func TestApp_Configure_CreatesActivityPubEnabledAdminWhenEnabled(t *testing.T) {
	a := defaultApp(t)

	t.Setenv("WT_DATABASE_DRIVER", "memory")
	t.Setenv("WT_ACTIVITY_PUB_ACTIVE", "true")
	require.NoError(t, a.Configure())

	var admin model.User
	require.NoError(t, a.db.Preload("Profile").First(&admin).Error)

	assert.True(t, admin.Admin)
	assert.True(t, admin.ActivityPub)
	assert.NotEmpty(t, admin.Profile.PublicKey)
	assert.NotEmpty(t, admin.Profile.PrivateKey)
	assert.Equal(t, model.WorkoutVisibilityFollowers, admin.DefaultWorkoutVisibility)
}

func TestApp_Configure_CreatesPrivateAdminDefaultsWhenActivityPubDisabled(t *testing.T) {
	a := defaultApp(t)

	t.Setenv("WT_DATABASE_DRIVER", "memory")
	t.Setenv("WT_ACTIVITY_PUB_ACTIVE", "false")
	require.NoError(t, a.Configure())

	var admin model.User
	require.NoError(t, a.db.Preload("Profile").First(&admin).Error)

	assert.True(t, admin.Admin)
	assert.False(t, admin.ActivityPub)
	assert.Equal(t, model.WorkoutVisibilityPrivate, admin.DefaultWorkoutVisibility)
}
