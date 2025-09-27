package docker

import (
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// CommandExecutor interface allows mocking of command execution
type CommandExecutor interface {
	Execute(command string, args ...string) (string, error)
}

// RealCommandExecutor implements CommandExecutor using actual system commands
type RealCommandExecutor struct{}

func (r *RealCommandExecutor) Execute(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// DockerClient wraps Docker operations using system commands
type DockerClient struct {
	executor CommandExecutor
}

// RunContainerRequest contains parameters for running a container
type RunContainerRequest struct {
	Image  string            `json:"image"`
	Name   string            `json:"name"`
	Port   int               `json:"port"`
	Labels map[string]string `json:"labels,omitempty"`
}

// NewDockerClient creates a new Docker client with real command execution
func NewDockerClient(executor CommandExecutor) *DockerClient {
	if executor == nil {
		executor = &RealCommandExecutor{}
	}
	return &DockerClient{
		executor: executor,
	}
}

// RunContainer starts a new Docker container
func (c *DockerClient) RunContainer(req *RunContainerRequest) (string, error) {
	args := []string{
		"run", "-d",
		"--name", req.Name,
		"-p", fmt.Sprintf("%d:80", req.Port),
	}

	// Add labels
	for key, value := range req.Labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", key, value))
	}

	// Add image as the last argument
	args = append(args, req.Image)

	output, err := c.executor.Execute("docker", args...)
	if err != nil {
		return "", fmt.Errorf("failed to run container: %w", err)
	}

	// Docker returns the container ID followed by a newline
	containerID := strings.TrimSpace(output)
	return containerID, nil
}

// StopContainer stops a running container
func (c *DockerClient) StopContainer(containerID string) error {
	_, err := c.executor.Execute("docker", "stop", containerID)
	if err != nil {
		return fmt.Errorf("failed to stop container %s: %w", containerID, err)
	}
	return nil
}

// RemoveContainer removes a container (forcefully)
func (c *DockerClient) RemoveContainer(containerID string) error {
	_, err := c.executor.Execute("docker", "rm", "-f", containerID)
	if err != nil {
		return fmt.Errorf("failed to remove container %s: %w", containerID, err)
	}
	return nil
}

// IsContainerRunning checks if a container is currently running
func (c *DockerClient) IsContainerRunning(containerID string) (bool, error) {
	output, err := c.executor.Execute("docker", "inspect", "--format={{.State.Status}}", containerID)
	if err != nil {
		// If container doesn't exist or other error, consider it not running
		return false, nil
	}

	status := strings.TrimSpace(output)
	return status == "running", nil
}

// AllocatePort finds an available port from the given range
func (c *DockerClient) AllocatePort(usedPorts []int, portStart, portEnd int) (int, error) {
	// Get ports actually in use by Docker containers
	dockerPorts, err := c.GetUsedPortsFromContainers()
	if err != nil {
		return 0, fmt.Errorf("failed to get used ports from containers: %w", err)
	}

	// Combine database ports and actual Docker ports
	allUsedPorts := append(usedPorts, dockerPorts...)

	// Remove duplicates and sort
	portMap := make(map[int]bool)
	for _, port := range allUsedPorts {
		portMap[port] = true
	}

	var uniquePorts []int
	for port := range portMap {
		uniquePorts = append(uniquePorts, port)
	}
	sort.Ints(uniquePorts)

	for port := portStart; port <= portEnd; port++ {
		if !c.IsPortInUse(port, uniquePorts) {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available ports in range %d-%d", portStart, portEnd)
}

// IsPortInUse checks if a port is in the list of used ports
func (c *DockerClient) IsPortInUse(port int, usedPorts []int) bool {
	// Binary search since usedPorts should be sorted
	index := sort.SearchInts(usedPorts, port)
	return index < len(usedPorts) && usedPorts[index] == port
}

// GetContainerStatus returns the status of a container
func (c *DockerClient) GetContainerStatus(containerID string) (string, error) {
	output, err := c.executor.Execute("docker", "inspect", "--format={{.State.Status}}", containerID)
	if err != nil {
		return "", fmt.Errorf("failed to get container status: %w", err)
	}

	return strings.TrimSpace(output), nil
}

// ListContainersByLabel lists containers with specific labels
func (c *DockerClient) ListContainersByLabel(labelFilter string) ([]string, error) {
	output, err := c.executor.Execute("docker", "ps", "-q", "--filter", fmt.Sprintf("label=%s", labelFilter))
	if err != nil {
		return nil, fmt.Errorf("failed to list containers by label: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	var containerIDs []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			containerIDs = append(containerIDs, line)
		}
	}

	return containerIDs, nil
}

// GetContainerPort returns the host port mapped to container port 80
func (c *DockerClient) GetContainerPort(containerID string) (int, error) {
	output, err := c.executor.Execute("docker", "port", containerID, "80")
	if err != nil {
		return 0, fmt.Errorf("failed to get container port: %w", err)
	}

	// Parse output like "0.0.0.0:32001"
	parts := strings.Split(strings.TrimSpace(output), ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("unexpected port output format: %s", output)
	}

	port := 0
	if _, err := fmt.Sscanf(parts[1], "%d", &port); err != nil {
		return 0, fmt.Errorf("failed to parse port number: %w", err)
	}

	return port, nil
}

// RestartContainer restarts a container
func (c *DockerClient) RestartContainer(containerID string) error {
	_, err := c.executor.Execute("docker", "restart", containerID)
	if err != nil {
		return fmt.Errorf("failed to restart container %s: %w", containerID, err)
	}
	return nil
}

// ExecContainer executes a command inside a running container
func (c *DockerClient) ExecContainer(containerID string, command ...string) (string, error) {
	args := append([]string{"exec", containerID}, command...)
	output, err := c.executor.Execute("docker", args...)
	if err != nil {
		return "", fmt.Errorf("failed to exec command in container %s: %w", containerID, err)
	}
	return output, nil
}

// RunContainerWithOptions runs a container with custom options
func (c *DockerClient) RunContainerWithOptions(image string, options []string) (string, error) {
	args := append([]string{"run"}, options...)
	args = append(args, image)
	
	output, err := c.executor.Execute("docker", args...)
	if err != nil {
		return "", fmt.Errorf("failed to run container with options: %w", err)
	}
	
	// Docker returns the container ID followed by a newline
	containerID := strings.TrimSpace(output)
	return containerID, nil
}

// GetContainerIP retrieves the IP address of a container
func (c *DockerClient) GetContainerIP(containerID string) (string, error) {
	output, err := c.executor.Execute("docker", "inspect", "--format={{.NetworkSettings.IPAddress}}", containerID)
	if err != nil {
		return "", fmt.Errorf("failed to get container IP: %w", err)
	}
	
	ip := strings.TrimSpace(output)
	if ip == "" {
		return "", fmt.Errorf("container %s has no IP address (may not be running)", containerID)
	}
	
	return ip, nil
}

// GetUsedPortsFromContainers returns a list of ports actually in use by running containers
func (c *DockerClient) GetUsedPortsFromContainers() ([]int, error) {
	// Get all running containers with port mappings
	output, err := c.executor.Execute("docker", "ps", "--format", "{{.Ports}}")
	if err != nil {
		return nil, fmt.Errorf("failed to list container ports: %w", err)
	}

	var usedPorts []int
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Parse port mappings like "0.0.0.0:32000->80/tcp, [::]:32000->80/tcp"
		// Extract host ports using regex
		re := regexp.MustCompile(`0\.0\.0\.0:(\d+)->`)
		matches := re.FindAllStringSubmatch(line, -1)

		for _, match := range matches {
			if len(match) > 1 {
				if port, err := strconv.Atoi(match[1]); err == nil {
					usedPorts = append(usedPorts, port)
				}
			}
		}
	}

	return usedPorts, nil
}