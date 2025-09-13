package docker

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// MockCommandExecutor implements CommandExecutor for testing
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

func TestNewDockerClient(t *testing.T) {
	executor := NewMockExecutor()
	client := NewDockerClient(executor)
	
	if client == nil {
		t.Fatal("NewDockerClient() returned nil")
	}
}

func TestRunContainer(t *testing.T) {
	tests := []struct {
		name        string
		req         *RunContainerRequest
		mockOutput  string
		mockError   error
		expectedID  string
		expectError bool
	}{
		{
			name: "successful container run",
			req: &RunContainerRequest{
				Image:     "nginx:latest",
				Name:      "test-container",
				Port:      32001,
				Labels:    map[string]string{"app": "test", "replica": "test-1"},
			},
			mockOutput:  "container123\n",
			expectedID:  "container123",
			expectError: false,
		},
		{
			name: "docker run failure",
			req: &RunContainerRequest{
				Image: "invalid:image",
				Name:  "test-container",
				Port:  32001,
			},
			mockError:   errors.New("docker: Error response from daemon"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewMockExecutor()
			client := NewDockerClient(executor)

			// Build expected command
			expectedCmd := fmt.Sprintf("docker run -d --name %s -p %d:80", tt.req.Name, tt.req.Port)
			for k, v := range tt.req.Labels {
				expectedCmd += fmt.Sprintf(" --label %s=%s", k, v)
			}
			expectedCmd += " " + tt.req.Image

			if tt.mockError != nil {
				executor.SetError(expectedCmd, tt.mockError)
			} else {
				executor.SetOutput(expectedCmd, tt.mockOutput)
			}

			containerID, err := client.RunContainer(tt.req)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if containerID != tt.expectedID {
				t.Errorf("Expected container ID %s, got %s", tt.expectedID, containerID)
			}

			// Verify command was called
			commands := executor.GetCommands()
			if len(commands) != 1 || commands[0] != expectedCmd {
				t.Errorf("Expected command %s, got %v", expectedCmd, commands)
			}
		})
	}
}

func TestStopContainer(t *testing.T) {
	tests := []struct {
		name        string
		containerID string
		mockError   error
		expectError bool
	}{
		{
			name:        "successful stop",
			containerID: "container123",
			expectError: false,
		},
		{
			name:        "stop failure",
			containerID: "invalid",
			mockError:   errors.New("docker: Error response from daemon"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewMockExecutor()
			client := NewDockerClient(executor)

			expectedCmd := "docker stop " + tt.containerID
			if tt.mockError != nil {
				executor.SetError(expectedCmd, tt.mockError)
			}

			err := client.StopContainer(tt.containerID)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			commands := executor.GetCommands()
			if len(commands) != 1 || commands[0] != expectedCmd {
				t.Errorf("Expected command %s, got %v", expectedCmd, commands)
			}
		})
	}
}

func TestRemoveContainer(t *testing.T) {
	tests := []struct {
		name        string
		containerID string
		mockError   error
		expectError bool
	}{
		{
			name:        "successful removal",
			containerID: "container123",
			expectError: false,
		},
		{
			name:        "removal failure",
			containerID: "invalid",
			mockError:   errors.New("docker: Error response from daemon"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewMockExecutor()
			client := NewDockerClient(executor)

			expectedCmd := "docker rm -f " + tt.containerID
			if tt.mockError != nil {
				executor.SetError(expectedCmd, tt.mockError)
			}

			err := client.RemoveContainer(tt.containerID)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			commands := executor.GetCommands()
			if len(commands) != 1 || commands[0] != expectedCmd {
				t.Errorf("Expected command %s, got %v", expectedCmd, commands)
			}
		})
	}
}

func TestIsContainerRunning(t *testing.T) {
	tests := []struct {
		name        string
		containerID string
		mockOutput  string
		mockError   error
		expected    bool
		expectError bool
	}{
		{
			name:        "container running",
			containerID: "container123",
			mockOutput:  "running\n",
			expected:    true,
			expectError: false,
		},
		{
			name:        "container stopped",
			containerID: "container123",
			mockOutput:  "exited\n",
			expected:    false,
			expectError: false,
		},
		{
			name:        "container not found",
			containerID: "invalid",
			mockError:   errors.New("docker: Error response from daemon"),
			expected:    false,
			expectError: false, // We treat this as "not running"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewMockExecutor()
			client := NewDockerClient(executor)

			expectedCmd := "docker inspect --format={{.State.Status}} " + tt.containerID
			if tt.mockError != nil {
				executor.SetError(expectedCmd, tt.mockError)
			} else {
				executor.SetOutput(expectedCmd, tt.mockOutput)
			}

			running, err := client.IsContainerRunning(tt.containerID)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if running != tt.expected {
				t.Errorf("Expected running=%v, got %v", tt.expected, running)
			}
		})
	}
}

