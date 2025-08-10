package reconciler

import (
	"context"
	"database/sql"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	"github.com/cbridges/nginx-proxy-manager-helper/internal/config"
	"github.com/cbridges/nginx-proxy-manager-helper/internal/database"
	"github.com/cbridges/nginx-proxy-manager-helper/internal/models"
	"github.com/cbridges/nginx-proxy-manager-helper/internal/npm"
	"github.com/cbridges/nginx-proxy-manager-helper/internal/utils"
)

func ReconcileContainers(ctx context.Context, cli *client.Client, db *sql.DB, npmClient *npm.Client, cfg *config.Config) {
	log.Printf("DEBUG: Starting container reconciliation")

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		log.Printf("ERROR: Failed to list containers for reconciliation: %v", err)
		return
	}

	seenContainers := make(map[string]bool)

	for _, container := range containers {
		containerID := container.ID
		seenContainers[containerID] = true

		if container.State != "running" {
			continue
		}

		containerInfo, err := cli.ContainerInspect(ctx, containerID)
		if err != nil {
			log.Printf("DEBUG: Failed to inspect container %s during reconciliation: %v", containerID, err)
			continue
		}

		var domains []models.DomainConfig
		if containerInfo.Config != nil && containerInfo.Config.Labels != nil {
			domains = models.ExtractNginxDomains(containerInfo.Config.Labels)
		}

		if len(domains) > 0 {
			storedDomains, err := database.GetStoredDomains(db, containerID)
			if err != nil {
				log.Printf("ERROR: Database error during reconciliation: %v", err)
			}

			if !utils.EqualDomainConfigs(domains, storedDomains) {
				log.Printf("RECONCILE: Updating container %s domain configs", containerID)
				for _, config := range domains {
					log.Printf("RECONCILE:   New: %s~%s~%s", config.Domain, config.Address, config.Port)
				}
				if err := database.InsertOrUpdateDomains(db, containerID, domains); err != nil {
					log.Printf("ERROR: Failed to update domains during reconciliation: %v", err)
				} else {
					// Sync domains to NPM after successful database update
					log.Printf("DEBUG: Syncing domains to NPM for container %s", containerID)
					if err := npmClient.SyncDomainsToNPM(domains, cfg.CreateWildcardCerts, cfg.LetsEncryptEmail, cfg.CloudflareToken); err != nil {
						log.Printf("ERROR: Failed to sync domains to NPM for container %s: %v", containerID, err)
					} else {
						log.Printf("DEBUG: Successfully synced domains to NPM for container %s", containerID)
					}
				}
			}
		} else {
			// Container no longer has domain labels, remove existing domains
			existingDomains, err := database.GetDomainsForContainer(db, containerID)
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

			if err := database.RemoveDomain(db, containerID); err != nil {
				log.Printf("ERROR: Failed to remove domains during reconciliation: %v", err)
			}
		}
	}

	containerIDs, err := database.GetAllContainerIDs(db)
	if err != nil {
		log.Printf("ERROR: Failed to get container IDs during reconciliation: %v", err)
		return
	}

	for _, containerID := range containerIDs {
		if !seenContainers[containerID] {
			log.Printf("RECONCILE: Removing deleted container %s from database", containerID)

			// Get domains before removing them from database
			existingDomains, err := database.GetDomainsForContainer(db, containerID)
			if err != nil {
				log.Printf("ERROR: Failed to get existing domains for deleted container %s: %v", containerID, err)
			} else if len(existingDomains) > 0 {
				log.Printf("DEBUG: Removing %d domains from NPM for deleted container %s", len(existingDomains), containerID)
				if err := npmClient.RemoveProxyHostsByDomains(existingDomains); err != nil {
					log.Printf("ERROR: Failed to remove proxy hosts from NPM for deleted container %s: %v", containerID, err)
				} else {
					log.Printf("DEBUG: Successfully removed proxy hosts from NPM for deleted container %s", containerID)
				}
			}

			if err := database.RemoveDomain(db, containerID); err != nil {
				log.Printf("ERROR: Failed to remove deleted container during reconciliation: %v", err)
			}
		}
	}

	log.Printf("DEBUG: Container reconciliation completed")
}
