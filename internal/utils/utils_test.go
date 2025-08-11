package utils

import (
	"testing"

	"github.com/cbridges/nginx-proxy-manager-helper/internal/models"
)

func TestEqualStringSlices(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected bool
	}{
		{
			name:     "Empty slices",
			a:        []string{},
			b:        []string{},
			expected: true,
		},
		{
			name:     "Nil slices",
			a:        nil,
			b:        nil,
			expected: true,
		},
		{
			name:     "One empty, one nil",
			a:        []string{},
			b:        nil,
			expected: true,
		},
		{
			name:     "Identical single element",
			a:        []string{"test"},
			b:        []string{"test"},
			expected: true,
		},
		{
			name:     "Different single element",
			a:        []string{"test1"},
			b:        []string{"test2"},
			expected: false,
		},
		{
			name:     "Same elements same order",
			a:        []string{"a", "b", "c"},
			b:        []string{"a", "b", "c"},
			expected: true,
		},
		{
			name:     "Same elements different order",
			a:        []string{"a", "b", "c"},
			b:        []string{"c", "b", "a"},
			expected: true,
		},
		{
			name:     "Different lengths",
			a:        []string{"a", "b"},
			b:        []string{"a", "b", "c"},
			expected: false,
		},
		{
			name:     "Different elements",
			a:        []string{"a", "b", "c"},
			b:        []string{"d", "e", "f"},
			expected: false,
		},
		{
			name:     "Duplicate elements same count",
			a:        []string{"a", "a", "b"},
			b:        []string{"b", "a", "a"},
			expected: true,
		},
		{
			name:     "Duplicate elements different count",
			a:        []string{"a", "a", "b"},
			b:        []string{"a", "b", "b"},
			expected: false,
		},
		{
			name:     "One slice has duplicates",
			a:        []string{"a", "b", "c"},
			b:        []string{"a", "a", "b", "c"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EqualStringSlices(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("EqualStringSlices(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestEqualDomainConfigs(t *testing.T) {
	tests := []struct {
		name     string
		a        []models.DomainConfig
		b        []models.DomainConfig
		expected bool
	}{
		{
			name:     "Empty slices",
			a:        []models.DomainConfig{},
			b:        []models.DomainConfig{},
			expected: true,
		},
		{
			name:     "Nil slices",
			a:        nil,
			b:        nil,
			expected: true,
		},
		{
			name:     "One empty, one nil",
			a:        []models.DomainConfig{},
			b:        nil,
			expected: true,
		},
		{
			name: "Identical single config",
			a: []models.DomainConfig{
				{Domain: "example.com", Address: "192.168.1.1", Port: "8080"},
			},
			b: []models.DomainConfig{
				{Domain: "example.com", Address: "192.168.1.1", Port: "8080"},
			},
			expected: true,
		},
		{
			name: "Different domains",
			a: []models.DomainConfig{
				{Domain: "example.com", Address: "192.168.1.1", Port: "8080"},
			},
			b: []models.DomainConfig{
				{Domain: "test.com", Address: "192.168.1.1", Port: "8080"},
			},
			expected: false,
		},
		{
			name: "Different addresses",
			a: []models.DomainConfig{
				{Domain: "example.com", Address: "192.168.1.1", Port: "8080"},
			},
			b: []models.DomainConfig{
				{Domain: "example.com", Address: "192.168.1.2", Port: "8080"},
			},
			expected: false,
		},
		{
			name: "Different ports",
			a: []models.DomainConfig{
				{Domain: "example.com", Address: "192.168.1.1", Port: "8080"},
			},
			b: []models.DomainConfig{
				{Domain: "example.com", Address: "192.168.1.1", Port: "9000"},
			},
			expected: false,
		},
		{
			name: "Same configs same order",
			a: []models.DomainConfig{
				{Domain: "api.example.com", Address: "192.168.1.1", Port: "8080"},
				{Domain: "web.example.com", Address: "192.168.1.2", Port: "3000"},
			},
			b: []models.DomainConfig{
				{Domain: "api.example.com", Address: "192.168.1.1", Port: "8080"},
				{Domain: "web.example.com", Address: "192.168.1.2", Port: "3000"},
			},
			expected: true,
		},
		{
			name: "Same configs different order",
			a: []models.DomainConfig{
				{Domain: "api.example.com", Address: "192.168.1.1", Port: "8080"},
				{Domain: "web.example.com", Address: "192.168.1.2", Port: "3000"},
			},
			b: []models.DomainConfig{
				{Domain: "web.example.com", Address: "192.168.1.2", Port: "3000"},
				{Domain: "api.example.com", Address: "192.168.1.1", Port: "8080"},
			},
			expected: true,
		},
		{
			name: "Different lengths",
			a: []models.DomainConfig{
				{Domain: "api.example.com", Address: "192.168.1.1", Port: "8080"},
			},
			b: []models.DomainConfig{
				{Domain: "api.example.com", Address: "192.168.1.1", Port: "8080"},
				{Domain: "web.example.com", Address: "192.168.1.2", Port: "3000"},
			},
			expected: false,
		},
		{
			name: "Duplicate configs same count",
			a: []models.DomainConfig{
				{Domain: "example.com", Address: "192.168.1.1", Port: "8080"},
				{Domain: "example.com", Address: "192.168.1.1", Port: "8080"},
			},
			b: []models.DomainConfig{
				{Domain: "example.com", Address: "192.168.1.1", Port: "8080"},
				{Domain: "example.com", Address: "192.168.1.1", Port: "8080"},
			},
			expected: true,
		},
		{
			name: "Duplicate configs different count",
			a: []models.DomainConfig{
				{Domain: "example.com", Address: "192.168.1.1", Port: "8080"},
				{Domain: "example.com", Address: "192.168.1.1", Port: "8080"},
			},
			b: []models.DomainConfig{
				{Domain: "example.com", Address: "192.168.1.1", Port: "8080"},
			},
			expected: false,
		},
		{
			name: "Mixed configs",
			a: []models.DomainConfig{
				{Domain: "api.example.com", Address: "192.168.1.1", Port: "8080"},
				{Domain: "web.example.com", Address: "192.168.1.2", Port: "3000"},
				{Domain: "db.example.com", Address: "192.168.1.3", Port: "5432"},
			},
			b: []models.DomainConfig{
				{Domain: "db.example.com", Address: "192.168.1.3", Port: "5432"},
				{Domain: "api.example.com", Address: "192.168.1.1", Port: "8080"},
				{Domain: "web.example.com", Address: "192.168.1.2", Port: "3000"},
			},
			expected: true,
		},
		{
			name: "HTTPS and HTTP protocols",
			a: []models.DomainConfig{
				{Domain: "https://secure.example.com", Address: "192.168.1.1", Port: "8443"},
				{Domain: "http://plain.example.com", Address: "192.168.1.2", Port: "8080"},
			},
			b: []models.DomainConfig{
				{Domain: "http://plain.example.com", Address: "192.168.1.2", Port: "8080"},
				{Domain: "https://secure.example.com", Address: "192.168.1.1", Port: "8443"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EqualDomainConfigs(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("EqualDomainConfigs(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}
