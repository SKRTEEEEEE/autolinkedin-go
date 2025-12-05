package infrastructure_test

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

// DockerComposeService represents a service in docker-compose.yml
type DockerComposeService struct {
	Build       map[string]interface{} `yaml:"build,omitempty"`
	Image       string                 `yaml:"image,omitempty"`
	Container   string                 `yaml:"container_name,omitempty"`
	Ports       []string               `yaml:"ports,omitempty"`
	Environment []string               `yaml:"environment,omitempty"`
	Volumes     []string               `yaml:"volumes,omitempty"`
	DependsOn   map[string]interface{} `yaml:"depends_on,omitempty"`
	Networks    []string               `yaml:"networks,omitempty"`
	Restart     string                 `yaml:"restart,omitempty"`
	Command     interface{}            `yaml:"command,omitempty"`
	Healthcheck map[string]interface{} `yaml:"healthcheck,omitempty"`
	Tmpfs       []string               `yaml:"tmpfs,omitempty"`
}

// DockerCompose represents docker-compose.yml structure
type DockerCompose struct {
	Version  string                          `yaml:"version"`
	Services map[string]DockerComposeService `yaml:"services"`
	Networks map[string]map[string]string    `yaml:"networks,omitempty"`
	Volumes  map[string]map[string]string    `yaml:"volumes,omitempty"`
}

// TestDockerComposeDevConfig validates docker-compose.yml for development
func TestDockerComposeDevConfig(t *testing.T) {
	// Read docker-compose.yml
	data, err := os.ReadFile("../../docker-compose.yml")
	if err != nil {
		t.Fatalf("Failed to read docker-compose.yml: %v", err)
	}

	var compose DockerCompose
	if err := yaml.Unmarshal(data, &compose); err != nil {
		t.Fatalf("Failed to parse docker-compose.yml: %v", err)
	}

	// Validate version
	if compose.Version != "3.8" {
		t.Errorf("Expected docker-compose version 3.8, got %s", compose.Version)
	}

	// Validate app service exists
	app, ok := compose.Services["app"]
	if !ok {
		t.Fatal("App service not found in docker-compose.yml")
	}

	// Validate app service has build configuration
	if app.Build == nil {
		t.Error("App service must have build configuration")
	}

	// Validate app service has volumes for hot reload
	if len(app.Volumes) == 0 {
		t.Error("App service must have volumes mounted for hot reload")
	}

	// Check for source code volume mount
	hasSourceVolume := false
	for _, vol := range app.Volumes {
		if vol == "./src:/app:delegated" || vol == "./src:/app" {
			hasSourceVolume = true
			break
		}
	}
	if !hasSourceVolume {
		t.Error("App service must mount ./src directory for hot reload")
	}

	// Validate MongoDB service
	mongodb, ok := compose.Services["mongodb"]
	if !ok {
		t.Fatal("MongoDB service not found in docker-compose.yml")
	}

	// MongoDB must have healthcheck
	if mongodb.Healthcheck == nil {
		t.Error("MongoDB service must have healthcheck configured")
	}

	// MongoDB must have persistent volume
	if len(mongodb.Volumes) == 0 {
		t.Error("MongoDB service must have persistent volumes in development mode")
	}

	// Validate NATS service
	nats, ok := compose.Services["nats"]
	if !ok {
		t.Fatal("NATS service not found in docker-compose.yml")
	}

	if nats.Image == "" {
		t.Error("NATS service must specify an image")
	}

	// Validate networks
	if compose.Networks == nil || len(compose.Networks) == 0 {
		t.Error("docker-compose.yml must define at least one network")
	}

	// Validate volumes
	if compose.Volumes == nil || len(compose.Volumes) == 0 {
		t.Error("docker-compose.yml must define persistent volumes for development")
	}
}

