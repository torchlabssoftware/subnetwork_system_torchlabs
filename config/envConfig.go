package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	PORT string
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
		PORT: getEnv("PORT", "8080"),
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
	required := map[string]string{}

	for key, val := range required {
		if strings.TrimSpace(val) == "" {
			log.Fatalf("Not provide %v in env", key)
		}
	}
}
