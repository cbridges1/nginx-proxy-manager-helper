package cloudflare

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/cbridges/nginx-proxy-manager-helper/internal/config"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name   string
		config *config.Config
		want   *Client
	}{
		{
			name: "Enabled client",
			config: &config.Config{
				CloudflareEnabled: true,
				CloudflareToken:   "test-token",
				CloudflareZoneID:  "test-zone",
				CloudflareDomains: []string{"example.com", "test.com"},
			},
			want: &Client{
				APIToken: "test-token",
				ZoneID:   "test-zone",
				Enabled:  true,
				Domains:  []string{"example.com", "test.com"},
			},
		},
		{
			name: "Disabled client",
			config: &config.Config{
				CloudflareEnabled: false,
			},
			want: &Client{
				Enabled: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewClient(tt.config)

			if got.Enabled != tt.want.Enabled {
				t.Errorf("NewClient() Enabled = %v, want %v", got.Enabled, tt.want.Enabled)
			}

			if got.Enabled {
				if got.APIToken != tt.want.APIToken {
					t.Errorf("NewClient() APIToken = %v, want %v", got.APIToken, tt.want.APIToken)
				}

				if got.ZoneID != tt.want.ZoneID {
					t.Errorf("NewClient() ZoneID = %v, want %v", got.ZoneID, tt.want.ZoneID)
				}

				if !reflect.DeepEqual(got.Domains, tt.want.Domains) {
					t.Errorf("NewClient() Domains = %v, want %v", got.Domains, tt.want.Domains)
				}

				if got.client == nil {
					t.Error("NewClient() http client should be initialized")
				}
			}
		})
	}
}

func TestGetCurrentIP(t *testing.T) {
	tests := []struct {
		name    string
		client  *Client
		wantErr bool
	}{
		{
			name: "Valid client",
			client: &Client{
				client: &http.Client{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.client.GetCurrentIP()
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.GetCurrentIP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// For successful cases, just check that we got a non-empty IP
			if !tt.wantErr && got == "" {
				t.Error("Client.GetCurrentIP() returned empty IP")
			}
		})
	}
}

func TestUpdateDNSRecords(t *testing.T) {
	tests := []struct {
		name    string
		client  *Client
		wantErr bool
	}{
		{
			name: "Disabled client",
			client: &Client{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "No domains configured",
			client: &Client{
				Enabled: true,
				Domains: []string{},
			},
			wantErr: false,
		},
		{
			name: "Client with domains but will fail on IP detection",
			client: &Client{
				Enabled:  true,
				Domains:  []string{"example.com"},
				APIToken: "test-token",
				ZoneID:   "test-zone",
				client:   &http.Client{},
			},
			wantErr: false, // Should not error on disabled or empty domains
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.client.UpdateDNSRecords()
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.UpdateDNSRecords() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCloudflareStructs(t *testing.T) {
	// Test DNSRecord struct
	record := DNSRecord{
		ID:      "test-id",
		Type:    "A",
		Name:    "example.com",
		Content: "192.168.1.1",
		TTL:     300,
		Proxied: false,
	}

	if record.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got %s", record.ID)
	}

	if record.Type != "A" {
		t.Errorf("Expected Type 'A', got %s", record.Type)
	}

	if record.Name != "example.com" {
		t.Errorf("Expected Name 'example.com', got %s", record.Name)
	}

	if record.Content != "192.168.1.1" {
		t.Errorf("Expected Content '192.168.1.1', got %s", record.Content)
	}

	if record.TTL != 300 {
		t.Errorf("Expected TTL 300, got %d", record.TTL)
	}

	if record.Proxied != false {
		t.Errorf("Expected Proxied false, got %t", record.Proxied)
	}

	// Test Error struct
	cfError := Error{
		Code:    1001,
		Message: "DNS record not found",
	}

	if cfError.Code != 1001 {
		t.Errorf("Expected error code 1001, got %d", cfError.Code)
	}

	if cfError.Message != "DNS record not found" {
		t.Errorf("Expected error message 'DNS record not found', got %s", cfError.Message)
	}

	// Test Response struct
	response := Response{
		Success: true,
		Errors:  []Error{cfError},
		Result:  []DNSRecord{record},
	}

	if !response.Success {
		t.Error("Expected Success to be true")
	}

	if len(response.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(response.Errors))
	}

	if len(response.Result) != 1 {
		t.Errorf("Expected 1 DNS record, got %d", len(response.Result))
	}

	// Test CreateRecord struct
	createRecord := CreateRecord{
		Type:    "A",
		Name:    "test.example.com",
		Content: "192.168.1.2",
		TTL:     600,
		Proxied: true,
	}

	if createRecord.Type != "A" {
		t.Errorf("Expected Type 'A', got %s", createRecord.Type)
	}

	if createRecord.Name != "test.example.com" {
		t.Errorf("Expected Name 'test.example.com', got %s", createRecord.Name)
	}

	if createRecord.Content != "192.168.1.2" {
		t.Errorf("Expected Content '192.168.1.2', got %s", createRecord.Content)
	}

	if createRecord.TTL != 600 {
		t.Errorf("Expected TTL 600, got %d", createRecord.TTL)
	}

	if createRecord.Proxied != true {
		t.Errorf("Expected Proxied true, got %t", createRecord.Proxied)
	}

	// Test UpdateRecord struct
	updateRecord := UpdateRecord{
		Type:    "A",
		Name:    "update.example.com",
		Content: "192.168.1.3",
		TTL:     300,
		Proxied: false,
	}

	if updateRecord.Type != "A" {
		t.Errorf("Expected Type 'A', got %s", updateRecord.Type)
	}

	if updateRecord.Name != "update.example.com" {
		t.Errorf("Expected Name 'update.example.com', got %s", updateRecord.Name)
	}

	if updateRecord.Content != "192.168.1.3" {
		t.Errorf("Expected Content '192.168.1.3', got %s", updateRecord.Content)
	}

	if updateRecord.TTL != 300 {
		t.Errorf("Expected TTL 300, got %d", updateRecord.TTL)
	}

	if updateRecord.Proxied != false {
		t.Errorf("Expected Proxied false, got %t", updateRecord.Proxied)
	}
}
