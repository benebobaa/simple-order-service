package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setBase sets every relevant variable to a known value so tests are isolated
// from the developer's shell environment.
func setBase(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "test")
	t.Setenv("PORT", "8080")
	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/test?sslmode=disable")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("JWT_TTL", "24h")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("AUTO_MIGRATE", "true")
	t.Setenv("DB_MAX_CONNS", "")
	t.Setenv("DB_MIN_CONNS", "")
	t.Setenv("DB_MAX_CONN_LIFETIME", "")
	t.Setenv("DB_MAX_CONN_IDLE_TIME", "")
}

func TestLoadDefaults(t *testing.T) {
	setBase(t)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, time.Hour*24, cfg.JWTTTL)

	// Zero means "use the pgx default".
	assert.Equal(t, int32(0), cfg.DBMaxConns)
	assert.Equal(t, int32(0), cfg.DBMinConns)
	assert.Equal(t, time.Duration(0), cfg.DBMaxConnLifetime)
	assert.Equal(t, time.Duration(0), cfg.DBMaxConnIdleTime)
}

func TestLoadPoolOverrides(t *testing.T) {
	setBase(t)
	t.Setenv("DB_MAX_CONNS", "12")
	t.Setenv("DB_MIN_CONNS", "3")
	t.Setenv("DB_MAX_CONN_LIFETIME", "45m")
	t.Setenv("DB_MAX_CONN_IDLE_TIME", "5m")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, int32(12), cfg.DBMaxConns)
	assert.Equal(t, int32(3), cfg.DBMinConns)
	assert.Equal(t, 45*time.Minute, cfg.DBMaxConnLifetime)
	assert.Equal(t, 5*time.Minute, cfg.DBMaxConnIdleTime)
}

func TestLoadPoolValidation(t *testing.T) {
	testCases := map[string]map[string]string{
		"negative max conns": {"DB_MAX_CONNS": "-1"},
		"invalid max conns":  {"DB_MAX_CONNS": "many"},
		"zero lifetime":      {"DB_MAX_CONN_LIFETIME": "0s"},
		"invalid lifetime":   {"DB_MAX_CONN_LIFETIME": "soon"},
		"min above max":      {"DB_MIN_CONNS": "9", "DB_MAX_CONNS": "4"},
	}

	for name, overrides := range testCases {
		t.Run(name, func(t *testing.T) {
			setBase(t)
			for key, value := range overrides {
				t.Setenv(key, value)
			}

			_, err := Load()
			assert.Error(t, err)
		})
	}
}

func TestLoadRequiresDatabaseURLAndSecret(t *testing.T) {
	setBase(t)
	t.Setenv("DATABASE_URL", "")
	_, err := Load()
	assert.Error(t, err)

	setBase(t)
	t.Setenv("JWT_SECRET", "")
	_, err = Load()
	assert.Error(t, err)
}
