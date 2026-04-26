package service

import (
	"io"
	"net/http"
	"testing"

	"github.com/AepyornisNet/aepyornis/pkg/config"
	"github.com/AepyornisNet/aepyornis/pkg/model"
	"github.com/labstack/echo/v4"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivityPubRequestService_HTTPClientRoutesLocalRequests(t *testing.T) {
	e := echo.New()
	e.GET("/ap/users/:username", func(c echo.Context) error {
		return c.String(http.StatusOK, "actor:"+c.Param("username"))
	})

	injector := do.New(Package)
	do.ProvideValue(injector, &config.Config{Config: model.Config{EnvConfig: model.EnvConfig{
		Host: "https://local.example",
	}}})
	do.ProvideValue(injector, e)

	svc, err := NewActivityPubRequestService(injector)
	require.NoError(t, err)

	resp, err := svc.HTTPClient().Get("https://local.example/ap/users/alice")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "actor:alice", string(body))
}
