package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string 
	Port        string 
}

func Load() (*Config, error) {
	// Load configuration from environment variables or other sources 
	var err error = godotenv.Load() // Load environment variables from .env file
	if err != nil {
		log.Printf("Failed to load ENV variables: %v", err)
		return nil, err
	}

	var config *Config = &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Port:        os.Getenv("PORT"),
	}

	return config, nil

}