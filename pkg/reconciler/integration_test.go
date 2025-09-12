package reconciler

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mansoormajeed/dubernetes/pkg/config"
	"github.com/mansoormajeed/dubernetes/pkg/database"
	"github.com/mansoormajeed/dubernetes/pkg/docker"
	"github.com/mansoormajeed/dubernetes/pkg/nginx"
)

func TestReconcilerIntegration(t *testing.T) {
	// Create temporary database for testing
	tmpDB := "/tmp/test_reconciler_integration.db"
	defer os.Remove(tmpDB)

	// Create database
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
		Reconciler: config.ReconcilerConfig{
			Interval: 100 * time.Millisecond, // Very fast for testing
		},
		Containers: config.ContainerConfig{
			PortRangeStart: 32000,
			PortRangeEnd:   32999,
		},
		Proxy: config.ProxyConfig{
			Port: 80,
		},
		Orchestrator: config.OrchestratorConfig{
			Port: 8080,
			Host: "localhost",
		},
	}

	// Create mock executor for docker operations
	mockExecutor := &IntegrationMockExecutor{
		containers: make(map[string]ContainerInfo),
		nextID:     1,
	}
	dockerClient := docker.NewDockerClient(mockExecutor)

	// Create nginx manager
	nginxManager := nginx.NewManager(cfg, dockerClient)

	// Create reconciler
	reconciler := NewReconciler(cfg, db, dockerClient, nginxManager)

	t.Run("complete workflow: Database -> Reconciler -> Nginx", func(t *testing.T) {
		// Step 1: Create pod directly in database
		pod := &database.Pod{
			Name:         "integration-test",
			Image:        "nginx:latest",
			Replicas:     2,
			Host:         "test.local",
			DesiredState: "running",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		err := db.CreatePod(pod)
		if err != nil {
			t.Fatalf("Failed to create pod in database: %v", err)
		}

		// Step 2: Run reconciliation to create containers
		err = reconciler.ReconcileOnce()
		if err != nil {
			t.Fatalf("Reconciliation failed: %v", err)
		}

		// Verify replicas were created
		replicas, err := db.ListReplicasForPod("integration-test")
		if err != nil {
			t.Fatalf("Failed to list replicas: %v", err)
		}

		if len(replicas) != 2 {
			t.Errorf("Expected 2 replicas, got %d", len(replicas))
		}

		// Verify containers were created in mock
		runningContainers := mockExecutor.GetRunningContainers()
		if len(runningContainers) != 2 {
			t.Errorf("Expected 2 running containers, got %d", len(runningContainers))
		}

		// Verify nginx config was updated
		configPath := nginxManager.GetConfigPath()
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Errorf("Nginx config file was not created")
		}

		// Read and verify nginx config content
		configContent, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("Failed to read nginx config: %v", err)
		}

		configStr := string(configContent)
		if configStr == "" {
			t.Errorf("Nginx config should not be empty")
		}

		// Step 3: Scale pod up by updating database
		pod.Replicas = 3
		err = db.UpdatePod(pod)
		if err != nil {
			t.Fatalf("Failed to update pod in database: %v", err)
		}

		// Run reconciliation again
		err = reconciler.ReconcileOnce()
		if err != nil {
			t.Fatalf("Reconciliation after scale failed: %v", err)
		}

		// Verify scaled replicas
		replicas, err = db.ListReplicasForPod("integration-test")
		if err != nil {
			t.Fatalf("Failed to list replicas after scale: %v", err)
		}

		if len(replicas) != 3 {
			t.Errorf("Expected 3 replicas after scale, got %d", len(replicas))
		}

		runningContainers = mockExecutor.GetRunningContainers()
		if len(runningContainers) != 3 {
			t.Errorf("Expected 3 running containers after scale, got %d", len(runningContainers))
		}

		// Step 4: Delete pod by updating desired state
		pod.DesiredState = "stopped"
		err = db.UpdatePod(pod)
		if err != nil {
			t.Fatalf("Failed to update pod desired state: %v", err)
		}

		// Run reconciliation to clean up
		err = reconciler.ReconcileOnce()
		if err != nil {
			t.Fatalf("Reconciliation after delete failed: %v", err)
		}

		// Verify containers were stopped
		runningContainers = mockExecutor.GetRunningContainers()
		if len(runningContainers) != 0 {
			t.Errorf("Expected 0 running containers after delete, got %d", len(runningContainers))
		}
	})

	t.Run("container failure and restart workflow", func(t *testing.T) {
		// Reset for this test
		if err := db.Reset(); err != nil {
			t.Fatalf("Failed to reset database: %v", err)
		}
		mockExecutor.Reset()

		// Create pod directly in database
		pod := &database.Pod{
			Name:         "restart-test",
			Image:        "nginx:latest",
			Replicas:     1,
			Host:         "restart.local",
			DesiredState: "running",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		err := db.CreatePod(pod)
		if err != nil {
			t.Fatalf("Failed to create pod in database: %v", err)
		}

		// Run reconciliation to create container
		err = reconciler.ReconcileOnce()
		if err != nil {
			t.Fatalf("Initial reconciliation failed: %v", err)
		}

		// Verify container was created
		runningContainers := mockExecutor.GetRunningContainers()
		if len(runningContainers) != 1 {
			t.Fatalf("Expected 1 running container, got %d", len(runningContainers))
		}

		// Simulate container failure
		for containerID := range runningContainers {
			mockExecutor.StopContainer(containerID)
		}

		// Wait a bit for backoff timing
		time.Sleep(10 * time.Millisecond)

		// Run reconciliation to detect and restart failed container
		err = reconciler.ReconcileOnce()
		if err != nil {
			t.Fatalf("Reconciliation after container failure failed: %v", err)
		}

		// Verify container was restarted
		replicas, err := db.ListReplicasForPod("restart-test")
		if err != nil {
			t.Fatalf("Failed to list replicas: %v", err)
		}

		if len(replicas) != 1 {
			t.Fatalf("Expected 1 replica, got %d", len(replicas))
		}

		replica := replicas[0]
		if replica.RestartCount == 0 {
			t.Errorf("Expected restart count > 0, got %d", replica.RestartCount)
		}

		// Verify new container is running
		runningContainers = mockExecutor.GetRunningContainers()
		if len(runningContainers) != 1 {
			t.Errorf("Expected 1 running container after restart, got %d", len(runningContainers))
		}
	})
}

