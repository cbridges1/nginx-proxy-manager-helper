package config

import (
	"os"
	"reflect"
	"testing"
)

func TestConfig(t *testing.T) {
	config := &Config{
		NPMBaseURL:          "http://test:8181",
		NPMEmail:            "test@example.com",
		NPMPassword:         "testpass",
		CloudflareEnabled:   true,
		CloudflareToken:     "testtoken",
		CloudflareZoneID:    "testzone",
		CloudflareDomains:   []string{"example.com", "test.com"},
		CreateWildcardCerts: true,
		LetsEncryptEmail:    "ssl@example.com",
	}

	if config.NPMBaseURL != "http://test:8181" {
		t.Errorf("Expected NPMBaseURL to be 'http://test:8181', got '%s'", config.NPMBaseURL)
	}

	if config.NPMEmail != "test@example.com" {
		t.Errorf("Expected NPMEmail to be 'test@example.com', got '%s'", config.NPMEmail)
	}

	if !config.CloudflareEnabled {
		t.Error("Expected CloudflareEnabled to be true")
	}

	if !config.CreateWildcardCerts {
		t.Error("Expected CreateWildcardCerts to be true")
	}

	expectedDomains := []string{"example.com", "test.com"}
	if !reflect.DeepEqual(config.CloudflareDomains, expectedDomains) {
		t.Errorf("Expected CloudflareDomains to be %v, got %v", expectedDomains, config.CloudflareDomains)
	}
}

func TestLoadWithEnvironmentVariables(t *testing.T) {
	// Set up environment variables for testing
	envVars := map[string]string{
		"NPM_BASE_URL":          "http://env-test:8181",
		"NPM_EMAIL":             "env@example.com",
		"NPM_PASSWORD":          "envpass",
		"CLOUDFLARE_ENABLED":    "true",
		"CLOUDFLARE_TOKEN":      "envtoken",
		"CLOUDFLARE_ZONE_ID":    "envzone",
		"CLOUDFLARE_DOMAINS":    "env1.com,env2.com,env3.com",
		"CREATE_WILDCARD_CERTS": "true",
		"LETSENCRYPT_EMAIL":     "envssl@example.com",
	}

	// Set environment variables
	for key, value := range envVars {
		os.Setenv(key, value)
	}

	// Clean up environment variables after test
	defer func() {
		for key := range envVars {
			os.Unsetenv(key)
		}
	}()

	config, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if config.NPMBaseURL != "http://env-test:8181" {
		t.Errorf("Expected NPMBaseURL to be 'http://env-test:8181', got '%s'", config.NPMBaseURL)
	}

	if config.NPMEmail != "env@example.com" {
		t.Errorf("Expected NPMEmail to be 'env@example.com', got '%s'", config.NPMEmail)
	}

	if config.NPMPassword != "envpass" {
		t.Errorf("Expected NPMPassword to be 'envpass', got '%s'", config.NPMPassword)
	}

	if !config.CloudflareEnabled {
		t.Error("Expected CloudflareEnabled to be true")
	}

	if config.CloudflareToken != "envtoken" {
		t.Errorf("Expected CloudflareToken to be 'envtoken', got '%s'", config.CloudflareToken)
	}

	if config.CloudflareZoneID != "envzone" {
		t.Errorf("Expected CloudflareZoneID to be 'envzone', got '%s'", config.CloudflareZoneID)
	}

	expectedDomains := []string{"env1.com", "env2.com", "env3.com"}
	if !reflect.DeepEqual(config.CloudflareDomains, expectedDomains) {
		t.Errorf("Expected CloudflareDomains to be %v, got %v", expectedDomains, config.CloudflareDomains)
	}

	if !config.CreateWildcardCerts {
		t.Error("Expected CreateWildcardCerts to be true")
	}

	if config.LetsEncryptEmail != "envssl@example.com" {
		t.Errorf("Expected LetsEncryptEmail to be 'envssl@example.com', got '%s'", config.LetsEncryptEmail)
	}
}

