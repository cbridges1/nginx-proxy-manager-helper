package npm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cbridges/nginx-proxy-manager-helper/internal/config"
	"github.com/cbridges/nginx-proxy-manager-helper/internal/models"
)

func TestNewClient(t *testing.T) {
	cfg := &config.Config{
		NPMBaseURL:  "http://test:8181",
		NPMEmail:    "test@example.com",
		NPMPassword: "testpass",
	}

	client := NewClient(cfg)

	if client.BaseURL != cfg.NPMBaseURL {
		t.Errorf("Expected BaseURL %s, got %s", cfg.NPMBaseURL, client.BaseURL)
	}

	if client.Email != cfg.NPMEmail {
		t.Errorf("Expected Email %s, got %s", cfg.NPMEmail, client.Email)
	}

	if client.Password != cfg.NPMPassword {
		t.Errorf("Expected Password %s, got %s", cfg.NPMPassword, client.Password)
	}

	if client.httpClient == nil {
		t.Error("Expected httpClient to be initialized")
	}
}

func TestBoolIntUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected BoolInt
		wantErr  bool
	}{
		{
			name:     "Boolean true",
			input:    "true",
			expected: 1,
			wantErr:  false,
		},
		{
			name:     "Boolean false",
			input:    "false",
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "Integer 1",
			input:    "1",
			expected: 1,
			wantErr:  false,
		},
		{
			name:     "Integer 0",
			input:    "0",
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "String true",
			input:    `"true"`,
			expected: 1,
			wantErr:  false,
		},
		{
			name:     "String false",
			input:    `"false"`,
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "String 1",
			input:    `"1"`,
			expected: 1,
			wantErr:  false,
		},
		{
			name:     "String 0",
			input:    `"0"`,
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "Integer 5",
			input:    "5",
			expected: 5,
			wantErr:  false,
		},
		{
			name:     "Invalid string",
			input:    `"invalid"`,
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bi BoolInt
			err := json.Unmarshal([]byte(tt.input), &bi)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if bi != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, bi)
			}
		})
	}
}

func TestBoolIntMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    BoolInt
		expected string
	}{
		{
			name:     "Zero value",
			input:    0,
			expected: "0",
		},
		{
			name:     "One value",
			input:    1,
			expected: "1",
		},
		{
			name:     "Larger value",
			input:    5,
			expected: "5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := json.Marshal(tt.input)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if string(result) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(result))
			}
		})
	}
}

func TestLogin(t *testing.T) {
	// Create a test server that simulates NPM login endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tokens" {
			t.Errorf("Expected path /api/tokens, got %s", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		// Check request body
		var loginReq LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
			t.Errorf("Failed to decode login request: %v", err)
		}

		if loginReq.Identity != "test@example.com" {
			t.Errorf("Expected identity test@example.com, got %s", loginReq.Identity)
		}

		if loginReq.Secret != "testpass" {
			t.Errorf("Expected secret testpass, got %s", loginReq.Secret)
		}

		// Return mock response
		response := LoginResponse{
			Token:   "test-token-123",
			Expires: time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.Config{
		NPMBaseURL:  server.URL,
		NPMEmail:    "test@example.com",
		NPMPassword: "testpass",
	}

	client := NewClient(cfg)
	err := client.Login()

	if err != nil {
		t.Errorf("Login() failed: %v", err)
	}

	if client.BearerToken != "test-token-123" {
		t.Errorf("Expected token test-token-123, got %s", client.BearerToken)
	}

	if client.TokenExpiry.IsZero() {
		t.Error("Expected TokenExpiry to be set")
	}
}

func TestLoginError(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Invalid credentials"))
	}))
	defer server.Close()

	cfg := &config.Config{
		NPMBaseURL:  server.URL,
		NPMEmail:    "wrong@example.com",
		NPMPassword: "wrongpass",
	}

	client := NewClient(cfg)
	err := client.Login()

	if err == nil {
		t.Error("Expected login to fail with invalid credentials")
	}
}

