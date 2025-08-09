package main

import (
	"log"
	"strings"
)

type DomainConfig struct {
	Domain  string
	Address string
	Port    string
}

func ExtractNginxDomains(labels map[string]string) []DomainConfig {
	var domains []DomainConfig
	for key, value := range labels {
		if strings.HasPrefix(key, "nginx-domain") {
			log.Printf("DEBUG: Found nginx-domain label: %s=%s", key, value)
			entryList := strings.FieldsFunc(value, func(c rune) bool {
				return c == ',' || c == ';'
			})
			log.Printf("DEBUG: Split into %d entries: %v", len(entryList), entryList)

			for _, entry := range entryList {
				entry = strings.TrimSpace(entry)
				log.Printf("DEBUG: Processing entry: '%s'", entry)
				if entry != "" {
					parts := strings.Split(entry, "~")
					log.Printf("DEBUG: Split entry into %d parts: %v", len(parts), parts)
					if len(parts) == 3 {
						domain := strings.TrimSpace(parts[0])
						address := strings.TrimSpace(parts[1])
						port := strings.TrimSpace(parts[2])
						log.Printf("DEBUG: Parsed - domain='%s', address='%s', port='%s'", domain, address, port)

						if domain != "" && address != "" && port != "" {
							config := DomainConfig{
								Domain:  domain,
								Address: address,
								Port:    port,
							}
							domains = append(domains, config)
							log.Printf("DEBUG: Added domain config: %+v", config)
						} else {
							log.Printf("DEBUG: Skipping empty field - domain='%s', address='%s', port='%s'", domain, address, port)
						}
					} else {
						log.Printf("WARNING: Invalid nginx-domain format '%s', expected 'domain~address~port'", entry)
					}
				} else {
					log.Printf("DEBUG: Skipping empty entry")
				}
			}
		}
	}

	log.Printf("DEBUG: ExtractNginxDomains returning %d domains: %+v", len(domains), domains)
	return domains
}
