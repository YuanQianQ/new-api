package common

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

const redisConnectionsPerCPU = 10

func parseRedisOptions(connectionString string) (*redis.Options, error) {
	options, err := redis.ParseURL(connectionString)
	if err != nil {
		return nil, err
	}

	concurrency := runtime.GOMAXPROCS(0)
	if options.PoolSize <= 0 {
		options.PoolSize = redisConnectionsPerCPU * concurrency
	}
	options.PoolSize = redisIntFromEnv("REDIS_POOL_SIZE", options.PoolSize, 1)

	if options.MinIdleConns <= 0 {
		options.MinIdleConns = min(concurrency, options.PoolSize)
	}
	options.MinIdleConns = redisIntFromEnv("REDIS_MIN_IDLE_CONNS", options.MinIdleConns, 0)
	if options.MinIdleConns > options.PoolSize {
		SysError(fmt.Sprintf(
			"REDIS_MIN_IDLE_CONNS (%d) exceeds REDIS_POOL_SIZE (%d), using pool size",
			options.MinIdleConns,
			options.PoolSize,
		))
		options.MinIdleConns = options.PoolSize
	}

	options.DialTimeout = redisDurationFromEnv("REDIS_DIAL_TIMEOUT", options.DialTimeout)
	options.ReadTimeout = redisDurationFromEnv("REDIS_READ_TIMEOUT", options.ReadTimeout)
	options.WriteTimeout = redisDurationFromEnv("REDIS_WRITE_TIMEOUT", options.WriteTimeout)
	options.PoolTimeout = redisDurationFromEnv("REDIS_POOL_TIMEOUT", options.PoolTimeout)

	return options, nil
}

func redisIntFromEnv(name string, fallback, minimum int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum {
		SysError(fmt.Sprintf("invalid %s=%q, using %d", name, raw, fallback))
		return fallback
	}
	return value
}

func redisDurationFromEnv(name string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}

	value, err := time.ParseDuration(raw)
	if err != nil || value < 0 {
		SysError(fmt.Sprintf("invalid %s=%q, using %s", name, raw, fallback))
		return fallback
	}
	return value
}
