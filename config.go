package main

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	NPMBaseURL  string
	NPMEmail    string
	NPMPassword string
}

func LoadConfig() (*Config, error) {
	// Set up viper to read from .env file
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("NPM_BASE_URL", "http://localhost:81")
	viper.SetDefault("NPM_EMAIL", "admin@example.com")
	viper.SetDefault("NPM_PASSWORD", "changeme")

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Printf("DEBUG: No .env file found, using environment variables and defaults")
		} else {
			log.Printf("WARNING: Error reading .env file: %v", err)
		}
	} else {
		log.Printf("DEBUG: Loaded configuration from .env file")
	}

	config := &Config{
		NPMBaseURL:  viper.GetString("NPM_BASE_URL"),
		NPMEmail:    viper.GetString("NPM_EMAIL"),
		NPMPassword: viper.GetString("NPM_PASSWORD"),
	}

	log.Printf("DEBUG: NPM configuration - URL: %s, Email: %s", config.NPMBaseURL, config.NPMEmail)

	return config, nil
}
