package configs_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"reporting-service/configs"
)

func TestLoad_RequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	_, err := configs.Load()
	assert.Error(t, err)
}

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("PORT", "")
	t.Setenv("DEBUG", "")

	cfg, err := configs.Load()
	require.NoError(t, err)
	assert.Equal(t, "8080", cfg.Port)
	assert.False(t, cfg.Debug)
	assert.Equal(t, "postgres://localhost/test", cfg.DatabaseURL)
}

func TestLoad_CustomValues(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("PORT", "9090")
	t.Setenv("DEBUG", "true")

	cfg, err := configs.Load()
	require.NoError(t, err)
	assert.Equal(t, "9090", cfg.Port)
	assert.True(t, cfg.Debug)
}
