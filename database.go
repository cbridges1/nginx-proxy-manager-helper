package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/tursodatabase/go-libsql"
)

func InitDB() (*sql.DB, error) {
	dbURL := "file:./containers.db"

	db, err := sql.Open("libsql", dbURL)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}

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

func InsertOrUpdateDomain(db *sql.DB, containerID string, config DomainConfig) error {
	query := `
		INSERT INTO container_domains (container_id, domain, address, port, created_at, updated_at) 
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(container_id, domain) DO UPDATE SET
			address = excluded.address,
			port = excluded.port,
			updated_at = CURRENT_TIMESTAMP
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

func InsertOrUpdateDomains(db *sql.DB, containerID string, domains []DomainConfig) error {
	if err := RemoveAllDomains(db, containerID); err != nil {
		return fmt.Errorf("failed to remove existing domains: %w", err)
	}

	for _, config := range domains {
		if err := InsertOrUpdateDomain(db, containerID, config); err != nil {
			return fmt.Errorf("failed to insert domain %s: %w", config.Domain, err)
		}
	}

	log.Printf("DEBUG: Inserted %d domains for container %s", len(domains), containerID)
	return nil
}

func RemoveAllDomains(db *sql.DB, containerID string) error {
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

func RemoveDomain(db *sql.DB, containerID string) error {
	return RemoveAllDomains(db, containerID)
}

func VerifyInsert(db *sql.DB, containerID string) {
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

func ListAllEntries(db *sql.DB) {
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

func GetStoredDomains(db *sql.DB, containerID string) ([]DomainConfig, error) {
	query := `SELECT domain, address, port FROM container_domains WHERE container_id = ?`
	rows, err := db.Query(query, containerID)
	if err != nil {
		return nil, fmt.Errorf("database error during query: %w", err)
	}
	defer rows.Close()

	var storedDomains []DomainConfig
	for rows.Next() {
		var config DomainConfig
		if err := rows.Scan(&config.Domain, &config.Address, &config.Port); err != nil {
			return nil, fmt.Errorf("failed to scan domain: %w", err)
		}
		storedDomains = append(storedDomains, config)
	}

	return storedDomains, nil
}

func GetAllContainerIDs(db *sql.DB) ([]string, error) {
	query := `SELECT container_id FROM container_domains`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query container_domains: %w", err)
	}
	defer rows.Close()

	var containerIDs []string
	for rows.Next() {
		var containerID string
		if err := rows.Scan(&containerID); err != nil {
			return nil, fmt.Errorf("failed to scan container_id: %w", err)
		}
		containerIDs = append(containerIDs, containerID)
	}

	return containerIDs, nil
}

func GetAllDomains(db *sql.DB) ([]DomainConfig, error) {
	query := `SELECT domain, address, port FROM container_domains`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all domains: %w", err)
	}
	defer rows.Close()

	var domains []DomainConfig
	for rows.Next() {
		var config DomainConfig
		if err := rows.Scan(&config.Domain, &config.Address, &config.Port); err != nil {
			return nil, fmt.Errorf("failed to scan domain: %w", err)
		}
		domains = append(domains, config)
	}

	return domains, nil
}

func GetDomainsForContainer(db *sql.DB, containerID string) ([]string, error) {
	query := `SELECT domain FROM container_domains WHERE container_id = ?`
	rows, err := db.Query(query, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query domains for container: %w", err)
	}
	defer rows.Close()

	var domains []string
	for rows.Next() {
		var domain string
		if err := rows.Scan(&domain); err != nil {
			return nil, fmt.Errorf("failed to scan domain: %w", err)
		}
		domains = append(domains, domain)
	}

	return domains, nil
}
