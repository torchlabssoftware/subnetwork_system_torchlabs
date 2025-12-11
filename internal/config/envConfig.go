package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	PORT           string
	DBURL          string
	Admin_API_KEY  string
	Worker_API_KEY string
}

func Load() Config {
	appEnv := os.Getenv("APP_ENV")
	log.Println("APP_ENV:", appEnv)

	if strings.ToLower(appEnv) == "dev" || strings.ToLower(appEnv) == "" {
		if err := godotenv.Load(".env.dev"); err != nil {
			log.Println("Error in loading .env.dev file:", err)
		}
	} else {
		log.Println("Running in production mode, using environment variables from Docker")
	}

	config := Config{
		PORT:           getEnv("PORT", "8080"),
		DBURL:          getEnv("DB_URL", ""),
		Admin_API_KEY:  getEnv("ADMIN_API_KEY", ""),
		Worker_API_KEY: getEnv("WORKER_API_KEY", ""),
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
		"DBURL":          c.DBURL,
		"ADMIN_API_KEY":  c.Admin_API_KEY,
		"WORKER_API_KEY": c.Worker_API_KEY,
	}

	for key, val := range required {
		if strings.TrimSpace(val) == "" {
			log.Fatalf("Not provide %v in env", key)
		}
	}
}
