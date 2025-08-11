package test

import (
	"database/sql"
	"testing"

	"github.com/cbridges/nginx-proxy-manager-helper/internal/config"
	"github.com/cbridges/nginx-proxy-manager-helper/internal/database"
	"github.com/cbridges/nginx-proxy-manager-helper/internal/models"
	"github.com/cbridges/nginx-proxy-manager-helper/internal/npm"
	"github.com/cbridges/nginx-proxy-manager-helper/internal/utils"
	_ "github.com/tursodatabase/go-libsql"
)

func setupIntegrationTestDB(t *testing.T) *sql.DB {
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

func TestDatabaseIntegration(t *testing.T) {
	db := setupIntegrationTestDB(t)
	defer db.Close()

	containerID := "integration-test-container"
	domains := []models.DomainConfig{
		{Domain: "api.integration.test", Address: "192.168.100.1", Port: "8080"},
		{Domain: "web.integration.test", Address: "192.168.100.2", Port: "3000"},
	}

	// Test full database workflow
	err := database.InsertOrUpdateDomains(db, containerID, domains)
	if err != nil {
		t.Fatalf("Failed to insert domains: %v", err)
	}

	// Retrieve domains
	storedDomains, err := database.GetStoredDomains(db, containerID)
	if err != nil {
		t.Fatalf("Failed to get stored domains: %v", err)
	}

	// Verify domains match using utils
	if !utils.EqualDomainConfigs(domains, storedDomains) {
		t.Errorf("Stored domains don't match inserted domains")
		t.Logf("Expected: %+v", domains)
		t.Logf("Got: %+v", storedDomains)
	}

	// Test getting all domains
	allDomains, err := database.GetAllDomains(db)
	if err != nil {
		t.Fatalf("Failed to get all domains: %v", err)
	}

	if len(allDomains) != 2 {
		t.Errorf("Expected 2 domains, got %d", len(allDomains))
	}

	// Test getting container IDs
	containerIDs, err := database.GetAllContainerIDs(db)
	if err != nil {
		t.Fatalf("Failed to get container IDs: %v", err)
	}

	if len(containerIDs) != 2 { // Should have 2 entries (one for each domain)
		t.Errorf("Expected 2 container ID entries, got %d", len(containerIDs))
	}

	// Test domain names for container
	domainNames, err := database.GetDomainsForContainer(db, containerID)
	if err != nil {
		t.Fatalf("Failed to get domain names: %v", err)
	}

	expectedNames := []string{"api.integration.test", "web.integration.test"}
	if !utils.EqualStringSlices(domainNames, expectedNames) {
		t.Errorf("Domain names don't match")
		t.Logf("Expected: %+v", expectedNames)
		t.Logf("Got: %+v", domainNames)
	}

	// Test removal
	err = database.RemoveDomain(db, containerID)
	if err != nil {
		t.Fatalf("Failed to remove domains: %v", err)
	}

	// Verify removal
	remainingDomains, err := database.GetStoredDomains(db, containerID)
	if err != nil {
		t.Fatalf("Failed to check remaining domains: %v", err)
	}

	if len(remainingDomains) != 0 {
		t.Errorf("Expected 0 remaining domains, got %d", len(remainingDomains))
	}
}

func TestConfigIntegration(t *testing.T) {
	// Test that config loading works
	// Note: This test may use actual .env file if present, so we test structure rather than specific values

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test that config structure is populated (values may come from .env file)
	if cfg.NPMBaseURL == "" {
		t.Error("Expected NPMBaseURL to be non-empty")
	}

	if cfg.NPMEmail == "" {
		t.Error("Expected NPMEmail to be non-empty")
	}

	// Just verify the fields exist and are accessible
	_ = cfg.CreateWildcardCerts
	_ = cfg.CloudflareDomains
	_ = cfg.CloudflareEnabled
	_ = cfg.LetsEncryptEmail

	t.Logf("Config loaded successfully with NPMBaseURL: %s", cfg.NPMBaseURL)
}

func TestModelsIntegration(t *testing.T) {
	// Test domain extraction with various label formats
	testLabels := map[string]string{
		"nginx-domain-1": "https://secure.integration.test~192.168.200.1~8443",
		"nginx-domain-2": "http://plain.integration.test~192.168.200.2~8080",
		"nginx-domain-3": "api.integration.test~192.168.200.3~9000,web.integration.test~192.168.200.4~3000",
		"other-label":    "should-be-ignored",
	}

	domains := models.ExtractNginxDomains(testLabels)

	expectedDomains := []models.DomainConfig{
		{Domain: "https://secure.integration.test", Address: "192.168.200.1", Port: "8443"},
		{Domain: "http://plain.integration.test", Address: "192.168.200.2", Port: "8080"},
		{Domain: "api.integration.test", Address: "192.168.200.3", Port: "9000"},
		{Domain: "web.integration.test", Address: "192.168.200.4", Port: "3000"},
	}

	if len(domains) != len(expectedDomains) {
		t.Errorf("Expected %d domains, got %d", len(expectedDomains), len(domains))
	}

	// Check that all expected domains are present (order might vary)
	for _, expected := range expectedDomains {
		found := false
		for _, actual := range domains {
			if actual.Domain == expected.Domain &&
				actual.Address == expected.Address &&
				actual.Port == expected.Port {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected domain not found: %+v", expected)
		}
	}
}

func TestUtilsIntegration(t *testing.T) {
	// Test utils functions with real data scenarios

	domains1 := []models.DomainConfig{
		{Domain: "test1.com", Address: "192.168.1.1", Port: "8080"},
		{Domain: "test2.com", Address: "192.168.1.2", Port: "9000"},
	}

	domains2 := []models.DomainConfig{
		{Domain: "test2.com", Address: "192.168.1.2", Port: "9000"},
		{Domain: "test1.com", Address: "192.168.1.1", Port: "8080"},
	}

	domains3 := []models.DomainConfig{
		{Domain: "test1.com", Address: "192.168.1.1", Port: "8080"},
		{Domain: "test3.com", Address: "192.168.1.3", Port: "9000"}, // Different domain
	}

	// Test equal domains in different order
	if !utils.EqualDomainConfigs(domains1, domains2) {
		t.Error("Expected domains1 and domains2 to be equal (different order)")
	}

	// Test different domains
	if utils.EqualDomainConfigs(domains1, domains3) {
		t.Error("Expected domains1 and domains3 to be different")
	}

	// Test string slice comparison
	strings1 := []string{"a", "b", "c"}
	strings2 := []string{"c", "a", "b"}
	strings3 := []string{"a", "b", "d"}

	if !utils.EqualStringSlices(strings1, strings2) {
		t.Error("Expected strings1 and strings2 to be equal (different order)")
	}

	if utils.EqualStringSlices(strings1, strings3) {
		t.Error("Expected strings1 and strings3 to be different")
	}
}

func TestNPMClientStructure(t *testing.T) {
	// Test NPM client creation without making actual HTTP calls
	cfg := &config.Config{
		NPMBaseURL:  "http://test-npm:8181",
		NPMEmail:    "test@integration.com",
		NPMPassword: "test-password",
	}

	client := npm.NewClient(cfg)

	if client.BaseURL != cfg.NPMBaseURL {
		t.Errorf("Expected BaseURL %s, got %s", cfg.NPMBaseURL, client.BaseURL)
	}

	if client.Email != cfg.NPMEmail {
		t.Errorf("Expected Email %s, got %s", cfg.NPMEmail, client.Email)
	}

	if client.Password != cfg.NPMPassword {
		t.Errorf("Expected Password %s, got %s", cfg.NPMPassword, client.Password)
	}

	// Test that empty domains list doesn't cause errors
	err := client.RemoveProxyHostsByDomains([]string{})
	if err != nil {
		t.Errorf("RemoveProxyHostsByDomains with empty slice should not error, got: %v", err)
	}
}

// TestFullWorkflow tests a complete workflow integration
func TestFullWorkflow(t *testing.T) {
	// This test simulates a full workflow without external dependencies

	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 2. Initialize database
	db := setupIntegrationTestDB(t)
	defer db.Close()

	// 3. Create some test data
	containerID := "workflow-test-container"
	testLabels := map[string]string{
		"nginx-domain": "workflow.test.com~192.168.1.100~8080",
	}

	// 4. Extract domains from labels
	domains := models.ExtractNginxDomains(testLabels)
	if len(domains) != 1 {
		t.Fatalf("Expected 1 domain from labels, got %d", len(domains))
	}

	// 5. Store in database
	err = database.InsertOrUpdateDomains(db, containerID, domains)
	if err != nil {
		t.Fatalf("Failed to store domains: %v", err)
	}

	// 6. Retrieve and verify
	storedDomains, err := database.GetStoredDomains(db, containerID)
	if err != nil {
		t.Fatalf("Failed to retrieve domains: %v", err)
	}

	if !utils.EqualDomainConfigs(domains, storedDomains) {
		t.Errorf("Workflow failed: stored domains don't match original")
	}

	// 7. Create NPM client (structure test only)
	npmClient := npm.NewClient(cfg)
	if npmClient == nil {
		t.Error("Failed to create NPM client")
	}

	// 8. Test cleanup
	domainNames, err := database.GetDomainsForContainer(db, containerID)
	if err != nil {
		t.Fatalf("Failed to get domain names: %v", err)
	}

	if len(domainNames) != 1 || domainNames[0] != "workflow.test.com" {
		t.Errorf("Expected domain name 'workflow.test.com', got %v", domainNames)
	}

	// 9. Remove domains
	err = database.RemoveDomain(db, containerID)
	if err != nil {
		t.Fatalf("Failed to remove domains: %v", err)
	}

	// 10. Verify removal
	finalDomains, err := database.GetStoredDomains(db, containerID)
	if err != nil {
		t.Fatalf("Failed to check final domains: %v", err)
	}

	if len(finalDomains) != 0 {
		t.Errorf("Expected 0 domains after removal, got %d", len(finalDomains))
	}

	t.Log("Full workflow integration test passed")
}
