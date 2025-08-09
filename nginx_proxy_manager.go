package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
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

func (c *NPMClient) SyncDomainsToNPM(domains []DomainConfig) error {
	log.Printf("DEBUG: Syncing %d domains to NPM", len(domains))

	// Get existing proxy hosts
	existingHosts, err := c.GetProxyHosts()
	if err != nil {
		return fmt.Errorf("failed to get existing proxy hosts: %w", err)
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
		domain := domainConfig.Domain

		// Convert port string to int
		var port int
		if _, err := fmt.Sscanf(domainConfig.Port, "%d", &port); err != nil {
			log.Printf("ERROR: Invalid port '%s' for domain %s: %v", domainConfig.Port, domain, err)
			continue
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

			if needsUpdate {
				log.Printf("DEBUG: Updating existing proxy host for domain %s", domain)
				updatedHost := ProxyHost{
					ID:                    existingHost.ID,
					DomainNames:           []string{domain},
					ForwardHost:           domainConfig.Address,
					ForwardPort:           port,
					ForwardScheme:         "http", // Default to http
					AccessListID:          int(existingHost.AccessListID),
					CertificateID:         int(existingHost.CertificateID),
					SSLForced:             int(existingHost.SSLForced),
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
			log.Printf("DEBUG: Creating new proxy host for domain %s", domain)
			newHost := ProxyHost{
				DomainNames:           []string{domain},
				ForwardHost:           domainConfig.Address,
				ForwardPort:           port,
				ForwardScheme:         "http", // Default to http
				AccessListID:          0,
				CertificateID:         0,
				SSLForced:             0,
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
