package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// BoolInt is a custom type that can unmarshal both boolean and integer values
type BoolInt int

func (bi *BoolInt) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as boolean first
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		if b {
			*bi = 1
		} else {
			*bi = 0
		}
		return nil
	}

	// Try to unmarshal as integer
	var i int
	if err := json.Unmarshal(data, &i); err == nil {
		*bi = BoolInt(i)
		return nil
	}

	// Try to unmarshal as string and convert
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		if val, err := strconv.Atoi(s); err == nil {
			*bi = BoolInt(val)
			return nil
		}
		// Handle string boolean values
		if s == "true" || s == "1" {
			*bi = 1
			return nil
		}
		if s == "false" || s == "0" {
			*bi = 0
			return nil
		}
	}

	return fmt.Errorf("cannot unmarshal %s into BoolInt", data)
}

func (bi BoolInt) MarshalJSON() ([]byte, error) {
	return json.Marshal(int(bi))
}

type NPMClient struct {
	BaseURL     string
	Email       string
	Password    string
	BearerToken string
	TokenExpiry time.Time
	httpClient  *http.Client
}

type LoginRequest struct {
	Identity string `json:"identity"`
	Secret   string `json:"secret"`
}

type LoginResponse struct {
	Token   string `json:"token"`
	Expires string `json:"expires"`
}

type ProxyHost struct {
	ID                    int      `json:"id,omitempty"`
	DomainNames           []string `json:"domain_names"`
	ForwardHost           string   `json:"forward_host"`
	ForwardPort           int      `json:"forward_port"`
	ForwardScheme         string   `json:"forward_scheme"`
	AccessListID          int      `json:"access_list_id"`
	CertificateID         int      `json:"certificate_id"`
	SSLForced             int      `json:"ssl_forced"`
	CachingEnabled        int      `json:"caching_enabled"`
	BlockExploits         int      `json:"block_exploits"`
	AdvancedConfig        string   `json:"advanced_config"`
	AllowWebsocketUpgrade int      `json:"allow_websocket_upgrade"`
	HTTP2Support          int      `json:"http2_support"`
	HSTSEnabled           int      `json:"hsts_enabled"`
	HSTSSubdomains        int      `json:"hsts_subdomains"`
	Enabled               int      `json:"enabled"`
}

type ProxyHostResponse struct {
	ID                    int      `json:"id"`
	CreatedOn             string   `json:"created_on"`
	ModifiedOn            string   `json:"modified_on"`
	DomainNames           []string `json:"domain_names"`
	ForwardHost           string   `json:"forward_host"`
	ForwardPort           int      `json:"forward_port"`
	ForwardScheme         string   `json:"forward_scheme"`
	AccessListID          BoolInt  `json:"access_list_id"`
	CertificateID         BoolInt  `json:"certificate_id"`
	SSLForced             BoolInt  `json:"ssl_forced"`
	CachingEnabled        BoolInt  `json:"caching_enabled"`
	BlockExploits         BoolInt  `json:"block_exploits"`
	AdvancedConfig        string   `json:"advanced_config"`
	AllowWebsocketUpgrade BoolInt  `json:"allow_websocket_upgrade"`
	HTTP2Support          BoolInt  `json:"http2_support"`
	HSTSEnabled           BoolInt  `json:"hsts_enabled"`
	HSTSSubdomains        BoolInt  `json:"hsts_subdomains"`
	Enabled               BoolInt  `json:"enabled"`
}

type Certificate struct {
	ID          int             `json:"id,omitempty"`
	CreatedOn   string          `json:"created_on,omitempty"`
	ModifiedOn  string          `json:"modified_on,omitempty"`
	OwnerUserID int             `json:"owner_user_id,omitempty"`
	Provider    string          `json:"provider"`
	NiceName    string          `json:"nice_name"`
	DomainNames []string        `json:"domain_names"`
	ExpiresOn   string          `json:"expires_on,omitempty"`
	Meta        CertificateMeta `json:"meta"`
}

