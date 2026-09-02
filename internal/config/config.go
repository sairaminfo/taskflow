// Package config loads runtime configuration from environment variables with
// sensible defaults, so the service is twelve-factor friendly and container ready.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all tunable runtime settings.
type Config struct {
	Addr            string        // HTTP listen address, e.g. ":8080"
	ShutdownTimeout time.Duration // max time to wait for graceful shutdown
	EventBuffer     int           // event queue capacity
	EventWorkers    int           // number of event worker goroutines
}

// Load reads configuration from the environment.
func Load() Config {
	return Config{
		Addr:            getString("TASKFLOW_ADDR", ":8080"),
		ShutdownTimeout: getDuration("TASKFLOW_SHUTDOWN_TIMEOUT", 10*time.Second),
		EventBuffer:     getInt("TASKFLOW_EVENT_BUFFER", 256),
		EventWorkers:    getInt("TASKFLOW_EVENT_WORKERS", 4),
	}
}

func getString(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getDuration(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
