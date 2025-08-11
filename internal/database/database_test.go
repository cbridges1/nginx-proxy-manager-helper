package database

import (
	"database/sql"
	"reflect"
	"testing"

	"github.com/cbridges/nginx-proxy-manager-helper/internal/models"
	_ "github.com/tursodatabase/go-libsql"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("libsql", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
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
		t.Fatalf("Failed to create test table: %v", err)
	}

	return db
}

func TestInit(t *testing.T) {
	// This test is more of an integration test as it creates an actual DB file
	// For unit testing, we'll use the setupTestDB helper
	db := setupTestDB(t)
	defer db.Close()

	// Test that the table exists
	var tableName string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='container_domains'").Scan(&tableName)
	if err != nil {
		t.Fatalf("Failed to find container_domains table: %v", err)
	}

	if tableName != "container_domains" {
		t.Errorf("Expected table name 'container_domains', got '%s'", tableName)
	}
}

func TestInsertOrUpdateDomain(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	containerID := "test-container-123"
	config := models.DomainConfig{
		Domain:  "example.com",
		Address: "192.168.1.10",
		Port:    "8080",
	}

	// Test insert
	err := InsertOrUpdateDomain(db, containerID, config)
	if err != nil {
		t.Fatalf("InsertOrUpdateDomain() failed: %v", err)
	}

	// Verify insert
	var domain, address, port string
	err = db.QueryRow("SELECT domain, address, port FROM container_domains WHERE container_id = ?", containerID).Scan(&domain, &address, &port)
	if err != nil {
		t.Fatalf("Failed to query inserted domain: %v", err)
	}

	if domain != config.Domain || address != config.Address || port != config.Port {
		t.Errorf("Inserted values don't match. Expected %+v, got domain=%s, address=%s, port=%s", config, domain, address, port)
	}

	// Test update
	updatedConfig := models.DomainConfig{
		Domain:  "example.com",
		Address: "192.168.1.20",
		Port:    "9000",
	}

	err = InsertOrUpdateDomain(db, containerID, updatedConfig)
	if err != nil {
		t.Fatalf("InsertOrUpdateDomain() update failed: %v", err)
	}

	// Verify update
	err = db.QueryRow("SELECT domain, address, port FROM container_domains WHERE container_id = ?", containerID).Scan(&domain, &address, &port)
	if err != nil {
		t.Fatalf("Failed to query updated domain: %v", err)
	}

	if address != updatedConfig.Address || port != updatedConfig.Port {
		t.Errorf("Updated values don't match. Expected address=%s, port=%s, got address=%s, port=%s", updatedConfig.Address, updatedConfig.Port, address, port)
	}
}

func TestInsertOrUpdateDomains(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	containerID := "test-container-456"
	domains := []models.DomainConfig{
		{Domain: "api.example.com", Address: "192.168.1.10", Port: "8080"},
		{Domain: "web.example.com", Address: "192.168.1.10", Port: "3000"},
	}

	err := InsertOrUpdateDomains(db, containerID, domains)
	if err != nil {
		t.Fatalf("InsertOrUpdateDomains() failed: %v", err)
	}

	// Verify all domains were inserted
	rows, err := db.Query("SELECT domain, address, port FROM container_domains WHERE container_id = ? ORDER BY domain", containerID)
	if err != nil {
		t.Fatalf("Failed to query domains: %v", err)
	}
	defer rows.Close()

	var results []models.DomainConfig
	for rows.Next() {
		var config models.DomainConfig
		if err := rows.Scan(&config.Domain, &config.Address, &config.Port); err != nil {
			t.Fatalf("Failed to scan domain: %v", err)
		}
		results = append(results, config)
	}

	if len(results) != len(domains) {
		t.Errorf("Expected %d domains, got %d", len(domains), len(results))
	}

	// Check that domains are correctly sorted and match
	expected := []models.DomainConfig{
		{Domain: "api.example.com", Address: "192.168.1.10", Port: "8080"},
		{Domain: "web.example.com", Address: "192.168.1.10", Port: "3000"},
	}

	if !reflect.DeepEqual(results, expected) {
		t.Errorf("Results don't match expected. Got %+v, want %+v", results, expected)
	}
}

func TestRemoveAllDomains(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	containerID := "test-container-789"
	domains := []models.DomainConfig{
		{Domain: "api.example.com", Address: "192.168.1.10", Port: "8080"},
		{Domain: "web.example.com", Address: "192.168.1.10", Port: "3000"},
	}

	// Insert domains first
	err := InsertOrUpdateDomains(db, containerID, domains)
	if err != nil {
		t.Fatalf("Failed to insert domains: %v", err)
	}

	// Remove all domains
	err = RemoveAllDomains(db, containerID)
	if err != nil {
		t.Fatalf("RemoveAllDomains() failed: %v", err)
	}

	// Verify domains were removed
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM container_domains WHERE container_id = ?", containerID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query domain count: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 domains after removal, got %d", count)
	}
}

func TestRemoveDomain(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	containerID := "test-container-remove"
	domains := []models.DomainConfig{
		{Domain: "test.example.com", Address: "192.168.1.10", Port: "8080"},
	}

	err := InsertOrUpdateDomains(db, containerID, domains)
	if err != nil {
		t.Fatalf("Failed to insert domains: %v", err)
	}

	err = RemoveDomain(db, containerID)
	if err != nil {
		t.Fatalf("RemoveDomain() failed: %v", err)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM container_domains WHERE container_id = ?", containerID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query domain count: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 domains after removal, got %d", count)
	}
}

