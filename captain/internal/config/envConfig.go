package config

import (
	"log"
	"os"
	"strings"
)

type Config struct {
	PORT                string
	POSTGRES_URL        string
	Admin_API_KEY       string
	Worker_API_KEY      string
	CLICKHOUSE_URL      string
	CLICKHOUSE_DB       string
	CLICKHOUSE_USER     string
	CLICKHOUSE_PASSWORD string
}

func Load() Config {

	config := Config{
		PORT:                getEnv("PORT", "8080"),
		POSTGRES_URL:        getEnv("POSTGRES_URL", ""),
		Admin_API_KEY:       getEnv("ADMIN_API_KEY", ""),
		Worker_API_KEY:      getEnv("WORKER_API_KEY", ""),
		CLICKHOUSE_URL:      getEnv("CLICKHOUSE_URL", ""),
		CLICKHOUSE_DB:       getEnv("CLICKHOUSE_DB", ""),
		CLICKHOUSE_USER:     getEnv("CLICKHOUSE_USER", ""),
		CLICKHOUSE_PASSWORD: getEnv("CLICKHOUSE_PASSWORD", ""),
	}

	config.validate()

	return config
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func (c *Config) validate() {
	required := map[string]string{
		"POSTGRES_URL":        c.POSTGRES_URL,
		"ADMIN_API_KEY":       c.Admin_API_KEY,
		"WORKER_API_KEY":      c.Worker_API_KEY,
		"CLICKHOUSE_URL":      c.CLICKHOUSE_URL,
		"CLICKHOUSE_DB":       c.CLICKHOUSE_DB,
		"CLICKHOUSE_USER":     c.CLICKHOUSE_USER,
		"CLICKHOUSE_PASSWORD": c.CLICKHOUSE_PASSWORD,
	}

	for key, val := range required {
		if strings.TrimSpace(val) == "" {
			log.Fatalf("Not provide %v in env", key)
		}
	}
}
