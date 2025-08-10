package config

import (
	"log"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	NPMBaseURL          string
	NPMEmail            string
	NPMPassword         string
	CloudflareEnabled   bool
	CloudflareToken     string
	CloudflareZoneID    string
	CloudflareDomains   []string
	CreateWildcardCerts bool
	LetsEncryptEmail    string
}

func Load() (*Config, error) {
	// Set up viper to read from .env file
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("NPM_BASE_URL", "http://localhost:81")
	viper.SetDefault("NPM_EMAIL", "admin@example.com")
	viper.SetDefault("NPM_PASSWORD", "changeme")
	viper.SetDefault("CLOUDFLARE_ENABLED", "false")
	viper.SetDefault("CLOUDFLARE_TOKEN", "")
	viper.SetDefault("CLOUDFLARE_ZONE_ID", "")
	viper.SetDefault("CLOUDFLARE_DOMAINS", "")
	viper.SetDefault("CREATE_WILDCARD_CERTS", "false")
	viper.SetDefault("LETSENCRYPT_EMAIL", "")

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

	// Parse Cloudflare domains from comma-separated string
	domainsStr := viper.GetString("CLOUDFLARE_DOMAINS")
	var domains []string
	if domainsStr != "" {
		for _, domain := range strings.Split(domainsStr, ",") {
			domain = strings.TrimSpace(domain)
			if domain != "" {
				domains = append(domains, domain)
			}
		}
	}

	config := &Config{
		NPMBaseURL:          viper.GetString("NPM_BASE_URL"),
		NPMEmail:            viper.GetString("NPM_EMAIL"),
		NPMPassword:         viper.GetString("NPM_PASSWORD"),
		CloudflareEnabled:   viper.GetBool("CLOUDFLARE_ENABLED"),
		CloudflareToken:     viper.GetString("CLOUDFLARE_TOKEN"),
		CloudflareZoneID:    viper.GetString("CLOUDFLARE_ZONE_ID"),
		CloudflareDomains:   domains,
		CreateWildcardCerts: viper.GetBool("CREATE_WILDCARD_CERTS"),
		LetsEncryptEmail:    viper.GetString("LETSENCRYPT_EMAIL"),
	}

	log.Printf("DEBUG: NPM configuration - URL: %s, Email: %s", config.NPMBaseURL, config.NPMEmail)
	if config.CloudflareEnabled {
		log.Printf("DEBUG: Cloudflare DNS updates enabled for %d domains", len(config.CloudflareDomains))
	} else {
		log.Printf("DEBUG: Cloudflare DNS updates disabled")
	}
	if config.CreateWildcardCerts && config.CloudflareEnabled {
		log.Printf("DEBUG: Wildcard certificate creation enabled for Cloudflare domains")
	} else {
		log.Printf("DEBUG: Wildcard certificate creation disabled")
	}

	return config, nil
}
