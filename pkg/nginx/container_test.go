package nginx

import (
	"os"
	"strings"
	"testing"

	"github.com/mansoormajeed/dubernetes/pkg/config"
	"github.com/mansoormajeed/dubernetes/pkg/docker"
)

// MockCommandExecutor implements docker.CommandExecutor for testing
type MockCommandExecutor struct {
	commands []string
	outputs  map[string]string
	errors   map[string]error
}

func NewMockExecutor() *MockCommandExecutor {
	return &MockCommandExecutor{
		commands: []string{},
		outputs:  make(map[string]string),
		errors:   make(map[string]error),
	}
}

func (m *MockCommandExecutor) Execute(command string, args ...string) (string, error) {
	fullCmd := command + " " + strings.Join(args, " ")
	m.commands = append(m.commands, fullCmd)
	
	if err, exists := m.errors[fullCmd]; exists {
		return "", err
	}
	
	if output, exists := m.outputs[fullCmd]; exists {
		return output, nil
	}
	
	return "", nil
}

func (m *MockCommandExecutor) SetOutput(command string, output string) {
	m.outputs[command] = output
}

func (m *MockCommandExecutor) SetError(command string, err error) {
	m.errors[command] = err
}

func (m *MockCommandExecutor) GetCommands() []string {
	return m.commands
}

func TestNginxManager(t *testing.T) {
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

	t.Run("start nginx container", func(t *testing.T) {
		// Mock docker run command
		mockExecutor.SetOutput("docker run -d --name dubernetes-nginx -p 80:80 --label dubernetes.component=nginx-proxy --restart=unless-stopped -v /tmp/dubernetes-nginx.conf:/etc/nginx/nginx.conf nginx:latest", "nginx-container-id")

		// Test starting nginx
		err := manager.Start()
		if err != nil {
			t.Fatalf("Failed to start nginx: %v", err)
		}

		// Verify command was executed
		commands := mockExecutor.GetCommands()
		if len(commands) == 0 {
			t.Fatalf("Expected docker command to be executed")
		}

		if !strings.Contains(commands[0], "docker run") {
			t.Errorf("Expected docker run command, got: %s", commands[0])
		}
	})

	t.Run("stop nginx container", func(t *testing.T) {
		// Reset mock
		*mockExecutor = *NewMockExecutor()

		// Mock docker ps command to find nginx container
		mockExecutor.SetOutput("docker ps -q --filter label=dubernetes.component=nginx-proxy", "nginx-container-id")
		// Mock docker stop command
		mockExecutor.SetOutput("docker stop nginx-container-id", "")
		// Mock docker rm command
		mockExecutor.SetOutput("docker rm -f nginx-container-id", "")

		// Test stopping nginx
		err := manager.Stop()
		if err != nil {
			t.Fatalf("Failed to stop nginx: %v", err)
		}

		// Verify commands were executed
		commands := mockExecutor.GetCommands()
		hasListCommand := false
		hasStopCommand := false
		hasRemoveCommand := false

		for _, cmd := range commands {
			if strings.Contains(cmd, "docker ps") {
				hasListCommand = true
			}
			if strings.Contains(cmd, "docker stop") {
				hasStopCommand = true
			}
			if strings.Contains(cmd, "docker rm") {
				hasRemoveCommand = true
			}
		}

		if !hasListCommand {
			t.Errorf("Expected docker ps command to be executed")
		}
		if !hasStopCommand {
			t.Errorf("Expected docker stop command to be executed")
		}
		if !hasRemoveCommand {
			t.Errorf("Expected docker rm command to be executed")
		}
	})

	t.Run("check if nginx is running", func(t *testing.T) {
		// Reset mock
		*mockExecutor = *NewMockExecutor()

		// Test when nginx is running
		mockExecutor.SetOutput("docker ps -q --filter label=dubernetes.component=nginx-proxy", "nginx-container-id")

		isRunning, err := manager.IsRunning()
		if err != nil {
			t.Fatalf("Failed to check if nginx is running: %v", err)
		}
		if !isRunning {
			t.Errorf("Expected nginx to be running, but it wasn't")
		}

		// Reset and test when nginx is not running
		*mockExecutor = *NewMockExecutor()
		mockExecutor.SetOutput("docker ps -q --filter label=dubernetes.component=nginx-proxy", "")

		isRunning, err = manager.IsRunning()
		if err != nil {
			t.Fatalf("Failed to check if nginx is running: %v", err)
		}
		if isRunning {
			t.Errorf("Expected nginx to not be running, but it was")
		}
	})

	t.Run("restart nginx container", func(t *testing.T) {
		// Reset mock
		*mockExecutor = *NewMockExecutor()

		// Mock stop commands
		mockExecutor.SetOutput("docker ps -q --filter label=dubernetes.component=nginx-proxy", "nginx-container-id")
		mockExecutor.SetOutput("docker stop nginx-container-id", "")
		mockExecutor.SetOutput("docker rm -f nginx-container-id", "")
		// Mock start command
		mockExecutor.SetOutput("docker run -d --name dubernetes-nginx -p 80:80 --label dubernetes.component=nginx-proxy --restart=unless-stopped -v /tmp/dubernetes-nginx.conf:/etc/nginx/nginx.conf nginx:latest", "new-nginx-container-id")

		// Test restarting nginx
		err := manager.Restart()
		if err != nil {
			t.Fatalf("Failed to restart nginx: %v", err)
		}

		// Verify commands were executed (stop and start)
		commands := mockExecutor.GetCommands()
		if len(commands) < 4 {
			t.Errorf("Expected at least 4 commands (list, stop, remove, run), got %d", len(commands))
		}
	})
}