// TestDockerComposeTestConfig validates docker-compose.test.yml for isolated testing
func TestDockerComposeTestConfig(t *testing.T) {
	// Read docker-compose.test.yml
	data, err := os.ReadFile("../../docker-compose.test.yml")
	if err != nil {
		t.Fatalf("Failed to read docker-compose.test.yml: %v", err)
	}

	var compose DockerCompose
	if err := yaml.Unmarshal(data, &compose); err != nil {
		t.Fatalf("Failed to parse docker-compose.test.yml: %v", err)
	}

	// Validate version
	if compose.Version != "3.8" {
		t.Errorf("Expected docker-compose version 3.8, got %s", compose.Version)
	}

	// Validate app service exists
	app, ok := compose.Services["app"]
	if !ok {
		t.Fatal("App service not found in docker-compose.test.yml")
	}

	// Validate app runs tests
	if app.Command == nil {
		t.Error("App service must have command to run tests")
	}

	// Validate MongoDB test service uses tmpfs
	mongodbTest, ok := compose.Services["mongodb-test"]
	if !ok {
		t.Fatal("MongoDB test service not found in docker-compose.test.yml")
	}

	// MongoDB test must use tmpfs for ephemeral storage
	if len(mongodbTest.Tmpfs) == 0 {
		t.Error("MongoDB test service must use tmpfs for ephemeral storage")
	}

	// Check tmpfs includes data directory
	hasTmpfsData := false
	for _, tmpfs := range mongodbTest.Tmpfs {
		if tmpfs == "/data/db" {
			hasTmpfsData = true
			break
		}
	}
	if !hasTmpfsData {
		t.Error("MongoDB test service must use tmpfs for /data/db")
	}

	// Validate NATS test service uses tmpfs
	natsTest, ok := compose.Services["nats-test"]
	if !ok {
		t.Fatal("NATS test service not found in docker-compose.test.yml")
	}

	if len(natsTest.Tmpfs) == 0 {
		t.Error("NATS test service must use tmpfs for ephemeral storage")
	}

	// Test environment must NOT have persistent volumes defined
	if compose.Volumes != nil && len(compose.Volumes) > 0 {
		t.Error("docker-compose.test.yml must NOT define persistent volumes (ephemeral only)")
	}

	// Validate separate network for isolation
	if compose.Networks == nil || len(compose.Networks) == 0 {
		t.Error("docker-compose.test.yml must define isolated network")
	}

	// Ensure network name is different from development
	for networkName := range compose.Networks {
		if networkName == "linkgenai-network" {
			t.Error("Test environment must use a different network name than development")
		}
	}
}

// TestDockerfileStructure validates Dockerfile multi-stage build
func TestDockerfileStructure(t *testing.T) {
	// Read Dockerfile
	data, err := os.ReadFile("../../Dockerfile")
	if err != nil {
		t.Fatalf("Failed to read Dockerfile: %v", err)
	}

	content := string(data)

	// Check for multi-stage build
	if !containsAll(content, []string{"FROM", "AS development", "AS builder", "AS production"}) {
		t.Error("Dockerfile must have multi-stage build with development, builder, and production stages")
	}

	// Check development stage has hot reload tool
	if !containsAll(content, []string{"cosmtrek/air", "AS development"}) {
		t.Error("Development stage must install air for hot reload")
	}

	// Check builder stage builds the binary
	if !containsAll(content, []string{"AS builder", "go build"}) {
		t.Error("Builder stage must compile the Go binary")
	}

	// Check production stage uses minimal image
	if !containsAll(content, []string{"alpine:latest", "AS production"}) {
		t.Error("Production stage must use minimal alpine image")
	}

	// Check production stage doesn't have unnecessary tools
	productionIndex := findIndex(content, "AS production")
	if productionIndex > 0 {
		productionSection := content[productionIndex:]
		if containsAll(productionSection, []string{"air", "git"}) {
			t.Error("Production stage must not contain development tools")
		}
	}
}

// TestDockerIgnoreFile validates .dockerignore configuration
func TestDockerIgnoreFile(t *testing.T) {
	// Read .dockerignore
	data, err := os.ReadFile("../../.dockerignore")
	if err != nil {
		t.Fatalf("Failed to read .dockerignore: %v", err)
	}

	content := string(data)

	// Check for common exclusions
	requiredExclusions := []string{".git", "test", "docs", "*.md"}

	for _, exclusion := range requiredExclusions {
		if !contains(content, exclusion) {
			t.Errorf(".dockerignore must exclude %s", exclusion)
		}
	}
}

// Helper function to check if content contains all substrings
func containsAll(content string, substrings []string) bool {
	for _, substr := range substrings {
		if !contains(content, substr) {
			return false
		}
	}
	return true
}

// Helper function to check if content contains substring
func contains(content, substr string) bool {
	return len(content) >= len(substr) && findIndex(content, substr) >= 0
}

// Helper function to find index of substring
func findIndex(content, substr string) int {
	for i := 0; i <= len(content)-len(substr); i++ {
		if content[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
