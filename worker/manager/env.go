package manager

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type EnvConfig struct {
	CaptainURL string
	APIKey     string
}

func EnvLoad() EnvConfig {
	appEnv := os.Getenv("APP_ENV")
	log.Println("APP_ENV:", appEnv)

	if strings.ToLower(appEnv) == "dev" || strings.ToLower(appEnv) == "" {
		if err := godotenv.Load(".env.dev"); err != nil {
			log.Println("Error in loading .env.dev file:", err)
		}
	} else {
		log.Println("Running in production mode, using environment variables from Docker")
	}

	config := EnvConfig{
		CaptainURL: getEnv("CAPTAIN_URL", ""),
		APIKey:     getEnv("API_KEY", ""),
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

func (c *EnvConfig) validate() {
	required := map[string]string{
		"CAPTAIN_URL": c.CaptainURL,
		"API_KEY":     c.APIKey,
	}

	var message strings.Builder
	isValid := true

	for key, val := range required {
		if strings.TrimSpace(val) == "" {
			message.WriteString("Not provide " + key + " in env.")
			isValid = false
		}
	}

	if !isValid {
		log.Fatalf("%s", message.String())
	}
}
