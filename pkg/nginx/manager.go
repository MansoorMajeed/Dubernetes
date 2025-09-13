package nginx

import (
	"fmt"
	"os"

	"github.com/mansoormajeed/dubernetes/pkg/config"
	"github.com/mansoormajeed/dubernetes/pkg/docker"
)

// Manager handles nginx container lifecycle and configuration
type Manager struct {
	config     *config.Config
	docker     *docker.DockerClient
	configPath string
}

// NewManager creates a new nginx manager
func NewManager(cfg *config.Config, dockerClient *docker.DockerClient) *Manager {
	configPath := cfg.Proxy.ConfigPath
	if configPath == "" {
		configPath = "/tmp/dubernetes-nginx.conf"
	}
	return &Manager{
		config:     cfg,
		docker:     dockerClient,
		configPath: configPath,
	}
}

// Start starts the nginx container with the current configuration
func (m *Manager) Start() error {
	// Create a basic nginx config if none exists
	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		defaultConfig := `events {
    worker_connections 1024;
}

http {
    upstream default {
        server 127.0.0.1:8080;
    }
    
    server {
        listen 80;
        location / {
            return 404;
        }
    }
}
`
		if err := os.WriteFile(m.configPath, []byte(defaultConfig), 0644); err != nil {
			return fmt.Errorf("failed to create default nginx config: %w", err)
		}
	}

	// Use the new RunContainerWithOptions method
	options := []string{
		"-d",
		"--name", "dubernetes-nginx",
		"-p", fmt.Sprintf("%d:80", m.config.Proxy.Port),
		"--label", "dubernetes.component=nginx-proxy",
		"--restart=unless-stopped",
		"-v", fmt.Sprintf("%s:/etc/nginx/nginx.conf", m.configPath),
	}

	containerID, err := m.docker.RunContainerWithOptions("nginx:latest", options)
	if err != nil {
		return fmt.Errorf("failed to start nginx container: %w", err)
	}

	if containerID == "" {
		return fmt.Errorf("failed to get container ID from docker run")
	}

	return nil
}

// Stop stops and removes the nginx container
func (m *Manager) Stop() error {
	// Find nginx containers
	containerIDs, err := m.docker.ListContainersByLabel("dubernetes.component=nginx-proxy")
	if err != nil {
		return fmt.Errorf("failed to list nginx containers: %w", err)
	}

	for _, containerID := range containerIDs {
		// Stop container
		if err := m.docker.StopContainer(containerID); err != nil {
			return fmt.Errorf("failed to stop nginx container %s: %w", containerID, err)
		}

		// Remove container
		if err := m.docker.RemoveContainer(containerID); err != nil {
			return fmt.Errorf("failed to remove nginx container %s: %w", containerID, err)
		}
	}

	return nil
}

// IsRunning checks if nginx container is currently running
func (m *Manager) IsRunning() (bool, error) {
	containerIDs, err := m.docker.ListContainersByLabel("dubernetes.component=nginx-proxy")
	if err != nil {
		return false, fmt.Errorf("failed to list nginx containers: %w", err)
	}

	return len(containerIDs) > 0, nil
}

// Restart stops and starts the nginx container
func (m *Manager) Restart() error {
	if err := m.Stop(); err != nil {
		return fmt.Errorf("failed to stop nginx during restart: %w", err)
	}

	if err := m.Start(); err != nil {
		return fmt.Errorf("failed to start nginx during restart: %w", err)
	}

	return nil
}

// UpdateConfig updates the nginx configuration and reloads nginx if running
func (m *Manager) UpdateConfig(nginxConfig string) error {
	// Validate config first
	if err := ValidateNginxConfig(nginxConfig); err != nil {
		return fmt.Errorf("invalid nginx config: %w", err)
	}

	// Write config to file
	if err := os.WriteFile(m.configPath, []byte(nginxConfig), 0644); err != nil {
		return fmt.Errorf("failed to write nginx config: %w", err)
	}

	// Check if nginx is running
	isRunning, err := m.IsRunning()
	if err != nil {
		return fmt.Errorf("failed to check if nginx is running: %w", err)
	}

	// If nginx is not running, start it
	if !isRunning {
		if err := m.Start(); err != nil {
			return fmt.Errorf("failed to start nginx: %w", err)
		}
		return nil // Config is already loaded when starting
	}

	// If nginx is running, test and reload the config
	if isRunning {
		containerIDs, err := m.docker.ListContainersByLabel("dubernetes.component=nginx-proxy")
		if err != nil {
			return fmt.Errorf("failed to find nginx container: %w", err)
		}

		if len(containerIDs) > 0 {
			containerID := containerIDs[0]

			// Test config first
			if _, err := m.docker.ExecContainer(containerID, "nginx", "-t"); err != nil {
				return fmt.Errorf("nginx config test failed: %w", err)
			}

			// Reload nginx
			if _, err := m.docker.ExecContainer(containerID, "nginx", "-s", "reload"); err != nil {
				return fmt.Errorf("failed to reload nginx: %w", err)
			}
		}
	}

	return nil
}

// GetConfigPath returns the path to the nginx configuration file
func (m *Manager) GetConfigPath() string {
	return m.configPath
}