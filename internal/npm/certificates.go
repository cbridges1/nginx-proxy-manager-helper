package npm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

func (c *Client) GetCertificates() ([]CertificateResponse, error) {
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

func (c *Client) CreateCloudflareWildcardCertificate(domain, email, cloudflareToken string) (*CertificateResponse, error) {
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

func (c *Client) CreateWildcardCertificatesForDomains(domains []string, email, cloudflareToken string) error {
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

func (c *Client) FindCertificateForDomain(domain string, createWildcardCerts bool, letsencryptEmail, cloudflareToken string) (int, error) {
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

func (c *Client) CreateCertificateForDomain(domain, email, cloudflareToken string) (int, error) {
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