type CertificateMeta struct {
	LetsencryptEmail       string `json:"letsencrypt_email"`
	LetsencryptAgree       bool   `json:"letsencrypt_agree"`
	DNSChallenge           bool   `json:"dns_challenge"`
	DNSProvider            string `json:"dns_provider"`
	DNSProviderCredentials string `json:"dns_provider_credentials"`
	PropagationSeconds     int    `json:"propagation_seconds"`
}

type CertificateResponse struct {
	ID          int             `json:"id"`
	CreatedOn   string          `json:"created_on"`
	ModifiedOn  string          `json:"modified_on"`
	OwnerUserID int             `json:"owner_user_id"`
	Provider    string          `json:"provider"`
	NiceName    string          `json:"nice_name"`
	DomainNames []string        `json:"domain_names"`
	ExpiresOn   string          `json:"expires_on"`
	Meta        CertificateMeta `json:"meta"`
}

type CertificateListResponse struct {
	Success bool                  `json:"success"`
	Result  []CertificateResponse `json:"result"`
}

type CertificateSingleResponse struct {
	Success bool                `json:"success"`
	Result  CertificateResponse `json:"result"`
}

func NewNPMClient(config *Config) *NPMClient {
	return &NPMClient{
		BaseURL:    config.NPMBaseURL,
		Email:      config.NPMEmail,
		Password:   config.NPMPassword,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *NPMClient) Login() error {
	loginReq := LoginRequest{
		Identity: c.Email,
		Secret:   c.Password,
	}

	jsonData, err := json.Marshal(loginReq)
	if err != nil {
		return fmt.Errorf("failed to marshal login request: %w", err)
	}

	resp, err := c.httpClient.Post(
		fmt.Sprintf("%s/api/tokens", c.BaseURL),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to make login request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return fmt.Errorf("failed to decode login response: %w", err)
	}

	c.BearerToken = loginResp.Token

	// Parse expiry time (assuming ISO format)
	if loginResp.Expires != "" {
		if expiry, err := time.Parse(time.RFC3339, loginResp.Expires); err == nil {
			c.TokenExpiry = expiry
		} else {
			// If parsing fails, assume token expires in 1 hour
			c.TokenExpiry = time.Now().Add(1 * time.Hour)
		}
	} else {
		c.TokenExpiry = time.Now().Add(1 * time.Hour)
	}

	log.Printf("DEBUG: NPM login successful, token expires at %v", c.TokenExpiry)
	return nil
}

func (c *NPMClient) ensureAuthenticated() error {
	// Check if token is expired or will expire soon (5 minutes buffer)
	if c.BearerToken == "" || time.Now().Add(5*time.Minute).After(c.TokenExpiry) {
		log.Printf("DEBUG: NPM token expired or missing, re-authenticating")
		return c.Login()
	}
	return nil
}

func (c *NPMClient) makeAuthenticatedRequest(method, endpoint string, body io.Reader) (*http.Response, error) {
	if err := c.ensureAuthenticated(); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", c.BaseURL, endpoint), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.BearerToken))
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

func (c *NPMClient) GetProxyHosts() ([]ProxyHostResponse, error) {
	resp, err := c.makeAuthenticatedRequest("GET", "/api/nginx/proxy-hosts", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get proxy hosts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get proxy hosts failed with status %d: %s", resp.StatusCode, string(body))
	}

	var hosts []ProxyHostResponse
	if err := json.NewDecoder(resp.Body).Decode(&hosts); err != nil {
		return nil, fmt.Errorf("failed to decode proxy hosts response: %w", err)
	}

	return hosts, nil
}

func (c *NPMClient) CreateProxyHost(host ProxyHost) (*ProxyHostResponse, error) {
	jsonData, err := json.Marshal(host)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal proxy host: %w", err)
	}

	log.Printf("DEBUG: Creating proxy host: %s", string(jsonData))

	resp, err := c.makeAuthenticatedRequest("POST", "/api/nginx/proxy-hosts", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy host: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("create proxy host failed with status %d: %s", resp.StatusCode, string(body))
	}

	var createdHost ProxyHostResponse
	if err := json.Unmarshal(body, &createdHost); err != nil {
		return nil, fmt.Errorf("failed to decode created proxy host response: %w", err)
	}

	log.Printf("DEBUG: Created proxy host with ID %d for domains %v", createdHost.ID, createdHost.DomainNames)
	return &createdHost, nil
}

func (c *NPMClient) UpdateProxyHost(hostID int, host ProxyHost) (*ProxyHostResponse, error) {
	jsonData, err := json.Marshal(host)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal proxy host: %w", err)
	}

	log.Printf("DEBUG: Updating proxy host ID %d: %s", hostID, string(jsonData))

	resp, err := c.makeAuthenticatedRequest("PUT", fmt.Sprintf("/api/nginx/proxy-hosts/%d", hostID), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to update proxy host: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update proxy host failed with status %d: %s", resp.StatusCode, string(body))
	}

	var updatedHost ProxyHostResponse
	if err := json.Unmarshal(body, &updatedHost); err != nil {
		return nil, fmt.Errorf("failed to decode updated proxy host response: %w", err)
	}

	log.Printf("DEBUG: Updated proxy host ID %d for domains %v", updatedHost.ID, updatedHost.DomainNames)
	return &updatedHost, nil
}

