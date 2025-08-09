package main

import (
	"context"
	"io"
	"log"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func main() {
	ctx := context.Background()

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

	//messages,
	_, errs := cli.Events(ctx, types.EventsOptions{})

	// Start reconciliation ticker (every minute)
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// Run initial reconciliation
	go ReconcileContainers(ctx, cli, db)

	for {
		select {
		case <-ticker.C:
			// Run reconciliation every minute
			go ReconcileContainers(ctx, cli, db)
		//case msg := <-messages:
		//	if msg.Type == "container" {
		//		containerID := msg.ID
		//
		//		// Handle container removal/destruction events
		//		if msg.Action == "destroy" {
		//			if err := RemoveDomain(db, containerID); err != nil {
		//				log.Printf("Failed to remove domain for container %s: %v", containerID, err)
		//			}
		//			fmt.Printf("Container Event: Action=%s, Name=%s, ID=%s (removed from database)\n",
		//				msg.Action, msg.Actor.Attributes["name"], containerID)
		//			continue
		//		}
		//
		//		// For other events, inspect the container to check for domain label
		//		containerInfo, err := cli.ContainerInspect(ctx, containerID)
		//		if err != nil {
		//			fmt.Printf("Container Event: Action=%s, Name=%s, ID=%s (failed to inspect: %v)\n",
		//				msg.Action, msg.Actor.Attributes["name"], containerID, err)
		//			continue
		//		}
		//
		//		// Look for nginx-domain labels
		//		var domains []DomainConfig
		//		if containerInfo.Config != nil && containerInfo.Config.Labels != nil {
		//			domains = ExtractNginxDomains(containerInfo.Config.Labels)
		//		}
		//
		//		log.Printf("Igloo: %s", domains)
		//		if len(domains) > 0 {
		//			// Insert or update domains in database
		//			log.Printf("Igloo %s", domains)
		//			if err := InsertOrUpdateDomains(db, containerID, domains); err != nil {
		//				log.Printf("Failed to store domains for container %s: %v", containerID, err)
		//			} else {
		//				// Verify the insert worked
		//				VerifyInsert(db, containerID)
		//				ListAllEntries(db)
		//			}
		//			fmt.Printf("Container Event: Action=%s, Name=%s, ID=%s, Domain Configs: (stored in database)\n",
		//				msg.Action, msg.Actor.Attributes["name"], containerID)
		//			for _, config := range domains {
		//				fmt.Printf("  - %s~%s~%s\n", config.Domain, config.Address, config.Port)
		//			}
		//		} else {
		//			// Remove from database if no domain labels exist
		//			if err := RemoveDomain(db, containerID); err != nil {
		//				log.Printf("Failed to remove domains for container %s: %v", containerID, err)
		//			}
		//			fmt.Printf("Container Event: Action=%s, Name=%s, ID=%s (no domain labels, removed from database)\n",
		//				msg.Action, msg.Actor.Attributes["name"], containerID)
		//		}
		//	}
		case err := <-errs:
			if err == io.EOF {
				return // End of stream
			}
			panic(err)
		}
	}
}
