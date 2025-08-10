package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func main() {
	ctx := context.Background()

	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Printf("DEBUG: Database connection successful")

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	messages, errs := cli.Events(ctx, types.EventsOptions{})

	// Start reconciliation ticker (every minute)
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// Start Cloudflare DNS update ticker (every 5 minutes)
	dnsTicker := time.NewTicker(5 * time.Minute)
	defer dnsTicker.Stop()

	// Initialize NPM client
	npmClient := NewNPMClient(config)
	log.Printf("DEBUG: NPM client initialized for %s", npmClient.BaseURL)

	// Initialize Cloudflare client
	cloudflareClient := NewCloudflareClient(config)
	if cloudflareClient.Enabled {
		log.Printf("DEBUG: Cloudflare DNS client enabled for %d domains", len(cloudflareClient.Domains))
		// Run initial DNS update
		go func() {
			log.Printf("DEBUG: Running initial Cloudflare DNS update")
			if err := cloudflareClient.UpdateDNSRecords(); err != nil {
				log.Printf("ERROR: Initial Cloudflare DNS update failed: %v", err)
			}
		}()
	} else {
		log.Printf("DEBUG: Cloudflare DNS updates disabled")
	}

	// Sync existing domains from database to NPM on startup
	log.Printf("DEBUG: Syncing existing domains from database to NPM")
	if existingDomains, err := GetAllDomains(db); err != nil {
		log.Printf("ERROR: Failed to get existing domains: %v", err)
	} else if len(existingDomains) > 0 {
		log.Printf("DEBUG: Found %d existing domains to sync", len(existingDomains))
		if err := npmClient.SyncDomainsToNPM(existingDomains); err != nil {
			log.Printf("ERROR: Failed to sync existing domains to NPM: %v", err)
		} else {
			log.Printf("DEBUG: Successfully synced existing domains to NPM")
		}
	}

	// Run initial reconciliation
	go ReconcileContainers(ctx, cli, db, npmClient)

	for {
		select {
		case <-ticker.C:
			// Run reconciliation every minute
			go ReconcileContainers(ctx, cli, db, npmClient)
		case <-dnsTicker.C:
			// Run Cloudflare DNS update every 5 minutes
			if cloudflareClient.Enabled {
				go func() {
					log.Printf("DEBUG: Running scheduled Cloudflare DNS update")
					if err := cloudflareClient.UpdateDNSRecords(); err != nil {
						log.Printf("ERROR: Scheduled Cloudflare DNS update failed: %v", err)
					}
				}()
			}
		case msg := <-messages:
			if msg.Type == "container" {
				containerID := msg.ID

				// Handle container removal/destruction events
				if msg.Action == "destroy" {
					existingDomains, err := GetDomainsForContainer(db, containerID)
					if err != nil {
						log.Printf("ERROR: Failed to get existing domains for container %s: %v", containerID, err)
					} else if len(existingDomains) > 0 {
						log.Printf("DEBUG: Removing %d domains from NPM for container %s", len(existingDomains), containerID)
						if err := npmClient.RemoveProxyHostsByDomains(existingDomains); err != nil {
							log.Printf("ERROR: Failed to remove proxy hosts from NPM for container %s: %v", containerID, err)
						} else {
							log.Printf("DEBUG: Successfully removed proxy hosts from NPM for container %s", containerID)
						}
					}

					if err := RemoveDomain(db, containerID); err != nil {
						log.Printf("Failed to remove domain for container %s: %v", containerID, err)
					}
					fmt.Printf("Container Event: Action=%s, Name=%s, ID=%s (removed from database)\n",
						msg.Action, msg.Actor.Attributes["name"], containerID)
					continue
				}

				// For other events, inspect the container to check for domain label
				containerInfo, err := cli.ContainerInspect(ctx, containerID)
				if err != nil {
					fmt.Printf("Container Event: Action=%s, Name=%s, ID=%s (failed to inspect: %v)\n",
						msg.Action, msg.Actor.Attributes["name"], containerID, err)
					continue
				}

				// Look for nginx-domain labels
				var domains []DomainConfig
				if containerInfo.Config != nil && containerInfo.Config.Labels != nil {
					domains = ExtractNginxDomains(containerInfo.Config.Labels)
				}

				log.Printf("Igloo: %s", domains)
				if len(domains) > 0 {
					// Insert or update domains in database
					log.Printf("Igloo %s", domains)
					if err := InsertOrUpdateDomains(db, containerID, domains); err != nil {
						log.Printf("Failed to store domains for container %s: %v", containerID, err)
					} else {
						// Verify the insert worked
						VerifyInsert(db, containerID)
						ListAllEntries(db)
						// Sync domains to NPM after successful database update
						log.Printf("DEBUG: Syncing domains to NPM for container %s", containerID)
						if err := npmClient.SyncDomainsToNPM(domains); err != nil {
							log.Printf("ERROR: Failed to sync domains to NPM for container %s: %v", containerID, err)
						} else {
							log.Printf("DEBUG: Successfully synced domains to NPM for container %s", containerID)
						}
					}
					fmt.Printf("Container Event: Action=%s, Name=%s, ID=%s, Domain Configs: (stored in database)\n",
						msg.Action, msg.Actor.Attributes["name"], containerID)
					for _, config := range domains {
						fmt.Printf("  - %s~%s~%s\n", config.Domain, config.Address, config.Port)
					}
				} else {
					// Remove from database if no domain labels exist
					if err := RemoveDomain(db, containerID); err != nil {
						log.Printf("Failed to remove domains for container %s: %v", containerID, err)
					}
					fmt.Printf("Container Event: Action=%s, Name=%s, ID=%s (no domain labels, removed from database)\n",
						msg.Action, msg.Actor.Attributes["name"], containerID)
				}
			}
		case err := <-errs:
			if err == io.EOF {
				return // End of stream
			}
			panic(err)
		}
	}
}
