package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	_ "github.com/tursodatabase/go-libsql"
	"io"
	"log"
	"strings"
	"time"
)

func initDB() (*sql.DB, error) {
	dbURL := "file:./containers.db"

	db, err := sql.Open("libsql", dbURL)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	//defer db.Close()

	// Create table if it doesn't exist
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS container_domains (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			container_id TEXT NOT NULL,
			domain TEXT NOT NULL,
			address TEXT NOT NULL,
			port TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(container_id, domain)
		)
	`

	if _, err := db.Exec(createTableSQL); err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return db, nil
}

type DomainConfig struct {
	Domain  string
	Address string
	Port    string
}

func extractNginxDomains(labels map[string]string) []DomainConfig {
	var domains []DomainConfig

	// Look for all labels that start with "nginx-domain"
	for key, value := range labels {
		if key == "nginx-domain" || strings.HasPrefix(key, "nginx-domain.") {
			// Split multiple entries by comma or semicolon and trim whitespace
			entryList := strings.FieldsFunc(value, func(c rune) bool {
				return c == ',' || c == ';'
			})

			for _, entry := range entryList {
				entry = strings.TrimSpace(entry)
				if entry != "" {
					// Parse domain~address~port format
					parts := strings.Split(entry, "~")
					if len(parts) == 3 {
						domain := strings.TrimSpace(parts[0])
						address := strings.TrimSpace(parts[1])
						port := strings.TrimSpace(parts[2])

						if domain != "" && address != "" && port != "" {
							domains = append(domains, DomainConfig{
								Domain:  domain,
								Address: address,
								Port:    port,
							})
						}
					} else {
						log.Printf("WARNING: Invalid nginx-domain format '%s', expected 'domain~address~port'", entry)
					}
				}
			}
		}
	}

	return domains
}

func insertOrUpdateDomain(db *sql.DB, containerID string, config DomainConfig) error {
	query := `
		INSERT OR IGNORE INTO container_domains (container_id, domain, address, port, created_at, updated_at) 
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`
	log.Printf("DEBUG: Executing insert/update for container %s with domain %s, address %s, port %s",
		containerID, config.Domain, config.Address, config.Port)
	result, err := db.Exec(query, containerID, config.Domain, config.Address, config.Port)
	if err != nil {
		return fmt.Errorf("failed to insert/update domain: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("DEBUG: Could not get rows affected: %v", err)
	} else {
		log.Printf("DEBUG: Rows affected: %d", rowsAffected)
	}

	return nil
}

func insertOrUpdateDomains(db *sql.DB, containerID string, domains []DomainConfig) error {
	// First, remove all existing domains for this container
	if err := removeAllDomains(db, containerID); err != nil {
		return fmt.Errorf("failed to remove existing domains: %w", err)
	}

	// Then insert all current domains
	for _, config := range domains {
		if err := insertOrUpdateDomain(db, containerID, config); err != nil {
			return fmt.Errorf("failed to insert domain %s: %w", config.Domain, err)
		}
	}

	log.Printf("DEBUG: Inserted %d domains for container %s", len(domains), containerID)
	return nil
}

func removeAllDomains(db *sql.DB, containerID string) error {
	query := `DELETE FROM container_domains WHERE container_id = ?`
	result, err := db.Exec(query, containerID)
	if err != nil {
		return fmt.Errorf("failed to remove domains: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("DEBUG: Could not get rows affected for delete all: %v", err)
	} else {
		log.Printf("DEBUG: Delete all rows affected: %d", rowsAffected)
	}

	return nil
}

func removeDomain(db *sql.DB, containerID string) error {
	return removeAllDomains(db, containerID)
}

func verifyInsert(db *sql.DB, containerID string) {
	query := `SELECT domain, address, port FROM container_domains WHERE container_id = ?`
	rows, err := db.Query(query, containerID)
	if err != nil {
		log.Printf("DEBUG: Error querying domains for container %s: %v", containerID, err)
		return
	}
	defer rows.Close()

	var configs []DomainConfig
	for rows.Next() {
		var config DomainConfig
		if err := rows.Scan(&config.Domain, &config.Address, &config.Port); err != nil {
			log.Printf("DEBUG: Error scanning domain config: %v", err)
			continue
		}
		configs = append(configs, config)
	}

	if len(configs) == 0 {
		log.Printf("DEBUG: No domain configs found for container %s", containerID)
	} else {
		log.Printf("DEBUG: Found %d domain configs for container %s:", len(configs), containerID)
		for _, config := range configs {
			log.Printf("DEBUG:   - Domain: %s, Address: %s, Port: %s", config.Domain, config.Address, config.Port)
		}
	}
}

func listAllEntries(db *sql.DB) {
	query := `SELECT id, container_id, domain, address, port, created_at, updated_at FROM container_domains`
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("DEBUG: Error listing all entries: %v", err)
		return
	}
	defer rows.Close()

	log.Printf("DEBUG: Current database contents:")
	count := 0
	for rows.Next() {
		var id int
		var containerID, domain, address, port, createdAt, updatedAt string
		if err := rows.Scan(&id, &containerID, &domain, &address, &port, &createdAt, &updatedAt); err != nil {
			log.Printf("DEBUG: Error scanning row: %v", err)
			continue
		}
		log.Printf("DEBUG: - ID: %d, Container: %s, Domain: %s, Address: %s, Port: %s, Created: %s, Updated: %s",
			id, containerID, domain, address, port, createdAt, updatedAt)
		count++
	}
	log.Printf("DEBUG: Total entries: %d", count)
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	// Create maps to count occurrences
	countA := make(map[string]int)
	countB := make(map[string]int)

	for _, item := range a {
		countA[item]++
	}

	for _, item := range b {
		countB[item]++
	}

	// Compare maps
	for key, count := range countA {
		if countB[key] != count {
			return false
		}
	}

	return true
}

func equalDomainConfigs(a, b []DomainConfig) bool {
	if len(a) != len(b) {
		return false
	}

	// Create maps to count occurrences of domain~address~port combinations
	countA := make(map[string]int)
	countB := make(map[string]int)

	for _, config := range a {
		key := fmt.Sprintf("%s~%s~%s", config.Domain, config.Address, config.Port)
		countA[key]++
	}

	for _, config := range b {
		key := fmt.Sprintf("%s~%s~%s", config.Domain, config.Address, config.Port)
		countB[key]++
	}

	// Compare maps
	for key, count := range countA {
		if countB[key] != count {
			return false
		}
	}

	return true
}

func reconcileContainers(ctx context.Context, cli *client.Client, db *sql.DB) {
	log.Printf("DEBUG: Starting container reconciliation")

	// Get all containers (including stopped ones)
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		log.Printf("ERROR: Failed to list containers for reconciliation: %v", err)
		return
	}

	// Track containers we've seen
	seenContainers := make(map[string]bool)

	// Check each container for domain label
	for _, container := range containers {
		containerID := container.ID
		seenContainers[containerID] = true

		// Skip if container is not running (we only care about running containers with domain labels)
		if container.State != "running" {
			continue
		}

		// Inspect container to get labels
		containerInfo, err := cli.ContainerInspect(ctx, containerID)
		if err != nil {
			log.Printf("DEBUG: Failed to inspect container %s during reconciliation: %v", containerID, err)
			continue
		}

		// Look for nginx-domain labels
		var domains []DomainConfig
		if containerInfo.Config != nil && containerInfo.Config.Labels != nil {
			domains = extractNginxDomains(containerInfo.Config.Labels)
		}

		if len(domains) > 0 {
			// Get existing domains from database
			query := `SELECT domain, address, port FROM container_domains WHERE container_id = ?`
			rows, err := db.Query(query, containerID)
			if err != nil {
				log.Printf("ERROR: Database error during reconciliation: %v", err)
				continue
			}

			var storedDomains []DomainConfig
			for rows.Next() {
				var config DomainConfig
				if err := rows.Scan(&config.Domain, &config.Address, &config.Port); err != nil {
					log.Printf("ERROR: Failed to scan domain during reconciliation: %v", err)
					continue
				}
				storedDomains = append(storedDomains, config)
			}
			rows.Close()

			// Compare current domains with stored domains
			if !equalDomainConfigs(domains, storedDomains) {
				log.Printf("RECONCILE: Updating container %s domain configs", containerID)
				for _, config := range domains {
					log.Printf("RECONCILE:   New: %s~%s~%s", config.Domain, config.Address, config.Port)
				}
				if err := insertOrUpdateDomains(db, containerID, domains); err != nil {
					log.Printf("ERROR: Failed to update domains during reconciliation: %v", err)
				}
			}
		} else {
			// No domain labels, remove from database if they exist
			if err := removeDomain(db, containerID); err != nil {
				log.Printf("ERROR: Failed to remove domains during reconciliation: %v", err)
			}
		}
	}

	// Remove entries for containers that no longer exist
	query := `SELECT container_id FROM container_domains`
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("ERROR: Failed to query container_domains during reconciliation: %v", err)
		return
	}
	defer rows.Close()

	var containersToRemove []string
	for rows.Next() {
		var containerID string
		if err := rows.Scan(&containerID); err != nil {
			log.Printf("ERROR: Failed to scan container_id during reconciliation: %v", err)
			continue
		}

		if !seenContainers[containerID] {
			containersToRemove = append(containersToRemove, containerID)
		}
	}

	for _, containerID := range containersToRemove {
		log.Printf("RECONCILE: Removing deleted container %s from database", containerID)
		if err := removeDomain(db, containerID); err != nil {
			log.Printf("ERROR: Failed to remove deleted container during reconciliation: %v", err)
		}
	}

	log.Printf("DEBUG: Container reconciliation completed")
}

func main() {
	ctx := context.Background()

	// Initialize database
	db, err := initDB()
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

	// Run initial reconciliation
	go reconcileContainers(ctx, cli, db)

	for {
		select {
		case <-ticker.C:
			// Run reconciliation every minute
			go reconcileContainers(ctx, cli, db)
		case msg := <-messages:
			if msg.Type == "container" {
				containerID := msg.ID

				// Handle container removal/destruction events
				if msg.Action == "destroy" {
					if err := removeDomain(db, containerID); err != nil {
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
					domains = extractNginxDomains(containerInfo.Config.Labels)
				}

				if len(domains) > 0 {
					// Insert or update domains in database
					if err := insertOrUpdateDomains(db, containerID, domains); err != nil {
						log.Printf("Failed to store domains for container %s: %v", containerID, err)
					} else {
						// Verify the insert worked
						verifyInsert(db, containerID)
						listAllEntries(db)
					}
					fmt.Printf("Container Event: Action=%s, Name=%s, ID=%s, Domain Configs: (stored in database)\n",
						msg.Action, msg.Actor.Attributes["name"], containerID)
					for _, config := range domains {
						fmt.Printf("  - %s~%s~%s\n", config.Domain, config.Address, config.Port)
					}
				} else {
					// Remove from database if no domain labels exist
					if err := removeDomain(db, containerID); err != nil {
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
