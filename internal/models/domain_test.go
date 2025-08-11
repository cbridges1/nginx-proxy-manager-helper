package models

import (
	"reflect"
	"testing"
)

func TestExtractNginxDomains(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		expected []DomainConfig
	}{
		{
			name:     "No nginx domain labels",
			labels:   map[string]string{"other": "value"},
			expected: []DomainConfig{},
		},
		{
			name: "Single nginx domain",
			labels: map[string]string{
				"nginx-domain": "example.com~192.168.1.10~8080",
			},
			expected: []DomainConfig{
				{Domain: "example.com", Address: "192.168.1.10", Port: "8080"},
			},
		},
		{
			name: "Multiple nginx domains with numbered suffix",
			labels: map[string]string{
				"nginx-domain-1": "api.example.com~192.168.1.10~8080",
				"nginx-domain-2": "web.example.com~192.168.1.10~3000",
			},
			expected: []DomainConfig{
				{Domain: "api.example.com", Address: "192.168.1.10", Port: "8080"},
				{Domain: "web.example.com", Address: "192.168.1.10", Port: "3000"},
			},
		},
		{
			name: "Multiple entries in single label with comma separator",
			labels: map[string]string{
				"nginx-domain": "api.example.com~192.168.1.10~8080,web.example.com~192.168.1.10~3000",
			},
			expected: []DomainConfig{
				{Domain: "api.example.com", Address: "192.168.1.10", Port: "8080"},
				{Domain: "web.example.com", Address: "192.168.1.10", Port: "3000"},
			},
		},
		{
			name: "Multiple entries in single label with semicolon separator",
			labels: map[string]string{
				"nginx-domain": "api.example.com~192.168.1.10~8080;web.example.com~192.168.1.10~3000",
			},
			expected: []DomainConfig{
				{Domain: "api.example.com", Address: "192.168.1.10", Port: "8080"},
				{Domain: "web.example.com", Address: "192.168.1.10", Port: "3000"},
			},
		},
		{
			name: "Domain with HTTPS protocol",
			labels: map[string]string{
				"nginx-domain": "https://secure.example.com~192.168.1.10~8443",
			},
			expected: []DomainConfig{
				{Domain: "https://secure.example.com", Address: "192.168.1.10", Port: "8443"},
			},
		},
		{
			name: "Domain with HTTP protocol",
			labels: map[string]string{
				"nginx-domain": "http://plain.example.com~192.168.1.10~8080",
			},
			expected: []DomainConfig{
				{Domain: "http://plain.example.com", Address: "192.168.1.10", Port: "8080"},
			},
		},
		{
			name: "Invalid format - missing parts",
			labels: map[string]string{
				"nginx-domain": "incomplete~data",
			},
			expected: []DomainConfig{},
		},
		{
			name: "Invalid format - too many parts",
			labels: map[string]string{
				"nginx-domain": "domain~address~port~extra",
			},
			expected: []DomainConfig{},
		},
		{
			name: "Empty fields",
			labels: map[string]string{
				"nginx-domain": "~192.168.1.10~8080",
			},
			expected: []DomainConfig{},
		},
		{
			name: "Mixed valid and invalid entries",
			labels: map[string]string{
				"nginx-domain": "valid.example.com~192.168.1.10~8080,invalid~data,another.example.com~192.168.1.11~9000",
			},
			expected: []DomainConfig{
				{Domain: "valid.example.com", Address: "192.168.1.10", Port: "8080"},
				{Domain: "another.example.com", Address: "192.168.1.11", Port: "9000"},
			},
		},
		{
			name: "Whitespace handling",
			labels: map[string]string{
				"nginx-domain": "  spaced.example.com  ~  192.168.1.10  ~  8080  ",
			},
			expected: []DomainConfig{
				{Domain: "spaced.example.com", Address: "192.168.1.10", Port: "8080"},
			},
		},
		{
			name: "Empty label value",
			labels: map[string]string{
				"nginx-domain": "",
			},
			expected: []DomainConfig{},
		},
		{
			name: "Only whitespace in label value",
			labels: map[string]string{
				"nginx-domain": "   ",
			},
			expected: []DomainConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractNginxDomains(tt.labels)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ExtractNginxDomains() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDomainConfig(t *testing.T) {
	config := DomainConfig{
		Domain:  "test.example.com",
		Address: "192.168.1.100",
		Port:    "8080",
	}

	if config.Domain != "test.example.com" {
		t.Errorf("Expected Domain to be 'test.example.com', got '%s'", config.Domain)
	}

	if config.Address != "192.168.1.100" {
		t.Errorf("Expected Address to be '192.168.1.100', got '%s'", config.Address)
	}

	if config.Port != "8080" {
		t.Errorf("Expected Port to be '8080', got '%s'", config.Port)
	}
}
