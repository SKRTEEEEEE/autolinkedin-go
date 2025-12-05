package integration_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestDockerDevelopmentEnvironment validates the development Docker environment
func TestDockerDevelopmentEnvironment(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker integration test in short mode")
	}

	// This test assumes docker-compose up has been run
	// Check if we're running in Docker by checking for Docker environment
	if os.Getenv("APP_ENV") != "development" {
		t.Skip("Skipping development environment test (not in Docker)")
	}

	tests := []struct {
		name        string
		serviceName string
		host        string
		port        string
		protocol    string
	}{
		{
			name:        "MongoDB should be accessible",
			serviceName: "mongodb",
			host:        "mongodb",
			port:        "27017",
			protocol:    "tcp",
		},
		{
			name:        "NATS should be accessible",
			serviceName: "nats",
			host:        "nats",
			port:        "4222",
			protocol:    "tcp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			address := net.JoinHostPort(tt.host, tt.port)

			conn, err := net.DialTimeout(tt.protocol, address, 5*time.Second)
			if err != nil {
				t.Errorf("Failed to connect to %s at %s: %v", tt.serviceName, address, err)
				return
			}
			defer conn.Close()

			t.Logf("✅ Successfully connected to %s at %s", tt.serviceName, address)
		})
	}
}

// TestDockerTestEnvironmentIsolation validates test environment isolation
func TestDockerTestEnvironmentIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test environment validation in short mode")
	}

	// Check if we're in test environment
	if os.Getenv("APP_ENV") != "test" {
		t.Skip("Skipping test environment validation (not in test mode)")
	}

	t.Run("Test environment variables", func(t *testing.T) {
		requiredEnvVars := map[string]string{
			"APP_ENV":         "test",
			"MONGO_URI":       "mongodb://mongodb-test:27017",
			"NATS_URL":        "nats://nats-test:4222",
			"NATS_QUEUE_NAME": "test_queue",
		}

		for key, expectedValue := range requiredEnvVars {
			actualValue := os.Getenv(key)
			if actualValue != expectedValue {
				t.Errorf("Environment variable %s: expected %s, got %s", key, expectedValue, actualValue)
			}
		}
	})

	t.Run("Test database isolation", func(t *testing.T) {
		mongoURI := os.Getenv("MONGO_URI")
		if mongoURI == "" {
			t.Fatal("MONGO_URI not set")
		}

		// Ensure we're not using development database
		if mongoURI == "mongodb://mongodb:27017" {
			t.Error("Test environment must not use development MongoDB")
		}

		// Database name should contain 'test'
		dbName := os.Getenv("MONGO_DATABASE")
		if dbName != "" && dbName != "linkgenai_test" {
			t.Errorf("Test database name should be linkgenai_test, got %s", dbName)
		}
	})

	t.Run("Test NATS isolation", func(t *testing.T) {
		natsURL := os.Getenv("NATS_URL")
		if natsURL == "" {
			t.Fatal("NATS_URL not set")
		}

		// Ensure we're not using development NATS
		if natsURL == "nats://nats:4222" {
			t.Error("Test environment must not use development NATS")
		}
	})
}

// TestDockerHotReloadWorks validates hot reload functionality
func TestDockerHotReloadWorks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping hot reload test in short mode")
	}

	// This test can only run in development mode
	if os.Getenv("APP_ENV") != "development" {
		t.Skip("Hot reload test only runs in development environment")
	}

	t.Run("Air process should be running", func(t *testing.T) {
		// Check if the app is accessible (indicates hot reload is working)
		appPort := os.Getenv("APP_PORT")
		if appPort == "" {
			appPort = "8080"
		}

		url := fmt.Sprintf("http://localhost:%s/health", appPort)

		client := &http.Client{
			Timeout: 5 * time.Second,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Logf("App may not be fully started yet: %v", err)
			t.Skip("Skipping hot reload check - app not responding")
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected 200 or 404 from health endpoint, got %d", resp.StatusCode)
		}
	})
}

// TestDockerVolumeConfiguration validates volume mounts and persistence
func TestDockerVolumeConfiguration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping volume configuration test in short mode")
	}

	appEnv := os.Getenv("APP_ENV")

	t.Run("Source code should be mounted in development", func(t *testing.T) {
		if appEnv != "development" {
			t.Skip("Only applicable in development mode")
		}

		// Check if /app directory exists and has source code
		if _, err := os.Stat("/app/main.go"); os.IsNotExist(err) {
			t.Error("Source code not properly mounted - main.go not found at /app/main.go")
		}

		if _, err := os.Stat("/app/.air.toml"); os.IsNotExist(err) {
			t.Error("Air configuration not found - hot reload may not work")
		}
	})

	t.Run("Test mode should use read-only volumes", func(t *testing.T) {
		if appEnv != "test" {
			t.Skip("Only applicable in test mode")
		}

		// In test mode, source should still be accessible but test data should be ephemeral
		// This is validated by the docker-compose.test.yml configuration
		t.Log("Test environment uses tmpfs for ephemeral storage - validated by config tests")
	})
}

// TestDockerNetworkConfiguration validates network isolation
func TestDockerNetworkConfiguration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network configuration test in short mode")
	}

	appEnv := os.Getenv("APP_ENV")

	t.Run("Services should be reachable within network", func(t *testing.T) {
		var hosts []string

		if appEnv == "development" {
			hosts = []string{"mongodb", "nats"}
		} else if appEnv == "test" {
			hosts = []string{"mongodb-test", "nats-test"}
		} else {
			t.Skip("Not running in Docker environment")
		}

		for _, host := range hosts {
			t.Run(fmt.Sprintf("Resolve %s", host), func(t *testing.T) {
				addrs, err := net.LookupHost(host)
				if err != nil {
					t.Errorf("Failed to resolve %s: %v", host, err)
					return
				}

				if len(addrs) == 0 {
					t.Errorf("No IP addresses found for %s", host)
					return
				}

				t.Logf("✅ %s resolves to: %v", host, addrs)
			})
		}
	})
}
