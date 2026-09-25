package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Setenv("POSTGRES_DSN", "postgres://u:p@localhost:5432/jongyoung?sslmode=disable")
	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, ":8080", cfg.App.Addr())
	assert.Equal(t, "http://keycloak.jongyoung.localhost/realms/jongyoung", cfg.Keycloak.Issuer())
}

func TestLoadRejectsBadConfig(t *testing.T) {
	t.Run("ไม่มี DSN", func(t *testing.T) {
		t.Setenv("POSTGRES_DSN", "")
		_, err := Load()
		assert.Error(t, err)
	})
	t.Run("DSN ไม่มีชื่อ database", func(t *testing.T) {
		t.Setenv("POSTGRES_DSN", "postgres://u:p@localhost:5432/")
		_, err := Load()
		assert.ErrorContains(t, err, "ไม่มีชื่อ database")
	})
	t.Run("APP_ENV ผิด", func(t *testing.T) {
		t.Setenv("POSTGRES_DSN", "postgres://u:p@localhost:5432/jongyoung")
		t.Setenv("APP_ENV", "staging")
		_, err := Load()
		assert.Error(t, err)
	})
}
