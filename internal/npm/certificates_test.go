package npm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGetCertificates(t *testing.T) {
	// Mock server for certificates endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/nginx/certificates" {
			t.Errorf("Expected path /api/nginx/certificates, got %s", r.URL.Path)
		}

		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		// Check for authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-token" {
			t.Errorf("Expected Authorization header 'Bearer test-token', got %s", authHeader)
		}

		// Return mock certificates
		certificates := []CertificateResponse{
			{
				ID:          1,
				Provider:    "letsencrypt",
				NiceName:    "Certificate for example.com",
				DomainNames: []string{"example.com"},
				ExpiresOn:   "2024-12-31T23:59:59Z",
				Meta: CertificateMeta{
					LetsencryptEmail: "test@example.com",
					LetsencryptAgree: true,
					DNSChallenge:     false,
				},
			},
			{
				ID:          2,
				Provider:    "letsencrypt",
				NiceName:    "Wildcard example.com",
				DomainNames: []string{"*.example.com", "example.com"},
				ExpiresOn:   "2024-12-31T23:59:59Z",
				Meta: CertificateMeta{
					LetsencryptEmail:       "test@example.com",
					LetsencryptAgree:       true,
					DNSChallenge:           true,
					DNSProvider:            "cloudflare",
					DNSProviderCredentials: "dns_cloudflare_api_token = test-token",
					PropagationSeconds:     120,
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(certificates)
	}))
	defer server.Close()

	client := &Client{
		BaseURL:     server.URL,
		BearerToken: "test-token",
		TokenExpiry: time.Now().Add(1 * time.Hour),
		httpClient:  &http.Client{},
	}

	certificates, err := client.GetCertificates()
	if err != nil {
		t.Fatalf("GetCertificates() failed: %v", err)
	}

	if len(certificates) != 2 {
		t.Errorf("Expected 2 certificates, got %d", len(certificates))
	}

	// Check first certificate
	if certificates[0].ID != 1 {
		t.Errorf("Expected first certificate ID 1, got %d", certificates[0].ID)
	}

	if certificates[0].NiceName != "Certificate for example.com" {
		t.Errorf("Expected first certificate name 'Certificate for example.com', got %s", certificates[0].NiceName)
	}

	if len(certificates[0].DomainNames) != 1 || certificates[0].DomainNames[0] != "example.com" {
		t.Errorf("Expected first certificate domains ['example.com'], got %v", certificates[0].DomainNames)
	}

	// Check second certificate (wildcard)
	if certificates[1].ID != 2 {
		t.Errorf("Expected second certificate ID 2, got %d", certificates[1].ID)
	}

	if certificates[1].NiceName != "Wildcard example.com" {
		t.Errorf("Expected second certificate name 'Wildcard example.com', got %s", certificates[1].NiceName)
	}

	expectedDomains := []string{"*.example.com", "example.com"}
	if len(certificates[1].DomainNames) != 2 {
		t.Errorf("Expected second certificate to have 2 domains, got %d", len(certificates[1].DomainNames))
	}

	for i, expected := range expectedDomains {
		if certificates[1].DomainNames[i] != expected {
			t.Errorf("Expected domain %s at index %d, got %s", expected, i, certificates[1].DomainNames[i])
		}
	}

	// Check DNS challenge settings
	if !certificates[1].Meta.DNSChallenge {
		t.Error("Expected second certificate to use DNS challenge")
	}

	if certificates[1].Meta.DNSProvider != "cloudflare" {
		t.Errorf("Expected DNS provider 'cloudflare', got %s", certificates[1].Meta.DNSProvider)
	}
}

