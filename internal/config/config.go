package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL    string
	Port           string
	FrontendOrigin string
	CookieSecure   bool
}

func Load() (*Config, error) {
	loadEnvFile(".env")
	loadEnvFile(".env.local")

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" || strings.Contains(databaseURL, "${") {
		databaseURL = buildDatabaseURL()
	}

	var config *Config = &Config{
		DatabaseURL:    databaseURL,
		Port:           fallbackEnv("PORT", "5000"),
		FrontendOrigin: fallbackEnv("FRONTEND_ORIGIN", "http://localhost:5173"),
		CookieSecure:   strings.EqualFold(os.Getenv("COOKIE_SECURE"), "true"),
	}

	return config, nil

}

func loadEnvFile(path string) {
	if _, err := os.Stat(path); err != nil {
		return
	}

	if err := godotenv.Load(filepath.Clean(path)); err != nil {
		log.Printf("Failed to load ENV file %s: %v", path, err)
	}
}

func buildDatabaseURL() string {
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	host := fallbackEnv("DB_HOST", "localhost")
	port := fallbackEnv("DB_PORT", "5432")
	name := os.Getenv("DB_NAME")
	sslMode := fallbackEnv("DB_SSLMODE", "disable")

	if user == "" || password == "" || name == "" {
		return ""
	}

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, password, host, port, name, sslMode)
}

func fallbackEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