func TestLoadWithDefaults(t *testing.T) {
	// Clear any existing environment variables that might affect the test
	envVars := []string{
		"NPM_BASE_URL", "NPM_EMAIL", "NPM_PASSWORD",
		"CLOUDFLARE_ENABLED", "CLOUDFLARE_TOKEN", "CLOUDFLARE_ZONE_ID", "CLOUDFLARE_DOMAINS",
		"CREATE_WILDCARD_CERTS", "LETSENCRYPT_EMAIL",
	}

	for _, key := range envVars {
		os.Unsetenv(key)
	}

	config, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	// Test default values
	if config.NPMBaseURL != "http://localhost:81" {
		t.Errorf("Expected default NPMBaseURL to be 'http://localhost:81', got '%s'", config.NPMBaseURL)
	}

	if config.NPMEmail != "admin@example.com" {
		t.Errorf("Expected default NPMEmail to be 'admin@example.com', got '%s'", config.NPMEmail)
	}

	if config.NPMPassword != "changeme" {
		t.Errorf("Expected default NPMPassword to be 'changeme', got '%s'", config.NPMPassword)
	}

	if config.CloudflareEnabled {
		t.Error("Expected default CloudflareEnabled to be false")
	}

	if config.CloudflareToken != "" {
		t.Errorf("Expected default CloudflareToken to be empty, got '%s'", config.CloudflareToken)
	}

	if config.CloudflareZoneID != "" {
		t.Errorf("Expected default CloudflareZoneID to be empty, got '%s'", config.CloudflareZoneID)
	}

	if len(config.CloudflareDomains) != 0 {
		t.Errorf("Expected default CloudflareDomains to be empty, got %v", config.CloudflareDomains)
	}

	if config.CreateWildcardCerts {
		t.Error("Expected default CreateWildcardCerts to be false")
	}

	if config.LetsEncryptEmail != "" {
		t.Errorf("Expected default LetsEncryptEmail to be empty, got '%s'", config.LetsEncryptEmail)
	}
}

func TestDomainsParsing(t *testing.T) {
	tests := []struct {
		name     string
		domains  string
		expected []string
	}{
		{
			name:     "Single domain",
			domains:  "example.com",
			expected: []string{"example.com"},
		},
		{
			name:     "Multiple domains",
			domains:  "example.com,test.com,demo.com",
			expected: []string{"example.com", "test.com", "demo.com"},
		},
		{
			name:     "Domains with spaces",
			domains:  " example.com , test.com , demo.com ",
			expected: []string{"example.com", "test.com", "demo.com"},
		},
		{
			name:     "Empty domain in list",
			domains:  "example.com,,test.com",
			expected: []string{"example.com", "test.com"},
		},
		{
			name:     "Empty string",
			domains:  "",
			expected: []string{},
		},
		{
			name:     "Only commas",
			domains:  ",,,",
			expected: []string{},
		},
		{
			name:     "Wildcard domains",
			domains:  "*.example.com,example.com",
			expected: []string{"*.example.com", "example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("CLOUDFLARE_DOMAINS", tt.domains)
			defer os.Unsetenv("CLOUDFLARE_DOMAINS")

			config, err := Load()
			if err != nil {
				t.Fatalf("Load() returned error: %v", err)
			}

			if !reflect.DeepEqual(config.CloudflareDomains, tt.expected) {
				t.Errorf("Expected CloudflareDomains to be %v, got %v", tt.expected, config.CloudflareDomains)
			}
		})
	}
}

func TestCreateWildcardCertsDefault(t *testing.T) {
	// Test that CREATE_WILDCARD_CERTS defaults to false when not set
	os.Unsetenv("CREATE_WILDCARD_CERTS")

	config, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if config.CreateWildcardCerts {
		t.Error("Expected CreateWildcardCerts to be false when environment variable is not set")
	}
}