func (c *NPMClient) DeleteProxyHost(hostID int) error {
	resp, err := c.makeAuthenticatedRequest("DELETE", fmt.Sprintf("/api/nginx/proxy-hosts/%d", hostID), nil)
	if err != nil {
		return fmt.Errorf("failed to delete proxy host: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete proxy host failed with status %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("DEBUG: Deleted proxy host ID %d", hostID)
	return nil
}

// sanitizeDomainName removes protocol and invalid characters from domain names
func sanitizeDomainName(domain string) string {
	// Remove common protocols
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "//")

	// Parse as URL to extract just the hostname if it contains paths/query params
	if parsedURL, err := url.Parse("http://" + domain); err == nil && parsedURL.Host != "" {
		domain = parsedURL.Host
	}

	// Remove port if present (NPM handles ports separately)
	if colonIndex := strings.LastIndex(domain, ":"); colonIndex != -1 {
		// Only remove if it looks like a port (numeric after colon)
		if portPart := domain[colonIndex+1:]; portPart != "" {
			if _, err := strconv.Atoi(portPart); err == nil {
				domain = domain[:colonIndex]
			}
		}
	}

	// Trim whitespace and convert to lowercase
	domain = strings.ToLower(strings.TrimSpace(domain))

	// Remove any remaining invalid characters according to NPM's pattern
	// Pattern: ^[^&| @!#%^();:/\\}{=+?<>,~`'"]+$
	// This means: no &, |, space, @, !, #, %, ^, (, ), ;, :, /, \, }, {, =, +, ?, <, >, ,, ~, `, ', "
	invalidChars := "&| @!#%^();:/\\}{=+?<>,~`'\""
	for _, char := range invalidChars {
		domain = strings.ReplaceAll(domain, string(char), "")
	}

	return domain
}

func (c *NPMClient) SyncDomainsToNPM(domains []DomainConfig, createWildcardCerts bool, letsencryptEmail, cloudflareToken string) error {
	log.Printf("DEBUG: Syncing %d domains to NPM", len(domains))

	// Get existing proxy hosts
	existingHosts, err := c.GetProxyHosts()
	if err != nil {
		//return fmt.Errorf("failed to get existing proxy hosts: %w", err)
	}

	log.Printf("DEBUG: Found %d existing proxy hosts", len(existingHosts))

	// Create a map of existing hosts by domain name for quick lookup
	existingHostMap := make(map[string]ProxyHostResponse)
	for _, host := range existingHosts {
		for _, domain := range host.DomainNames {
			existingHostMap[domain] = host
		}
	}

	// Process each domain
	for _, domainConfig := range domains {
		originalDomain := domainConfig.Domain
		domain := sanitizeDomainName(originalDomain)

		log.Printf("DEBUG: Processing domain: %s (original: %s, createWildcardCerts: %v)", domain, originalDomain, createWildcardCerts)

		if domain == "" {
			log.Printf("ERROR: Domain '%s' became empty after sanitization, skipping", originalDomain)
			continue
		}

		if domain != originalDomain {
			log.Printf("DEBUG: Sanitized domain '%s' to '%s'", originalDomain, domain)
		}

		// Convert port string to int
		var port int
		if _, err := fmt.Sscanf(domainConfig.Port, "%d", &port); err != nil {
			log.Printf("ERROR: Invalid port '%s' for domain %s: %v", domainConfig.Port, domain, err)
			continue
		}

		// Check if domain contains "https" to determine if HTTPS should be used
		useHTTPS := strings.Contains(strings.ToLower(originalDomain), "https")
		var certificateID int

		// If HTTPS is requested, try to find or create a certificate
		if useHTTPS {
			if letsencryptEmail != "" {
				certID, err := c.FindCertificateForDomain(domain, createWildcardCerts, letsencryptEmail, cloudflareToken)
				if err != nil {
					log.Printf("ERROR: Failed to find/create certificate for %s: %v", domain, err)
					// Continue without HTTPS if certificate creation fails
					useHTTPS = false
					certificateID = 0
				} else {
					certificateID = certID
					log.Printf("DEBUG: Using certificate ID %d for HTTPS domain %s", certificateID, domain)
				}
			} else {
				log.Printf("WARNING: HTTPS requested for %s but no LETSENCRYPT_EMAIL configured, creating HTTP proxy instead", domain)
				useHTTPS = false
				certificateID = 0
			}
		}

		// Check if this domain already exists
		if existingHost, exists := existingHostMap[domain]; exists {
			// Update existing host if needed
			needsUpdate := false
			if existingHost.ForwardHost != domainConfig.Address {
				needsUpdate = true
			}
			if existingHost.ForwardPort != port {
				needsUpdate = true
			}
			if int(existingHost.CertificateID) != certificateID {
				needsUpdate = true
			}

			if needsUpdate {
				log.Printf("DEBUG: Updating existing proxy host for domain %s (HTTPS: %v)", domain, useHTTPS)

				forwardScheme := "http"
				sslForced := 0
				if useHTTPS {
					//forwardScheme = "https"
					sslForced = 1
				}

				updatedHost := ProxyHost{
					ID:                    existingHost.ID,
					DomainNames:           []string{domain},
					ForwardHost:           domainConfig.Address,
					ForwardPort:           port,
					ForwardScheme:         forwardScheme,
					AccessListID:          int(existingHost.AccessListID),
					CertificateID:         certificateID,
					SSLForced:             sslForced,
					CachingEnabled:        int(existingHost.CachingEnabled),
					BlockExploits:         int(existingHost.BlockExploits),
					AdvancedConfig:        existingHost.AdvancedConfig,
					AllowWebsocketUpgrade: int(existingHost.AllowWebsocketUpgrade),
					HTTP2Support:          int(existingHost.HTTP2Support),
					HSTSEnabled:           int(existingHost.HSTSEnabled),
					HSTSSubdomains:        int(existingHost.HSTSSubdomains),
					Enabled:               1,
				}

				if _, err := c.UpdateProxyHost(existingHost.ID, updatedHost); err != nil {
					log.Printf("ERROR: Failed to update proxy host for domain %s: %v", domain, err)
					continue
				}
			} else {
				log.Printf("DEBUG: Proxy host for domain %s is already up to date", domain)
			}
		} else {
			// Create new proxy host
			log.Printf("DEBUG: Creating new proxy host for domain %s (HTTPS: %v, certificateID: %d)", domain, useHTTPS, certificateID)

			forwardScheme := "http"
			sslForced := 0
			if useHTTPS {
				//forwardScheme = "https"
				sslForced = 1
			}

			newHost := ProxyHost{
				DomainNames:           []string{domain},
				ForwardHost:           domainConfig.Address,
				ForwardPort:           port,
				ForwardScheme:         forwardScheme,
				AccessListID:          0,
				CertificateID:         certificateID,
				SSLForced:             sslForced,
				CachingEnabled:        0,
				BlockExploits:         1, // Enable by default for security
				AdvancedConfig:        "",
				AllowWebsocketUpgrade: 1, // Enable by default
				HTTP2Support:          1, // Enable by default
				HSTSEnabled:           0,
				HSTSSubdomains:        0,
				Enabled:               1,
			}

			if _, err := c.CreateProxyHost(newHost); err != nil {
				log.Printf("ERROR: Failed to create proxy host for domain %s: %v", domain, err)
				continue
			}
		}
	}

	log.Printf("DEBUG: Completed syncing domains to NPM")
	return nil
}

func (c *NPMClient) RemoveProxyHostsByDomains(domains []string) error {
	if len(domains) == 0 {
		return nil
	}

	log.Printf("DEBUG: Removing proxy hosts for %d domains from NPM", len(domains))

	// Sanitize domain names for consistency
	sanitizedDomains := make(map[string]bool)
	for _, domain := range domains {
		sanitized := sanitizeDomainName(domain)
		if sanitized != "" {
			sanitizedDomains[sanitized] = true
		}
	}

	// Get existing proxy hosts
	existingHosts, err := c.GetProxyHosts()
	if err != nil {
		return fmt.Errorf("failed to get existing proxy hosts for removal: %w", err)
	}

	// Find hosts that match the domains to remove
	var hostsToRemove []ProxyHostResponse
	for _, host := range existingHosts {
		for _, hostDomain := range host.DomainNames {
			sanitizedHostDomain := sanitizeDomainName(hostDomain)
			if sanitizedDomains[sanitizedHostDomain] {
				hostsToRemove = append(hostsToRemove, host)
				log.Printf("DEBUG: Found proxy host ID %d with domain '%s' to remove", host.ID, hostDomain)
				break // Don't add the same host multiple times if it has multiple matching domains
			}
		}
	}

	// Remove the identified hosts
	for _, host := range hostsToRemove {
		log.Printf("DEBUG: Removing proxy host ID %d with domains %v", host.ID, host.DomainNames)
		if err := c.DeleteProxyHost(host.ID); err != nil {
			log.Printf("ERROR: Failed to remove proxy host ID %d: %v", host.ID, err)
			// Continue with other removals even if one fails
		} else {
			log.Printf("DEBUG: Successfully removed proxy host ID %d", host.ID)
		}
	}

	log.Printf("DEBUG: Completed removing proxy hosts from NPM")
	return nil
}

func (c *NPMClient) GetCertificates() ([]CertificateResponse, error) {
	resp, err := c.makeAuthenticatedRequest("GET", "/api/nginx/certificates", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get certificates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get certificates failed with status %d: %s", resp.StatusCode, string(body))
	}

	var certificates []CertificateResponse
	if err := json.NewDecoder(resp.Body).Decode(&certificates); err != nil {
		return nil, fmt.Errorf("failed to decode certificates response: %w", err)
	}

	return certificates, nil
}

func (c *NPMClient) CreateCloudflareWildcardCertificate(domain, email, cloudflareToken string) (*CertificateResponse, error) {
	// Create wildcard domain (e.g., "example.com" -> "*.example.com")
	wildcardDomain := "*." + domain

	// Prepare DNS provider credentials for Cloudflare - try multiple formats
	// Format 1: Simple key=value format
	credentials := fmt.Sprintf("dns_cloudflare_api_token = %s", cloudflareToken)

	certificate := Certificate{
		Provider:    "letsencrypt",
		NiceName:    fmt.Sprintf("Wildcard %s", domain),
		DomainNames: []string{wildcardDomain, domain}, // Include both wildcard and apex domain
		Meta: CertificateMeta{
			LetsencryptEmail:       email,
			LetsencryptAgree:       true,
			DNSChallenge:           true,
			DNSProvider:            "cloudflare",
			DNSProviderCredentials: credentials,
			PropagationSeconds:     120,
		},
	}

	jsonData, err := json.Marshal(certificate)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal certificate: %w", err)
	}

	log.Printf("DEBUG: Creating wildcard certificate for %s", domain)
	log.Printf("DEBUG: Certificate request JSON: %s", string(jsonData))

	resp, err := c.makeAuthenticatedRequest("POST", "/api/nginx/certificates", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	log.Printf("DEBUG: Certificate creation response status: %d", resp.StatusCode)
	log.Printf("DEBUG: Certificate creation response body: %s", string(body))

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("create certificate failed with status %d: %s", resp.StatusCode, string(body))
	}

	var createdCert CertificateResponse
	if err := json.Unmarshal(body, &createdCert); err != nil {
		return nil, fmt.Errorf("failed to decode created certificate response: %w", err)
	}

	log.Printf("DEBUG: Created wildcard certificate with ID %d for domain %s", createdCert.ID, domain)
	return &createdCert, nil
}

