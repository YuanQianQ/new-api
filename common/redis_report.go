package common

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type RedisPoolReport struct {
	Size         int    `json:"size"`
	MinIdleConns int    `json:"min_idle_conns"`
	Hits         uint32 `json:"hits"`
	Misses       uint32 `json:"misses"`
	Timeouts     uint32 `json:"timeouts"`
	TotalConns   uint32 `json:"total_conns"`
	IdleConns    uint32 `json:"idle_conns"`
	StaleConns   uint32 `json:"stale_conns"`
}

type RedisServerReport struct {
	Version                  string `json:"version"`
	Mode                     string `json:"mode"`
	Role                     string `json:"role"`
	UptimeSeconds            int64  `json:"uptime_seconds"`
	ConnectedClients         int64  `json:"connected_clients"`
	UsedMemoryBytes          int64  `json:"used_memory_bytes"`
	PeakMemoryBytes          int64  `json:"peak_memory_bytes"`
	MaxMemoryBytes           int64  `json:"max_memory_bytes"`
	KeyCount                 int64  `json:"key_count"`
	TotalConnectionsReceived int64  `json:"total_connections_received"`
	TotalCommandsProcessed   int64  `json:"total_commands_processed"`
	OperationsPerSecond      int64  `json:"operations_per_second"`
}

type RedisConnectionReport struct {
	Enabled             bool              `json:"enabled"`
	Connected           bool              `json:"connected"`
	CheckedAt           int64             `json:"checked_at"`
	PingLatencyMs       float64           `json:"ping_latency_ms"`
	Endpoint            string            `json:"endpoint"`
	Database            int               `json:"database"`
	TLSEnabled          bool              `json:"tls_enabled"`
	Pool                RedisPoolReport   `json:"pool"`
	Server              RedisServerReport `json:"server"`
	ServerInfoAvailable bool              `json:"server_info_available"`
	KeyCountAvailable   bool              `json:"key_count_available"`
	Error               string            `json:"error,omitempty"`
	InfoError           string            `json:"info_error,omitempty"`
}

func GetRedisConnectionReport(ctx context.Context) RedisConnectionReport {
	report := RedisConnectionReport{
		Enabled:   RedisEnabled,
		CheckedAt: time.Now().Unix(),
	}
	if !RedisEnabled {
		return report
	}
	if RDB == nil {
		report.Error = "Redis client is not initialized"
		return report
	}

	options := RDB.Options()
	report.Endpoint = options.Addr
	report.Database = options.DB
	report.TLSEnabled = options.TLSConfig != nil
	report.Pool.Size = options.PoolSize
	report.Pool.MinIdleConns = options.MinIdleConns

	pingStartedAt := time.Now()
	if err := RDB.Ping(ctx).Err(); err != nil {
		report.PingLatencyMs = durationMilliseconds(time.Since(pingStartedAt))
		report.Error = MaskSensitiveInfo(err.Error())
		fillRedisPoolReport(&report)
		return report
	}

	report.PingLatencyMs = durationMilliseconds(time.Since(pingStartedAt))
	report.Connected = true

	pipe := RDB.Pipeline()
	infoCommand := pipe.Info(ctx)
	dbSizeCommand := pipe.DBSize(ctx)
	_, _ = pipe.Exec(ctx)

	info, infoErr := infoCommand.Result()
	dbSize, dbSizeErr := dbSizeCommand.Result()
	if infoErr == nil {
		report.Server = redisServerReportFromInfo(parseRedisInfo(info))
		report.ServerInfoAvailable = true
	}
	if dbSizeErr == nil {
		report.Server.KeyCount = dbSize
		report.KeyCountAvailable = true
	}
	if infoErr != nil || dbSizeErr != nil {
		report.InfoError = joinRedisReportErrors(infoErr, dbSizeErr)
	}

	fillRedisPoolReport(&report)
	report.CheckedAt = time.Now().Unix()
	return report
}

func fillRedisPoolReport(report *RedisConnectionReport) {
	stats := RDB.PoolStats()
	report.Pool.Hits = stats.Hits
	report.Pool.Misses = stats.Misses
	report.Pool.Timeouts = stats.Timeouts
	report.Pool.TotalConns = stats.TotalConns
	report.Pool.IdleConns = stats.IdleConns
	report.Pool.StaleConns = stats.StaleConns
}

func durationMilliseconds(duration time.Duration) float64 {
	return float64(duration.Microseconds()) / 1000
}

func parseRedisInfo(info string) map[string]string {
	values := make(map[string]string)
	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if ok {
			values[key] = strings.TrimSpace(value)
		}
	}
	return values
}

func redisServerReportFromInfo(info map[string]string) RedisServerReport {
	return RedisServerReport{
		Version:                  info["redis_version"],
		Mode:                     info["redis_mode"],
		Role:                     info["role"],
		UptimeSeconds:            redisInfoInt64(info, "uptime_in_seconds"),
		ConnectedClients:         redisInfoInt64(info, "connected_clients"),
		UsedMemoryBytes:          redisInfoInt64(info, "used_memory"),
		PeakMemoryBytes:          redisInfoInt64(info, "used_memory_peak"),
		MaxMemoryBytes:           redisInfoInt64(info, "maxmemory"),
		TotalConnectionsReceived: redisInfoInt64(info, "total_connections_received"),
		TotalCommandsProcessed:   redisInfoInt64(info, "total_commands_processed"),
		OperationsPerSecond:      redisInfoInt64(info, "instantaneous_ops_per_sec"),
	}
}

func redisInfoInt64(info map[string]string, key string) int64 {
	value, _ := strconv.ParseInt(info[key], 10, 64)
	return value
}

func joinRedisReportErrors(infoErr, dbSizeErr error) string {
	errors := make([]string, 0, 2)
	if infoErr != nil {
		errors = append(errors, fmt.Sprintf("INFO: %s", MaskSensitiveInfo(infoErr.Error())))
	}
	if dbSizeErr != nil {
		errors = append(errors, fmt.Sprintf("DBSIZE: %s", MaskSensitiveInfo(dbSizeErr.Error())))
	}
	return strings.Join(errors, "; ")
}