func TestGetStoredDomains(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	containerID := "test-container-get"
	expected := []models.DomainConfig{
		{Domain: "api.example.com", Address: "192.168.1.10", Port: "8080"},
		{Domain: "web.example.com", Address: "192.168.1.20", Port: "3000"},
	}

	err := InsertOrUpdateDomains(db, containerID, expected)
	if err != nil {
		t.Fatalf("Failed to insert domains: %v", err)
	}

	result, err := GetStoredDomains(db, containerID)
	if err != nil {
		t.Fatalf("GetStoredDomains() failed: %v", err)
	}

	if len(result) != len(expected) {
		t.Errorf("Expected %d domains, got %d", len(expected), len(result))
	}

	// Note: The results might not be in the same order as inserted,
	// so we'll check if all expected domains are present
	for _, expectedDomain := range expected {
		found := false
		for _, resultDomain := range result {
			if reflect.DeepEqual(expectedDomain, resultDomain) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected domain %+v not found in results %+v", expectedDomain, result)
		}
	}
}

func TestGetAllContainerIDs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	containers := []struct {
		id      string
		domains []models.DomainConfig
	}{
		{
			id: "container1",
			domains: []models.DomainConfig{
				{Domain: "test1.com", Address: "192.168.1.1", Port: "8080"},
			},
		},
		{
			id: "container2",
			domains: []models.DomainConfig{
				{Domain: "test2.com", Address: "192.168.1.2", Port: "9000"},
			},
		},
	}

	// Insert domains for multiple containers
	for _, container := range containers {
		err := InsertOrUpdateDomains(db, container.id, container.domains)
		if err != nil {
			t.Fatalf("Failed to insert domains for container %s: %v", container.id, err)
		}
	}

	containerIDs, err := GetAllContainerIDs(db)
	if err != nil {
		t.Fatalf("GetAllContainerIDs() failed: %v", err)
	}

	expectedIDs := []string{"container1", "container2"}
	if len(containerIDs) != len(expectedIDs) {
		t.Errorf("Expected %d container IDs, got %d", len(expectedIDs), len(containerIDs))
	}

	// Check that all expected IDs are present
	for _, expectedID := range expectedIDs {
		found := false
		for _, id := range containerIDs {
			if id == expectedID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected container ID %s not found in results %v", expectedID, containerIDs)
		}
	}
}

func TestGetAllDomains(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	containers := []struct {
		id      string
		domains []models.DomainConfig
	}{
		{
			id: "container1",
			domains: []models.DomainConfig{
				{Domain: "api.test1.com", Address: "192.168.1.1", Port: "8080"},
				{Domain: "web.test1.com", Address: "192.168.1.1", Port: "3000"},
			},
		},
		{
			id: "container2",
			domains: []models.DomainConfig{
				{Domain: "api.test2.com", Address: "192.168.1.2", Port: "8080"},
			},
		},
	}

	var expectedDomains []models.DomainConfig

	// Insert domains for multiple containers
	for _, container := range containers {
		err := InsertOrUpdateDomains(db, container.id, container.domains)
		if err != nil {
			t.Fatalf("Failed to insert domains for container %s: %v", container.id, err)
		}
		expectedDomains = append(expectedDomains, container.domains...)
	}

	allDomains, err := GetAllDomains(db)
	if err != nil {
		t.Fatalf("GetAllDomains() failed: %v", err)
	}

	if len(allDomains) != len(expectedDomains) {
		t.Errorf("Expected %d domains, got %d", len(expectedDomains), len(allDomains))
	}

	// Check that all expected domains are present
	for _, expectedDomain := range expectedDomains {
		found := false
		for _, domain := range allDomains {
			if reflect.DeepEqual(expectedDomain, domain) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected domain %+v not found in results %+v", expectedDomain, allDomains)
		}
	}
}

func TestGetDomainsForContainer(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	containerID := "test-container-domains"
	domains := []models.DomainConfig{
		{Domain: "api.example.com", Address: "192.168.1.10", Port: "8080"},
		{Domain: "web.example.com", Address: "192.168.1.20", Port: "3000"},
	}

	err := InsertOrUpdateDomains(db, containerID, domains)
	if err != nil {
		t.Fatalf("Failed to insert domains: %v", err)
	}

	result, err := GetDomainsForContainer(db, containerID)
	if err != nil {
		t.Fatalf("GetDomainsForContainer() failed: %v", err)
	}

	expectedDomainNames := []string{"api.example.com", "web.example.com"}
	if len(result) != len(expectedDomainNames) {
		t.Errorf("Expected %d domain names, got %d", len(expectedDomainNames), len(result))
	}

	// Check that all expected domain names are present
	for _, expectedName := range expectedDomainNames {
		found := false
		for _, name := range result {
			if name == expectedName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected domain name %s not found in results %v", expectedName, result)
		}
	}
}

func TestGetDomainsForContainer_NonExistent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	result, err := GetDomainsForContainer(db, "non-existent-container")
	if err != nil {
		t.Fatalf("GetDomainsForContainer() failed for non-existent container: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected 0 domains for non-existent container, got %d", len(result))
	}
}