func (c *NPMClient) CreateWildcardCertificatesForDomains(domains []string, email, cloudflareToken string) error {
	if len(domains) == 0 {
		return nil
	}

	log.Printf("DEBUG: Creating wildcard certificates for %d domains", len(domains))

	// Get existing certificates
	existingCerts, err := c.GetCertificates()
	if err != nil {
		return fmt.Errorf("failed to get existing certificates: %w", err)
	}

	// Create a map of existing wildcard certificates by domain
	existingWildcards := make(map[string]bool)
	for _, cert := range existingCerts {
		for _, certDomain := range cert.DomainNames {
			if strings.HasPrefix(certDomain, "*.") {
				// Extract base domain from wildcard (*.example.com -> example.com)
				baseDomain := strings.TrimPrefix(certDomain, "*.")
				existingWildcards[baseDomain] = true
			}
		}
	}

	// Process each domain
	for _, domain := range domains {
		domain = strings.TrimSpace(domain)
		if domain == "" {
			continue
		}

		// Skip if wildcard certificate already exists
		if existingWildcards[domain] {
			log.Printf("DEBUG: Wildcard certificate for %s already exists, skipping", domain)
			continue
		}

		// Create wildcard certificate
		log.Printf("DEBUG: Creating wildcard certificate for %s", domain)
		if _, err := c.CreateCloudflareWildcardCertificate(domain, email, cloudflareToken); err != nil {
			log.Printf("ERROR: Failed to create wildcard certificate for %s: %v", domain, err)
			// Continue with other domains even if one fails
		} else {
			log.Printf("INFO: Successfully created wildcard certificate for %s", domain)
		}
	}

	log.Printf("DEBUG: Completed wildcard certificate creation")
	return nil
}

