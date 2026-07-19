package common

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseRedisOptionsUsesConcurrencyAwarePoolDefaults(t *testing.T) {
	clearRedisOptionEnv(t)

	options, err := parseRedisOptions("redis://localhost:6379/2")
	require.NoError(t, err)
	require.Equal(t, 2, options.DB)
	require.Equal(t, 10*runtime.GOMAXPROCS(0), options.PoolSize)
	require.Equal(t, min(runtime.GOMAXPROCS(0), options.PoolSize), options.MinIdleConns)
}

func TestParseRedisOptionsPreservesConnectionStringSettings(t *testing.T) {
	clearRedisOptionEnv(t)

	options, err := parseRedisOptions(
		"redis://localhost:6379?pool_size=7&min_idle_conns=3&read_timeout=1500ms",
	)
	require.NoError(t, err)
	require.Equal(t, 7, options.PoolSize)
	require.Equal(t, 3, options.MinIdleConns)
	require.Equal(t, 1500*time.Millisecond, options.ReadTimeout)
}

func TestParseRedisOptionsAppliesPerformanceOverrides(t *testing.T) {
	clearRedisOptionEnv(t)
	t.Setenv("REDIS_POOL_SIZE", "24")
	t.Setenv("REDIS_MIN_IDLE_CONNS", "6")
	t.Setenv("REDIS_DIAL_TIMEOUT", "750ms")
	t.Setenv("REDIS_READ_TIMEOUT", "2s")
	t.Setenv("REDIS_WRITE_TIMEOUT", "2500ms")
	t.Setenv("REDIS_POOL_TIMEOUT", "3s")

	options, err := parseRedisOptions("redis://localhost:6379")
	require.NoError(t, err)
	require.Equal(t, 24, options.PoolSize)
	require.Equal(t, 6, options.MinIdleConns)
	require.Equal(t, 750*time.Millisecond, options.DialTimeout)
	require.Equal(t, 2*time.Second, options.ReadTimeout)
	require.Equal(t, 2500*time.Millisecond, options.WriteTimeout)
	require.Equal(t, 3*time.Second, options.PoolTimeout)
}

func TestParseRedisOptionsCapsMinimumIdleConnections(t *testing.T) {
	clearRedisOptionEnv(t)
	t.Setenv("REDIS_POOL_SIZE", "4")
	t.Setenv("REDIS_MIN_IDLE_CONNS", "8")

	options, err := parseRedisOptions("redis://localhost:6379")
	require.NoError(t, err)
	require.Equal(t, 4, options.PoolSize)
	require.Equal(t, 4, options.MinIdleConns)
}

func clearRedisOptionEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"REDIS_POOL_SIZE",
		"REDIS_MIN_IDLE_CONNS",
		"REDIS_DIAL_TIMEOUT",
		"REDIS_READ_TIMEOUT",
		"REDIS_WRITE_TIMEOUT",
		"REDIS_POOL_TIMEOUT",
	} {
		t.Setenv(name, "")
	}
}