// IntegrationMockExecutor is a more sophisticated mock for integration testing
type IntegrationMockExecutor struct {
	containers map[string]ContainerInfo
	nextID     int
}

type ContainerInfo struct {
	ID      string
	Name    string
	Status  string
	Port    int
	Labels  map[string]string
	Image   string
}

func (m *IntegrationMockExecutor) Execute(command string, args ...string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("no args provided")
	}

	switch args[0] {
	case "run":
		return m.handleRun(args)
	case "stop":
		return m.handleStop(args)
	case "rm":
		return m.handleRemove(args)
	case "inspect":
		return m.handleInspect(args)
	case "ps":
		return m.handlePs(args)
	case "exec":
		return m.handleExec(args)
	default:
		return "", nil
	}
}

func (m *IntegrationMockExecutor) handleRun(args []string) (string, error) {
	containerID := fmt.Sprintf("container-%d", m.nextID)
	m.nextID++

	// Parse run command
	name := ""
	port := 0
	labels := make(map[string]string)
	image := ""

	for i, arg := range args {
		if arg == "--name" && i+1 < len(args) {
			name = args[i+1]
		} else if arg == "-p" && i+1 < len(args) {
			// Parse port mapping like "32001:80"
			var hostPort, containerPort int
			fmt.Sscanf(args[i+1], "%d:%d", &hostPort, &containerPort)
			port = hostPort
		} else if arg == "--label" && i+1 < len(args) {
			// Parse label like "key=value"
			var key, value string
			fmt.Sscanf(args[i+1], "%[^=]=%s", &key, &value)
			labels[key] = value
		} else if i == len(args)-1 && !strings.HasPrefix(arg, "-") {
			image = arg
		}
	}

	m.containers[containerID] = ContainerInfo{
		ID:     containerID,
		Name:   name,
		Status: "running",
		Port:   port,
		Labels: labels,
		Image:  image,
	}

	return containerID, nil
}

func (m *IntegrationMockExecutor) handleStop(args []string) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("container ID required")
	}
	containerID := args[1]
	if container, exists := m.containers[containerID]; exists {
		container.Status = "exited"
		m.containers[containerID] = container
	}
	return "", nil
}

func (m *IntegrationMockExecutor) handleRemove(args []string) (string, error) {
	if len(args) < 3 {
		return "", fmt.Errorf("container ID required")
	}
	containerID := args[2] // -f containerID
	delete(m.containers, containerID)
	return "", nil
}

func (m *IntegrationMockExecutor) handleInspect(args []string) (string, error) {
	if len(args) < 4 {
		return "", fmt.Errorf("container ID required")
	}
	containerID := args[3] // --format={{.State.Status}} containerID
	if container, exists := m.containers[containerID]; exists {
		return container.Status, nil
	}
	return "", fmt.Errorf("container not found")
}

func (m *IntegrationMockExecutor) handlePs(args []string) (string, error) {
	// Return container IDs that match the filter
	var result []string
	for _, container := range m.containers {
		if container.Status == "running" {
			// Simple filter matching for nginx-proxy
			if len(args) > 3 && args[3] == "label=dubernetes.component=nginx-proxy" {
				if container.Labels["dubernetes.component"] == "nginx-proxy" {
					result = append(result, container.ID)
				}
			} else {
				result = append(result, container.ID)
			}
		}
	}
	return "", nil // Empty result for simplicity
}

func (m *IntegrationMockExecutor) handleExec(args []string) (string, error) {
	// Mock successful exec commands
	return "", nil
}

func (m *IntegrationMockExecutor) GetRunningContainers() map[string]ContainerInfo {
	running := make(map[string]ContainerInfo)
	for id, container := range m.containers {
		if container.Status == "running" {
			running[id] = container
		}
	}
	return running
}

func (m *IntegrationMockExecutor) StopContainer(containerID string) {
	if container, exists := m.containers[containerID]; exists {
		container.Status = "exited"
		m.containers[containerID] = container
	}
}

func (m *IntegrationMockExecutor) Reset() {
	m.containers = make(map[string]ContainerInfo)
	m.nextID = 1
}