func (c *NPMClient) FindCertificateForDomain(domain string, createWildcardCerts bool, letsencryptEmail, cloudflareToken string) (int, error) {
	// Get existing certificates
	certificates, err := c.GetCertificates()
	if err != nil {
		return 0, fmt.Errorf("failed to get certificates: %w", err)
	}

	// Look for an exact domain match first
	for _, cert := range certificates {
		for _, certDomain := range cert.DomainNames {
			if certDomain == domain {
				log.Printf("DEBUG: Found exact certificate match for %s (ID: %d)", domain, cert.ID)
				return cert.ID, nil
			}
		}
	}

	// Look for a matching wildcard certificate
	var rootDomain string
	parts := strings.Split(domain, ".")
	if len(parts) >= 2 {
		rootDomain = strings.Join(parts[len(parts)-2:], ".")
	}

	if rootDomain != "" {
		wildcardDomain := "*." + rootDomain
		for _, cert := range certificates {
			for _, certDomain := range cert.DomainNames {
				if certDomain == wildcardDomain {
					log.Printf("DEBUG: Found existing wildcard certificate for %s (wildcard: %s, ID: %d)", domain, wildcardDomain, cert.ID)
					return cert.ID, nil
				}
			}
		}
	}

	// No existing certificate found, create a new one based on createWildcardCerts setting
	if createWildcardCerts && rootDomain != "" && cloudflareToken != "" {
		// Try to create wildcard certificate if enabled and Cloudflare is configured
		log.Printf("DEBUG: Creating new wildcard certificate for %s", rootDomain)
		cert, err := c.CreateCloudflareWildcardCertificate(rootDomain, letsencryptEmail, cloudflareToken)
		if err != nil {
			log.Printf("ERROR: Failed to create wildcard certificate for %s: %v", rootDomain, err)
			// Fall back to individual certificate
		} else {
			log.Printf("DEBUG: Created wildcard certificate for %s (ID: %d)", rootDomain, cert.ID)
			return cert.ID, nil
		}
	}

	// Create individual certificate (either as fallback or when wildcards are disabled)
	log.Printf("DEBUG: Creating new individual certificate for %s", domain)
	return c.CreateCertificateForDomain(domain, letsencryptEmail, cloudflareToken)
}

