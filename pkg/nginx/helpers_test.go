package nginx

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mansoormajeed/dubernetes/pkg/config"
	"github.com/mansoormajeed/dubernetes/pkg/database"
	"github.com/mansoormajeed/dubernetes/pkg/docker"
)

func TestGenerateConfigFromDatabase(t *testing.T) {
	// Create temporary database for testing
	tmpDB := "/tmp/test_nginx_helpers.db"
	defer os.Remove(tmpDB)

	db, err := database.NewDatabase(tmpDB)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Reset database to clean state
	if err := db.Reset(); err != nil {
		t.Fatalf("Failed to reset database: %v", err)
	}

	t.Run("generate config from empty database", func(t *testing.T) {
		config, err := GenerateConfigFromDatabase(db)
		if err != nil {
			t.Fatalf("Failed to generate config from empty database: %v", err)
		}

		if config != "" {
			t.Errorf("Expected empty config, got: %s", config)
		}
	})

	t.Run("generate config with pods and replicas", func(t *testing.T) {
		// Create test pod
		pod := &database.Pod{
			Name:         "test-app",
			Image:        "nginx:latest",
			Replicas:     2,
			Host:         "test.local",
			DesiredState: "running",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if err := db.CreatePod(pod); err != nil {
			t.Fatalf("Failed to create pod: %v", err)
		}

		// Create replicas
		replicas := []*database.Replica{
			{PodName: "test-app", ReplicaID: "test-app-1", ContainerID: "container1", Port: 32001, Status: "running", CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{PodName: "test-app", ReplicaID: "test-app-2", ContainerID: "container2", Port: 32002, Status: "running", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		}

		for _, replica := range replicas {
			if err := db.CreateReplica(replica); err != nil {
				t.Fatalf("Failed to create replica: %v", err)
			}
		}

		// Generate config
		config, err := GenerateConfigFromDatabase(db)
		if err != nil {
			t.Fatalf("Failed to generate config from database: %v", err)
		}

		// Verify config contains expected content
		if !strings.Contains(config, "upstream test-app") {
			t.Errorf("Config should contain upstream block for test-app")
		}
		if !strings.Contains(config, "server localhost:32001") {
			t.Errorf("Config should contain replica on port 32001")
		}
		if !strings.Contains(config, "server localhost:32002") {
			t.Errorf("Config should contain replica on port 32002")
		}
		if !strings.Contains(config, "server_name test.local") {
			t.Errorf("Config should contain server_name test.local")
		}
	})
}

func TestUpdateManagerFromDatabase(t *testing.T) {
	// Create temporary database for testing
	tmpDB := "/tmp/test_nginx_manager_update.db"
	defer os.Remove(tmpDB)

	db, err := database.NewDatabase(tmpDB)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Reset database to clean state
	if err := db.Reset(); err != nil {
		t.Fatalf("Failed to reset database: %v", err)
	}

	// Create test config
	cfg := &config.Config{
		Proxy: config.ProxyConfig{
			Port: 80,
		},
	}

	// Create mock executor and docker client
	mockExecutor := NewMockExecutor()
	dockerClient := docker.NewDockerClient(mockExecutor)
	
	// Create nginx manager
	manager := NewManager(cfg, dockerClient)

	t.Run("update manager from database", func(t *testing.T) {
		// Create test pod
		pod := &database.Pod{
			Name:         "api-app",
			Image:        "node:14",
			Replicas:     1,
			Host:         "api.local",
			DesiredState: "running",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if err := db.CreatePod(pod); err != nil {
			t.Fatalf("Failed to create pod: %v", err)
		}

		// Create replica
		replica := &database.Replica{
			PodName:     "api-app",
			ReplicaID:   "api-app-1",
			ContainerID: "container1",
			Port:        32003,
			Status:      "running",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if err := db.CreateReplica(replica); err != nil {
			t.Fatalf("Failed to create replica: %v", err)
		}

		// Mock nginx not running (so no reload commands)
		mockExecutor.SetOutput("docker ps -q --filter label=dubernetes.component=nginx-proxy", "")

		// Update manager from database
		err := UpdateManagerFromDatabase(manager, db)
		if err != nil {
			t.Fatalf("Failed to update manager from database: %v", err)
		}

		// Verify config file was written
		if _, err := os.Stat(manager.GetConfigPath()); os.IsNotExist(err) {
			t.Errorf("Config file was not created at %s", manager.GetConfigPath())
		}

		// Read and verify config content
		configContent, err := os.ReadFile(manager.GetConfigPath())
		if err != nil {
			t.Fatalf("Failed to read config file: %v", err)
		}

		configStr := string(configContent)
		if !strings.Contains(configStr, "upstream api-app") {
			t.Errorf("Config should contain upstream block for api-app")
		}
		if !strings.Contains(configStr, "server localhost:32003") {
			t.Errorf("Config should contain replica on port 32003")
		}
		if !strings.Contains(configStr, "server_name api.local") {
			t.Errorf("Config should contain server_name api.local")
		}
	})

	t.Run("update manager with nginx running", func(t *testing.T) {
		// Reset mock
		*mockExecutor = *NewMockExecutor()

		// Mock nginx running
		mockExecutor.SetOutput("docker ps -q --filter label=dubernetes.component=nginx-proxy", "nginx-container-id")
		mockExecutor.SetOutput("docker exec nginx-container-id nginx -t", "")
		mockExecutor.SetOutput("docker exec nginx-container-id nginx -s reload", "")

		// Update manager from database
		err := UpdateManagerFromDatabase(manager, db)
		if err != nil {
			t.Fatalf("Failed to update manager from database: %v", err)
		}

		// Verify reload commands were executed
		commands := mockExecutor.GetCommands()
		hasTestCommand := false
		hasReloadCommand := false

		for _, cmd := range commands {
			if strings.Contains(cmd, "nginx -t") {
				hasTestCommand = true
			}
			if strings.Contains(cmd, "nginx -s reload") {
				hasReloadCommand = true
			}
		}

		if !hasTestCommand {
			t.Errorf("Expected nginx -t command to be executed")
		}
		if !hasReloadCommand {
			t.Errorf("Expected nginx -s reload command to be executed")
		}
	})
}