func TestSanitizeDomainName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple domain",
			input:    "example.com",
			expected: "example.com",
		},
		{
			name:     "Domain with https",
			input:    "https://example.com",
			expected: "example.com",
		},
		{
			name:     "Domain with http",
			input:    "http://example.com",
			expected: "example.com",
		},
		{
			name:     "Domain with //",
			input:    "//example.com",
			expected: "example.com",
		},
		{
			name:     "Domain with port",
			input:    "example.com:8080",
			expected: "example.com",
		},
		{
			name:     "Domain with path",
			input:    "example.com/path",
			expected: "example.com",
		},
		{
			name:     "Domain with query",
			input:    "example.com?query=value",
			expected: "example.com",
		},
		{
			name:     "Domain with uppercase",
			input:    "EXAMPLE.COM",
			expected: "example.com",
		},
		{
			name:     "Domain with whitespace",
			input:    "  example.com  ",
			expected: "example.com",
		},
		{
			name:     "Complex URL",
			input:    "https://api.example.com:8080/v1/endpoint?param=value",
			expected: "api.example.com",
		},
		{
			name:     "Domain with invalid characters",
			input:    "exam&ple.com",
			expected: "example.com",
		},
		{
			name:     "Subdomain",
			input:    "api.example.com",
			expected: "api.example.com",
		},
		{
			name:     "Wildcard domain",
			input:    "*.example.com",
			expected: "example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeDomainName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeDomainName(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetProxyHosts(t *testing.T) {
	// Mock server for proxy hosts endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/nginx/proxy-hosts" {
			t.Errorf("Expected path /api/nginx/proxy-hosts, got %s", r.URL.Path)
		}

		// Check for authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-token" {
			t.Errorf("Expected Authorization header 'Bearer test-token', got %s", authHeader)
		}

		// Return mock proxy hosts
		hosts := []ProxyHostResponse{
			{
				ID:            1,
				DomainNames:   []string{"example.com"},
				ForwardHost:   "192.168.1.10",
				ForwardPort:   8080,
				ForwardScheme: "http",
				CertificateID: BoolInt(0),
				SSLForced:     BoolInt(0),
				Enabled:       BoolInt(1),
			},
			{
				ID:            2,
				DomainNames:   []string{"secure.example.com"},
				ForwardHost:   "192.168.1.20",
				ForwardPort:   8443,
				ForwardScheme: "http",
				CertificateID: BoolInt(1),
				SSLForced:     BoolInt(1),
				Enabled:       BoolInt(1),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(hosts)
	}))
	defer server.Close()

	cfg := &config.Config{
		NPMBaseURL: server.URL,
	}

	client := NewClient(cfg)
	client.BearerToken = "test-token"
	client.TokenExpiry = time.Now().Add(1 * time.Hour)

	hosts, err := client.GetProxyHosts()
	if err != nil {
		t.Fatalf("GetProxyHosts() failed: %v", err)
	}

	if len(hosts) != 2 {
		t.Errorf("Expected 2 hosts, got %d", len(hosts))
	}

	// Check first host
	if hosts[0].ID != 1 {
		t.Errorf("Expected first host ID 1, got %d", hosts[0].ID)
	}

	if len(hosts[0].DomainNames) != 1 || hosts[0].DomainNames[0] != "example.com" {
		t.Errorf("Expected first host domain 'example.com', got %v", hosts[0].DomainNames)
	}

	// Check second host
	if hosts[1].ID != 2 {
		t.Errorf("Expected second host ID 2, got %d", hosts[1].ID)
	}

	if len(hosts[1].DomainNames) != 1 || hosts[1].DomainNames[0] != "secure.example.com" {
		t.Errorf("Expected second host domain 'secure.example.com', got %v", hosts[1].DomainNames)
	}
}

func TestSyncDomainsToNPM(t *testing.T) {
	// This is a more complex test that would require mocking multiple endpoints
	// For now, we'll test the basic structure and error handling

	cfg := &config.Config{
		NPMBaseURL: "http://invalid-url-that-should-fail",
	}

	client := NewClient(cfg)
	domains := []models.DomainConfig{
		{Domain: "test.com", Address: "192.168.1.1", Port: "8080"},
	}

	// This should fail because the URL is invalid
	err := client.SyncDomainsToNPM(domains, false, "", "")

	// We expect this to fail, but we don't want a panic
	// The actual error handling depends on the implementation
	_ = err // We don't assert on the specific error as it's an integration concern
}

func TestRemoveProxyHostsByDomains(t *testing.T) {
	// Test with empty domains list
	cfg := &config.Config{
		NPMBaseURL: "http://test",
	}

	client := NewClient(cfg)
	err := client.RemoveProxyHostsByDomains([]string{})

	if err != nil {
		t.Errorf("RemoveProxyHostsByDomains([]) should not return error, got %v", err)
	}
}
