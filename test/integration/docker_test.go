package integration

import (
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/mansoormajeed/dubernetes/pkg/docker"
)

func TestDockerIntegration(t *testing.T) {
	// Skip if Docker is not available
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping integration tests")
	}

	client := docker.NewDockerClient(nil) // Use real executor

	// Test container lifecycle
	req := &docker.RunContainerRequest{
		Image: "nginx:alpine",
		Name:  "dubernetes-test-integration",
		Port:  32090,
		Labels: map[string]string{
			"dubernetes.test": "integration",
			"dubernetes.pod":  "test-app",
		},
	}

	// Clean up any existing test container
	defer func() {
		client.RemoveContainer(req.Name)
	}()

	// Run container
	containerID, err := client.RunContainer(req)
	if err != nil {
		t.Fatalf("Failed to run container: %v", err)
	}

	if containerID == "" {
		t.Fatal("Container ID should not be empty")
	}

	// Wait a moment for container to start
	time.Sleep(2 * time.Second)

	// Check if container is running
	isRunning, err := client.IsContainerRunning(containerID)
	if err != nil {
		t.Fatalf("Failed to check container status: %v", err)
	}

	if !isRunning {
		t.Error("Container should be running")
	}

	// Get container status
	status, err := client.GetContainerStatus(containerID)
	if err != nil {
		t.Fatalf("Failed to get container status: %v", err)
	}

	if status != "running" {
		t.Errorf("Expected status 'running', got '%s'", status)
	}

	// Test port accessibility (optional - requires network testing)
	port, err := client.GetContainerPort(containerID)
	if err != nil {
		t.Fatalf("Failed to get container port: %v", err)
	}

	if port != req.Port {
		t.Errorf("Expected port %d, got %d", req.Port, port)
	}

	// Stop container
	err = client.StopContainer(containerID)
	if err != nil {
		t.Fatalf("Failed to stop container: %v", err)
	}

	// Wait for container to stop
	time.Sleep(1 * time.Second)

	// Check if container is stopped
	isRunning, err = client.IsContainerRunning(containerID)
	if err != nil {
		t.Fatalf("Failed to check container status after stop: %v", err)
	}

	if isRunning {
		t.Error("Container should not be running after stop")
	}

	// Remove container
	err = client.RemoveContainer(containerID)
	if err != nil {
		t.Fatalf("Failed to remove container: %v", err)
	}

	// Verify container is removed
	isRunning, err = client.IsContainerRunning(containerID)
	if err != nil {
		t.Fatalf("Failed to check container status after removal: %v", err)
	}

	if isRunning {
		t.Error("Container should not exist after removal")
	}
}

func TestDockerPortAllocation(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping integration tests")
	}

	client := docker.NewDockerClient(nil)

	// Test port allocation with real Docker containers
	usedPorts := []int{32100, 32102, 32104}

	// Allocate a port
	port, err := client.AllocatePort(usedPorts, 32100, 32110)
	if err != nil {
		t.Fatalf("Failed to allocate port: %v", err)
	}

	// Should get the first available port (32101)
	if port != 32101 {
		t.Errorf("Expected port 32101, got %d", port)
	}

	// Test running multiple containers with different ports
	containers := []string{}
	defer func() {
		// Clean up all test containers
		for _, containerName := range containers {
			client.RemoveContainer(containerName)
		}
	}()

	// Run containers on sequential ports
	for i := 0; i < 3; i++ {
		containerName := fmt.Sprintf("dubernetes-test-port-%d", i)
		containers = append(containers, containerName)

		// Get current used ports (simulate database query)
		currentUsedPorts := []int{}
		for j := 0; j < i; j++ {
			currentUsedPorts = append(currentUsedPorts, 32200+j)
		}

		allocatedPort, err := client.AllocatePort(currentUsedPorts, 32200, 32299)
		if err != nil {
			t.Fatalf("Failed to allocate port for container %d: %v", i, err)
		}

		req := &docker.RunContainerRequest{
			Image: "nginx:alpine",
			Name:  containerName,
			Port:  allocatedPort,
			Labels: map[string]string{
				"dubernetes.test": "port-allocation",
			},
		}

		_, err = client.RunContainer(req)
		if err != nil {
			t.Fatalf("Failed to run container %s: %v", containerName, err)
		}

		// Verify the port is as expected
		expectedPort := 32200 + i
		if allocatedPort != expectedPort {
			t.Errorf("Container %d: expected port %d, got %d", i, expectedPort, allocatedPort)
		}
	}
}

func TestDockerContainerRestart(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping integration tests")
	}

	client := docker.NewDockerClient(nil)

	req := &docker.RunContainerRequest{
		Image: "nginx:alpine",
		Name:  "dubernetes-test-restart",
		Port:  32095,
		Labels: map[string]string{
			"dubernetes.test": "restart",
		},
	}

	defer client.RemoveContainer(req.Name)

	// Run container
	containerID, err := client.RunContainer(req)
	if err != nil {
		t.Fatalf("Failed to run container: %v", err)
	}

	// Wait for container to start
	time.Sleep(2 * time.Second)

	// Restart container
	err = client.RestartContainer(containerID)
	if err != nil {
		t.Fatalf("Failed to restart container: %v", err)
	}

	// Wait for restart to complete
	time.Sleep(2 * time.Second)

	// Verify container is still running
	isRunning, err := client.IsContainerRunning(containerID)
	if err != nil {
		t.Fatalf("Failed to check container status after restart: %v", err)
	}

	if !isRunning {
		t.Error("Container should be running after restart")
	}
}

func TestDockerListByLabel(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping integration tests")
	}

	client := docker.NewDockerClient(nil)

	// Run a few test containers with labels
	containers := []string{
		"dubernetes-test-label-1",
		"dubernetes-test-label-2",
	}

	defer func() {
		for _, name := range containers {
			client.RemoveContainer(name)
		}
	}()

	// Run containers with specific labels
	for i, name := range containers {
		req := &docker.RunContainerRequest{
			Image: "nginx:alpine",
			Name:  name,
			Port:  32300 + i,
			Labels: map[string]string{
				"dubernetes.test": "label-test",
				"dubernetes.pod":  "test-app",
			},
		}

		_, err := client.RunContainer(req)
		if err != nil {
			t.Fatalf("Failed to run container %s: %v", name, err)
		}
	}

	// Wait for containers to start
	time.Sleep(2 * time.Second)

	// List containers by label
	foundContainers, err := client.ListContainersByLabel("dubernetes.test=label-test")
	if err != nil {
		t.Fatalf("Failed to list containers by label: %v", err)
	}

	if len(foundContainers) < 2 {
		t.Errorf("Expected at least 2 containers, found %d", len(foundContainers))
	}
}

// isDockerAvailable checks if Docker is available on the system
func isDockerAvailable() bool {
	cmd := exec.Command("docker", "version")
	err := cmd.Run()
	return err == nil
}