func TestNginxConfigUpdate(t *testing.T) {
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

	t.Run("update nginx config", func(t *testing.T) {
		nginxConfig := `upstream test-app {
    server localhost:32001;
}

server {
    listen 80;
    server_name test.local;
    
    location / {
        proxy_pass http://test-app;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
`

		// Mock nginx reload command
		mockExecutor.SetOutput("docker ps -q --filter label=dubernetes.component=nginx-proxy", "nginx-container-id")
		mockExecutor.SetOutput("docker exec nginx-container-id nginx -t", "")
		mockExecutor.SetOutput("docker exec nginx-container-id nginx -s reload", "")

		// Test updating config
		err := manager.UpdateConfig(nginxConfig)
		if err != nil {
			t.Fatalf("Failed to update nginx config: %v", err)
		}

		// Verify config file was written
		if _, err := os.Stat(manager.GetConfigPath()); os.IsNotExist(err) {
			t.Errorf("Config file was not created at %s", manager.GetConfigPath())
		}

		// Verify reload command was executed
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

	t.Run("update config when nginx not running", func(t *testing.T) {
		// Reset mock
		*mockExecutor = *NewMockExecutor()

		nginxConfig := `server {
    listen 80;
    return 404;
}
`

		// Mock nginx not running
		mockExecutor.SetOutput("docker ps -q --filter label=dubernetes.component=nginx-proxy", "")

		// Mock docker run command for starting nginx
		mockExecutor.SetOutput("docker run -d --name dubernetes-nginx -p 80:80 --label dubernetes.component=nginx-proxy --restart=unless-stopped -v /tmp/dubernetes-nginx.conf:/etc/nginx/nginx.conf nginx:latest", "nginx-container-123")

		// Test updating config when nginx is not running
		err := manager.UpdateConfig(nginxConfig)
		if err != nil {
			t.Fatalf("Failed to update nginx config when not running: %v", err)
		}

		// Should still write config file but not reload
		if _, err := os.Stat(manager.GetConfigPath()); os.IsNotExist(err) {
			t.Errorf("Config file was not created at %s", manager.GetConfigPath())
		}

		// Verify no reload commands were executed
		commands := mockExecutor.GetCommands()
		for _, cmd := range commands {
			if strings.Contains(cmd, "nginx -s reload") {
				t.Errorf("Should not execute reload command when nginx is not running")
			}
		}
	})

	t.Run("update with invalid config", func(t *testing.T) {
		// Reset mock
		*mockExecutor = *NewMockExecutor()

		invalidConfig := `upstream test {
    server localhost:32001
}` // Missing semicolon

		// Test updating with invalid config
		err := manager.UpdateConfig(invalidConfig)
		if err == nil {
			t.Errorf("Expected error for invalid config, but got nil")
		}

		// Verify no docker commands were executed
		commands := mockExecutor.GetCommands()
		if len(commands) > 0 {
			t.Errorf("Expected no docker commands for invalid config, but got: %v", commands)
		}
	})
}