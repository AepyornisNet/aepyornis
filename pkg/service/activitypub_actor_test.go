package service

import (
	"testing"

	"github.com/AepyornisNet/aepyornis/pkg/config"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/labstack/echo/v4"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivityPubActorService_ActorURLForLocalProfile(t *testing.T) {
	injector := do.New(Package)
	do.ProvideValue(injector, &config.Config{Config: model.Config{EnvConfig: model.EnvConfig{
		Host:    "https://local.example",
		WebRoot: "root",
	}}})
	do.ProvideValue(injector, echo.New())

	svc, err := NewActivityPubActorService(injector)
	require.NoError(t, err)

	actorURL, err := svc.ActorURL(&model.Profile{Username: "alice"})
	require.NoError(t, err)
	assert.Equal(t, "https://local.example/root/ap/users/alice", actorURL)
}

func TestActivityPubActorService_ActorURLForRemoteProfile(t *testing.T) {
	injector := do.New(Package)
	do.ProvideValue(injector, &config.Config{})
	do.ProvideValue(injector, echo.New())

	svc, err := NewActivityPubActorService(injector)
	require.NoError(t, err)

	domain := "remote.example"
	actorURL, err := svc.ActorURL(&model.Profile{Username: "alice", Domain: &domain})
	require.NoError(t, err)
	assert.Equal(t, "https://remote.example/ap/users/alice", actorURL)
}