func (c *NPMClient) CreateCertificateForDomain(domain, email, cloudflareToken string) (int, error) {
	var credentials string
	var dnsChallenge bool
	var dnsProvider string

	// Use Cloudflare DNS challenge if token is provided, otherwise use HTTP-01 challenge
	if cloudflareToken != "" {
		dnsChallenge = true
		dnsProvider = "cloudflare"
		credentials = fmt.Sprintf("dns_cloudflare_api_token = %s", cloudflareToken)
	} else {
		// Use HTTP-01 challenge when no DNS provider is configured
		dnsChallenge = false
		dnsProvider = ""
		credentials = ""
	}

	certificate := Certificate{
		Provider:    "letsencrypt",
		NiceName:    fmt.Sprintf("Certificate for %s", domain),
		DomainNames: []string{domain},
		Meta: CertificateMeta{
			LetsencryptEmail:       email,
			LetsencryptAgree:       true,
			DNSChallenge:           dnsChallenge,
			DNSProvider:            dnsProvider,
			DNSProviderCredentials: credentials,
			PropagationSeconds:     120,
		},
	}

	jsonData, err := json.Marshal(certificate)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal certificate: %w", err)
	}

	log.Printf("DEBUG: Creating certificate for domain %s", domain)
	log.Printf("DEBUG: Certificate request JSON: %s", string(jsonData))

	resp, err := c.makeAuthenticatedRequest("POST", "/api/nginx/certificates", bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, fmt.Errorf("failed to create certificate: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response body: %w", err)
	}

	log.Printf("DEBUG: Certificate creation response status: %d", resp.StatusCode)
	log.Printf("DEBUG: Certificate creation response body: %s", string(body))

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("create certificate failed with status %d: %s", resp.StatusCode, string(body))
	}

	var createdCert CertificateResponse
	if err := json.Unmarshal(body, &createdCert); err != nil {
		return 0, fmt.Errorf("failed to decode created certificate response: %w", err)
	}

	log.Printf("INFO: Created certificate with ID %d for domain %s", createdCert.ID, domain)
	return createdCert.ID, nil
}
