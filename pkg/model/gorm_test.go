package model

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGorm_Connect(t *testing.T) {
	if dsn := os.Getenv("WT_DSN"); dsn != "" {
		db, err := Connect("postgres", dsn, false, slog.Default())
		require.NoError(t, err)
		assert.NotNil(t, db)
	}

	db, err := Connect("invalid-driver", "some-dsn", false, slog.Default())
	require.Error(t, err)
	assert.Nil(t, db)

	db, err = Connect("sqlite", "some-dsn", false, slog.Default())
	require.Error(t, err)
	assert.Nil(t, db)
}
