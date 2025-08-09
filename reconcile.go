package main

import (
	"context"
	"database/sql"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func ReconcileContainers(ctx context.Context, cli *client.Client, db *sql.DB) {
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

		var domains []DomainConfig
		if containerInfo.Config != nil && containerInfo.Config.Labels != nil {
			domains = ExtractNginxDomains(containerInfo.Config.Labels)
		}

		if len(domains) > 0 {
			storedDomains, err := GetStoredDomains(db, containerID)
			if err != nil {
				log.Printf("ERROR: Database error during reconciliation: %v", err)
			}

			if !EqualDomainConfigs(domains, storedDomains) {
				log.Printf("RECONCILE: Updating container %s domain configs", containerID)
				for _, config := range domains {
					log.Printf("RECONCILE:   New: %s~%s~%s", config.Domain, config.Address, config.Port)
				}
				if err := InsertOrUpdateDomains(db, containerID, domains); err != nil {
					log.Printf("ERROR: Failed to update domains during reconciliation: %v", err)
				}
			}
		} else {
			if err := RemoveDomain(db, containerID); err != nil {
				log.Printf("ERROR: Failed to remove domains during reconciliation: %v", err)
			}
		}
	}

	containerIDs, err := GetAllContainerIDs(db)
	if err != nil {
		log.Printf("ERROR: Failed to get container IDs during reconciliation: %v", err)
		return
	}

	for _, containerID := range containerIDs {
		if !seenContainers[containerID] {
			log.Printf("RECONCILE: Removing deleted container %s from database", containerID)
			if err := RemoveDomain(db, containerID); err != nil {
				log.Printf("ERROR: Failed to remove deleted container during reconciliation: %v", err)
			}
		}
	}

	log.Printf("DEBUG: Container reconciliation completed")
}