func TestCreateCloudflareWildcardCertificate(t *testing.T) {
	// Mock server for certificate creation endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/nginx/certificates" {
			t.Errorf("Expected path /api/nginx/certificates, got %s", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		// Check request body
		var cert Certificate
		if err := json.NewDecoder(r.Body).Decode(&cert); err != nil {
			t.Errorf("Failed to decode certificate request: %v", err)
		}

		// Validate the certificate request
		if cert.Provider != "letsencrypt" {
			t.Errorf("Expected provider 'letsencrypt', got %s", cert.Provider)
		}

		if cert.NiceName != "Wildcard example.com" {
			t.Errorf("Expected nice name 'Wildcard example.com', got %s", cert.NiceName)
		}

		expectedDomains := []string{"*.example.com", "example.com"}
		if len(cert.DomainNames) != 2 {
			t.Errorf("Expected 2 domains, got %d", len(cert.DomainNames))
		}

		for i, expected := range expectedDomains {
			if cert.DomainNames[i] != expected {
				t.Errorf("Expected domain %s at index %d, got %s", expected, i, cert.DomainNames[i])
			}
		}

		// Check DNS challenge settings
		if !cert.Meta.DNSChallenge {
			t.Error("Expected DNS challenge to be enabled")
		}

		if cert.Meta.DNSProvider != "cloudflare" {
			t.Errorf("Expected DNS provider 'cloudflare', got %s", cert.Meta.DNSProvider)
		}

		if !strings.Contains(cert.Meta.DNSProviderCredentials, "test-cloudflare-token") {
			t.Errorf("Expected credentials to contain token, got %s", cert.Meta.DNSProviderCredentials)
		}

		if cert.Meta.PropagationSeconds != 120 {
			t.Errorf("Expected propagation seconds 120, got %d", cert.Meta.PropagationSeconds)
		}

		// Return mock response
		response := CertificateResponse{
			ID:          10,
			Provider:    cert.Provider,
			NiceName:    cert.NiceName,
			DomainNames: cert.DomainNames,
			ExpiresOn:   "2024-12-31T23:59:59Z",
			Meta:        cert.Meta,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &Client{
		BaseURL:     server.URL,
		BearerToken: "test-token",
		TokenExpiry: time.Now().Add(1 * time.Hour),
		httpClient:  &http.Client{},
	}

	cert, err := client.CreateCloudflareWildcardCertificate("example.com", "test@example.com", "test-cloudflare-token")
	if err != nil {
		t.Fatalf("CreateCloudflareWildcardCertificate() failed: %v", err)
	}

	if cert.ID != 10 {
		t.Errorf("Expected certificate ID 10, got %d", cert.ID)
	}

	if cert.NiceName != "Wildcard example.com" {
		t.Errorf("Expected certificate name 'Wildcard example.com', got %s", cert.NiceName)
	}

	expectedDomains := []string{"*.example.com", "example.com"}
	if len(cert.DomainNames) != 2 {
		t.Errorf("Expected 2 domains, got %d", len(cert.DomainNames))
	}

	for i, expected := range expectedDomains {
		if cert.DomainNames[i] != expected {
			t.Errorf("Expected domain %s at index %d, got %s", expected, i, cert.DomainNames[i])
		}
	}
}

func TestFindCertificateForDomain(t *testing.T) {
	// Mock server that returns existing certificates
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/nginx/certificates" && r.Method == "GET" {
			// Return existing certificates
			certificates := []CertificateResponse{
				{
					ID:          1,
					DomainNames: []string{"exact.example.com"},
				},
				{
					ID:          2,
					DomainNames: []string{"*.example.com", "example.com"},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(certificates)
			return
		}

		// Handle other requests (shouldn't happen in this test)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := &Client{
		BaseURL:     server.URL,
		BearerToken: "test-token",
		TokenExpiry: time.Now().Add(1 * time.Hour),
		httpClient:  &http.Client{},
	}

	// Test exact match
	certID, err := client.FindCertificateForDomain("exact.example.com", false, "", "")
	if err != nil {
		t.Fatalf("FindCertificateForDomain() failed for exact match: %v", err)
	}

	if certID != 1 {
		t.Errorf("Expected certificate ID 1 for exact match, got %d", certID)
	}

	// Test wildcard match
	certID, err = client.FindCertificateForDomain("api.example.com", false, "", "")
	if err != nil {
		t.Fatalf("FindCertificateForDomain() failed for wildcard match: %v", err)
	}

	if certID != 2 {
		t.Errorf("Expected certificate ID 2 for wildcard match, got %d", certID)
	}
}

func TestCreateWildcardCertificatesForDomains(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/nginx/certificates" && r.Method == "GET" {
			// Return no existing certificates
			certificates := []CertificateResponse{}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(certificates)
			return
		}

		if r.URL.Path == "/api/nginx/certificates" && r.Method == "POST" {
			requestCount++

			// Return success response
			response := CertificateResponse{
				ID:          requestCount,
				Provider:    "letsencrypt",
				NiceName:    "Wildcard certificate",
				DomainNames: []string{"*.example.com", "example.com"},
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(response)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := &Client{
		BaseURL:     server.URL,
		BearerToken: "test-token",
		TokenExpiry: time.Now().Add(1 * time.Hour),
		httpClient:  &http.Client{},
	}

	domains := []string{"example.com", "test.com"}
	err := client.CreateWildcardCertificatesForDomains(domains, "test@example.com", "test-token")

	if err != nil {
		t.Fatalf("CreateWildcardCertificatesForDomains() failed: %v", err)
	}

	// Should have made 2 POST requests (one for each domain)
	if requestCount != 2 {
		t.Errorf("Expected 2 certificate creation requests, got %d", requestCount)
	}
}

func TestCreateCertificateForDomain(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/nginx/certificates" {
			t.Errorf("Expected path /api/nginx/certificates, got %s", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		// Check request body
		var cert Certificate
		if err := json.NewDecoder(r.Body).Decode(&cert); err != nil {
			t.Errorf("Failed to decode certificate request: %v", err)
		}

		// Validate the certificate request
		if cert.Provider != "letsencrypt" {
			t.Errorf("Expected provider 'letsencrypt', got %s", cert.Provider)
		}

		if cert.NiceName != "Certificate for test.example.com" {
			t.Errorf("Expected nice name 'Certificate for test.example.com', got %s", cert.NiceName)
		}

		if len(cert.DomainNames) != 1 || cert.DomainNames[0] != "test.example.com" {
			t.Errorf("Expected domain names ['test.example.com'], got %v", cert.DomainNames)
		}

		// Return mock response
		response := CertificateResponse{
			ID:          5,
			Provider:    cert.Provider,
			NiceName:    cert.NiceName,
			DomainNames: cert.DomainNames,
			ExpiresOn:   "2024-12-31T23:59:59Z",
			Meta:        cert.Meta,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &Client{
		BaseURL:     server.URL,
		BearerToken: "test-token",
		TokenExpiry: time.Now().Add(1 * time.Hour),
		httpClient:  &http.Client{},
	}

	certID, err := client.CreateCertificateForDomain("test.example.com", "test@example.com", "")
	if err != nil {
		t.Fatalf("CreateCertificateForDomain() failed: %v", err)
	}

	if certID != 5 {
		t.Errorf("Expected certificate ID 5, got %d", certID)
	}
}
