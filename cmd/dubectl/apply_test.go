package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestApplyCommand(t *testing.T) {
	// Create test YAML file
	yamlContent := `name: test-app
image: nginx:latest
replicas: 2
access:
  host: test.local
`

	tmpFile, err := os.CreateTemp("", "test-pod-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(yamlContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	tests := []struct {
		name           string
		args           []string
		serverResponse string
		serverStatus   int
		expectError    bool
		expectOutput   string
	}{
		{
			name:           "successful apply",
			args:           []string{"apply", "-f", tmpFile.Name()},
			serverResponse: `{"name":"test-app","image":"nginx:latest","replicas":2,"access":{"host":"test.local"},"status":"running","created_at":"2024-01-01T00:00:00Z"}`,
			serverStatus:   http.StatusCreated,
			expectError:    false,
			expectOutput:   "pod/test-app created",
		},
		{
			name:           "server error",
			args:           []string{"apply", "-f", tmpFile.Name()},
			serverResponse: `{"error":"Internal server error"}`,
			serverStatus:   http.StatusInternalServerError,
			expectError:    true,
			expectOutput:   "server returned status 500",
		},
		{
			name:         "file not found",
			args:         []string{"apply", "-f", "nonexistent.yaml"},
			serverStatus: 0, // No server for this test
			expectError:  true,
			expectOutput: "no such file or directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server if needed
			var server *httptest.Server
			if tt.serverStatus != 0 {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Verify request method and path
					if r.Method != "POST" || r.URL.Path != "/pods" {
						t.Errorf("Expected POST /pods, got %s %s", r.Method, r.URL.Path)
					}

					// Verify content type
					if r.Header.Get("Content-Type") != "application/json" {
						t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
					}

					w.WriteHeader(tt.serverStatus)
					w.Write([]byte(tt.serverResponse))
				}))
				defer server.Close()
			}

			// Create fresh command for each test
			cmd := &cobra.Command{
				Use: "dubectl",
			}
			cmd.AddCommand(applyCmd)
			cmd.PersistentFlags().StringP("server", "s", "http://localhost:8080", "Server URL")
			cmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose output")

			// Set args
			args := tt.args
			if server != nil {
				args = append([]string{"--server", server.URL}, tt.args...)
			} else if tt.serverStatus == 0 {
				// For tests without a server, use default localhost URL
				args = append([]string{"--server", "http://localhost:8080"}, tt.args...)
			}
			cmd.SetArgs(args)

			// Capture output
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stdout = w
			os.Stderr = w

			// Execute command
			err := cmd.Execute()

			// Restore stdout/stderr
			w.Close()
			os.Stdout = oldStdout
			os.Stderr = oldStderr

			// Read output
			output, _ := io.ReadAll(r)
			outputStr := string(output)

			// Check results
			if tt.expectError && err == nil {
				t.Errorf("Expected error but command succeeded")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected success but got error: %v", err)
			}
			if !strings.Contains(outputStr, tt.expectOutput) {
				t.Errorf("Expected output to contain '%s', got: %s", tt.expectOutput, outputStr)
			}
		})
	}
}

func TestParseYAMLFile(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		expectError bool
		expectName  string
		expectImage string
	}{
		{
			name: "valid yaml",
			yamlContent: `name: test-app
image: nginx:latest
replicas: 2
access:
  host: test.local
`,
			expectError: false,
			expectName:  "test-app",
			expectImage: "nginx:latest",
		},
		{
			name: "minimal yaml",
			yamlContent: `name: minimal-app
image: redis:latest
`,
			expectError: false,
			expectName:  "minimal-app",
			expectImage: "redis:latest",
		},
		{
			name: "invalid yaml",
			yamlContent: `name: test-app
image: nginx:latest
invalid_field: [unclosed
`,
			expectError: true,
		},
		{
			name: "missing required fields",
			yamlContent: `replicas: 2
`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpFile, err := os.CreateTemp("", "test-*.yaml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(tt.yamlContent); err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			tmpFile.Close()

			// Test parsing
			podSpec, err := parseYAMLFile(tmpFile.Name())

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but parsing succeeded")
				}
				return
			}

			if err != nil {
				t.Errorf("Expected success but got error: %v", err)
				return
			}

			if podSpec.Name != tt.expectName {
				t.Errorf("Expected name %s, got %s", tt.expectName, podSpec.Name)
			}

			if podSpec.Image != tt.expectImage {
				t.Errorf("Expected image %s, got %s", tt.expectImage, podSpec.Image)
			}
		})
	}
}

func TestAPIClient(t *testing.T) {
	// Test API client functionality
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/pods":
			if r.Method == "POST" {
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"name":"test-app","status":"running"}`))
			} else if r.Method == "GET" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`[{"name":"test-app","status":"running"}]`))
			}
		case "/pods/test-app":
			if r.Method == "DELETE" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"message":"Pod deleted"}`))
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := &APIClient{
		BaseURL: server.URL,
		Client:  &http.Client{},
	}

	t.Run("create pod", func(t *testing.T) {
		podSpec := &PodSpec{
			Name:     "test-app",
			Image:    "nginx:latest",
			Replicas: 1,
		}

		err := client.CreatePod(podSpec)
		if err != nil {
			t.Errorf("Failed to create pod: %v", err)
		}
	})

	t.Run("list pods", func(t *testing.T) {
		pods, err := client.ListPods()
		if err != nil {
			t.Errorf("Failed to list pods: %v", err)
		}
		if len(pods) == 0 {
			t.Errorf("Expected at least one pod")
		}
	})

	t.Run("delete pod", func(t *testing.T) {
		err := client.DeletePod("test-app")
		if err != nil {
			t.Errorf("Failed to delete pod: %v", err)
		}
	})
}