package service

import (
	"net/http"
	"testing"

	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivityPubRequestService_HTTPClientUsesDefaultTransport(t *testing.T) {
	injector := do.New(Package)

	svc, err := NewActivityPubRequestService(injector)
	require.NoError(t, err)

	assert.NotNil(t, svc.HTTPClient())
	assert.Equal(t, http.DefaultTransport, svc.HTTPClient().Transport)
}