func TestAllocatePort(t *testing.T) {
	tests := []struct {
		name      string
		usedPorts []int
		portStart int
		portEnd   int
		expected  int
		expectErr bool
	}{
		{
			name:      "allocate first port",
			usedPorts: []int{},
			portStart: 32000,
			portEnd:   32999,
			expected:  32000,
			expectErr: false,
		},
		{
			name:      "allocate next available port",
			usedPorts: []int{32000, 32001, 32003},
			portStart: 32000,
			portEnd:   32999,
			expected:  32002,
			expectErr: false,
		},
		{
			name:      "no ports available",
			usedPorts: []int{32000, 32001},
			portStart: 32000,
			portEnd:   32001,
			expected:  0,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewMockExecutor()
			client := NewDockerClient(executor)

			port, err := client.AllocatePort(tt.usedPorts, tt.portStart, tt.portEnd)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if port != tt.expected {
				t.Errorf("Expected port %d, got %d", tt.expected, port)
			}
		})
	}
}

func TestPortConflictDetection(t *testing.T) {
	executor := NewMockExecutor()
	client := NewDockerClient(executor)

	// Test detecting conflicts
	usedPorts := []int{32000, 32001, 32005}
	
	// Test that used ports are detected as conflicts
	for _, port := range usedPorts {
		if !client.IsPortInUse(port, usedPorts) {
			t.Errorf("Port %d should be detected as in use", port)
		}
	}

	// Test that unused ports are not detected as conflicts
	unusedPorts := []int{32002, 32003, 32004, 32006}
	for _, port := range unusedPorts {
		if client.IsPortInUse(port, usedPorts) {
			t.Errorf("Port %d should not be detected as in use", port)
		}
	}
}

func TestDockerLabels(t *testing.T) {
	executor := NewMockExecutor()
	client := NewDockerClient(executor)

	req := &RunContainerRequest{
		Image: "nginx:latest",
		Name:  "test-container",
		Port:  32001,
		Labels: map[string]string{
			"dubernetes.pod":     "test-app",
			"dubernetes.replica": "test-app-1",
			"dubernetes.managed": "true",
		},
	}

	executor.SetOutput("docker run -d --name test-container -p 32001:80 --label dubernetes.pod=test-app --label dubernetes.replica=test-app-1 --label dubernetes.managed=true nginx:latest", "container123")

	containerID, err := client.RunContainer(req)
	if err != nil {
		t.Fatalf("RunContainer failed: %v", err)
	}

	if containerID != "container123" {
		t.Errorf("Expected container ID container123, got %s", containerID)
	}

	// Verify the command included all labels
	commands := executor.GetCommands()
	if len(commands) != 1 {
		t.Fatalf("Expected 1 command, got %d", len(commands))
	}

	cmd := commands[0]
	for k, v := range req.Labels {
		expectedLabel := fmt.Sprintf("--label %s=%s", k, v)
		if !strings.Contains(cmd, expectedLabel) {
			t.Errorf("Command should contain label %s, got: %s", expectedLabel, cmd)
		}
	}
}

func TestGetContainerIP(t *testing.T) {
	tests := []struct {
		name          string
		containerID   string
		expectedIP    string
		mockOutput    string
		mockError     error
		wantErr       bool
	}{
		{
			name:        "successful IP retrieval",
			containerID: "container123",
			expectedIP:  "172.17.0.2",
			mockOutput:  "172.17.0.2",
			wantErr:     false,
		},
		{
			name:        "container not found",
			containerID: "nonexistent",
			mockError:   errors.New("No such container"),
			wantErr:     true,
		},
		{
			name:        "empty IP address",
			containerID: "container456",
			mockOutput:  "",
			wantErr:     true,
		},
		{
			name:        "IP with whitespace",
			containerID: "container789",
			expectedIP:  "172.17.0.3",
			mockOutput:  "  172.17.0.3  \n",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewMockExecutor()
			client := NewDockerClient(executor)

			expectedCmd := fmt.Sprintf("docker inspect --format={{.NetworkSettings.IPAddress}} %s", tt.containerID)
			
			if tt.mockError != nil {
				executor.SetError(expectedCmd, tt.mockError)
			} else {
				executor.SetOutput(expectedCmd, tt.mockOutput)
			}

			ip, err := client.GetContainerIP(tt.containerID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetContainerIP() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("GetContainerIP() unexpected error: %v", err)
				return
			}

			if ip != tt.expectedIP {
				t.Errorf("GetContainerIP() = %s, want %s", ip, tt.expectedIP)
			}

			// Verify correct command was executed
			commands := executor.GetCommands()
			if len(commands) != 1 {
				t.Fatalf("Expected 1 command, got %d", len(commands))
			}

			if commands[0] != expectedCmd {
				t.Errorf("Expected command %s, got %s", expectedCmd, commands[0])
			}
		})
	}
}