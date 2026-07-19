package common

import (
	"context"
	"testing"

	"github.com/go-redis/redis/v8"
)

func TestGetRedisConnectionReportWhenDisabled(t *testing.T) {
	previousEnabled, previousClient := RedisEnabled, RDB
	t.Cleanup(func() {
		RedisEnabled, RDB = previousEnabled, previousClient
	})

	RedisEnabled = false
	RDB = redis.NewClient(&redis.Options{Addr: "client-must-not-be-used:6379"})
	t.Cleanup(func() { _ = RDB.Close() })

	report := GetRedisConnectionReport(context.Background())
	if report.Enabled || report.Connected {
		t.Fatalf("expected disabled report, got %+v", report)
	}
	if report.Error != "" {
		t.Fatalf("expected no error for disabled Redis, got %q", report.Error)
	}
}

func TestGetRedisConnectionReportWithNilClient(t *testing.T) {
	previousEnabled, previousClient := RedisEnabled, RDB
	t.Cleanup(func() {
		RedisEnabled, RDB = previousEnabled, previousClient
	})

	RedisEnabled = true
	RDB = nil

	report := GetRedisConnectionReport(context.Background())
	if report.Connected {
		t.Fatal("expected Redis to be disconnected")
	}
	if report.Error == "" {
		t.Fatal("expected an initialization error")
	}
}

func TestParseRedisInfo(t *testing.T) {
	info := "# Server\r\nredis_version:7.4.1\r\nredis_mode:standalone\r\n\r\n# Stats\r\ntotal_commands_processed:12345\r\nmalformed\r\n"
	values := parseRedisInfo(info)
	report := redisServerReportFromInfo(values)

	if report.Version != "7.4.1" {
		t.Fatalf("unexpected Redis version: %q", report.Version)
	}
	if report.Mode != "standalone" {
		t.Fatalf("unexpected Redis mode: %q", report.Mode)
	}
	if report.TotalCommandsProcessed != 12345 {
		t.Fatalf("unexpected command count: %d", report.TotalCommandsProcessed)
	}
}
