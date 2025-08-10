package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

type CloudflareClient struct {
	APIToken string
	ZoneID   string
	Enabled  bool
	Domains  []string
	client   *http.Client
}

type CloudflareDNSRecord struct {
	ID      string `json:"id,omitempty"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
}

type CloudflareResponse struct {
	Success bool                  `json:"success"`
	Errors  []CloudflareError     `json:"errors"`
	Result  []CloudflareDNSRecord `json:"result"`
}

type CloudflareSingleResponse struct {
	Success bool                `json:"success"`
	Errors  []CloudflareError   `json:"errors"`
	Result  CloudflareDNSRecord `json:"result"`
}

type CloudflareError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type CloudflareCreateRecord struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
}

type CloudflareUpdateRecord struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
}

func NewCloudflareClient(config *Config) *CloudflareClient {
	if !config.CloudflareEnabled {
		return &CloudflareClient{Enabled: false}
	}

	return &CloudflareClient{
		APIToken: config.CloudflareToken,
		ZoneID:   config.CloudflareZoneID,
		Enabled:  config.CloudflareEnabled,
		Domains:  config.CloudflareDomains,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *CloudflareClient) GetCurrentIP() (string, error) {
	// Try multiple IP detection services for reliability
	services := []string{
		"https://ipv4.icanhazip.com",
		"https://api.ipify.org",
		"https://checkip.amazonaws.com",
	}

	for _, service := range services {
		resp, err := c.client.Get(service)
		if err != nil {
			log.Printf("DEBUG: Failed to get IP from %s: %v", service, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Printf("DEBUG: Failed to read response from %s: %v", service, err)
				continue
			}

			ip := strings.TrimSpace(string(body))
			// Validate IP format
			if net.ParseIP(ip) != nil {
				log.Printf("DEBUG: Detected current IP: %s (from %s)", ip, service)
				return ip, nil
			}
			log.Printf("DEBUG: Invalid IP format from %s: %s", service, ip)
		}
	}

	return "", fmt.Errorf("failed to detect current IP address from any service")
}

func (c *CloudflareClient) makeRequest(method, endpoint string, body io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records%s", c.ZoneID, endpoint)

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIToken))
	req.Header.Set("Content-Type", "application/json")

	return c.client.Do(req)
}

func (c *CloudflareClient) GetDNSRecords(domain string) ([]CloudflareDNSRecord, error) {
	endpoint := fmt.Sprintf("?name=%s&type=A", domain)

	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get DNS records: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var cfResp CloudflareResponse
	if err := json.Unmarshal(body, &cfResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !cfResp.Success {
		return nil, fmt.Errorf("cloudflare API error: %v", cfResp.Errors)
	}

	return cfResp.Result, nil
}

func (c *CloudflareClient) CreateDNSRecord(domain, ip string) error {
	record := CloudflareCreateRecord{
		Type:    "A",
		Name:    domain,
		Content: ip,
		TTL:     300, // 5 minutes
		Proxied: false,
	}

	jsonData, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	resp, err := c.makeRequest("POST", "", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create DNS record: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var cfResp CloudflareSingleResponse
	if err := json.Unmarshal(body, &cfResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !cfResp.Success {
		return fmt.Errorf("cloudflare API error: %v", cfResp.Errors)
	}

	log.Printf("DEBUG: Created DNS record for %s -> %s", domain, ip)
	return nil
}

func (c *CloudflareClient) UpdateDNSRecord(recordID, domain, ip string) error {
	record := CloudflareUpdateRecord{
		Type:    "A",
		Name:    domain,
		Content: ip,
		TTL:     300, // 5 minutes
		Proxied: false,
	}

	jsonData, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	endpoint := fmt.Sprintf("/%s", recordID)
	resp, err := c.makeRequest("PUT", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to update DNS record: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var cfResp CloudflareSingleResponse
	if err := json.Unmarshal(body, &cfResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !cfResp.Success {
		return fmt.Errorf("cloudflare API error: %v", cfResp.Errors)
	}

	log.Printf("DEBUG: Updated DNS record for %s -> %s", domain, ip)
	return nil
}

func (c *CloudflareClient) UpdateDNSRecords() error {
	if !c.Enabled {
		return nil
	}

	if len(c.Domains) == 0 {
		log.Printf("DEBUG: No Cloudflare domains configured, skipping DNS update")
		return nil
	}

	log.Printf("DEBUG: Starting Cloudflare DNS update for %d domains", len(c.Domains))

	// Get current IP
	currentIP, err := c.GetCurrentIP()
	if err != nil {
		return fmt.Errorf("failed to get current IP: %w", err)
	}

	// Process each domain
	for _, domain := range c.Domains {
		domain = strings.TrimSpace(domain)
		if domain == "" {
			continue
		}

		log.Printf("DEBUG: Processing domain: %s", domain)

		// Get existing DNS records
		records, err := c.GetDNSRecords(domain)
		if err != nil {
			log.Printf("ERROR: Failed to get DNS records for %s: %v", domain, err)
			continue
		}

		if len(records) == 0 {
			// Create new record
			log.Printf("DEBUG: Creating new DNS record for %s", domain)
			if err := c.CreateDNSRecord(domain, currentIP); err != nil {
				log.Printf("ERROR: Failed to create DNS record for %s: %v", domain, err)
			} else {
				log.Printf("INFO: Created DNS record: %s -> %s", domain, currentIP)
			}
		} else {
			// Update existing record(s)
			for _, record := range records {
				if record.Content != currentIP {
					log.Printf("DEBUG: Updating DNS record for %s: %s -> %s", domain, record.Content, currentIP)
					if err := c.UpdateDNSRecord(record.ID, domain, currentIP); err != nil {
						log.Printf("ERROR: Failed to update DNS record for %s: %v", domain, err)
					} else {
						log.Printf("INFO: Updated DNS record: %s -> %s", domain, currentIP)
					}
				} else {
					log.Printf("DEBUG: DNS record for %s is already up to date (%s)", domain, currentIP)
				}
			}
		}
	}

	log.Printf("DEBUG: Completed Cloudflare DNS update")
	return nil
}
