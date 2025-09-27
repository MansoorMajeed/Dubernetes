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

func TestGetCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		serverResponse string
		serverStatus   int
		expectError    bool
		expectOutput   string
	}{
		{
			name:           "list all pods",
			args:           []string{"get", "pods"},
			serverResponse: `[{"name":"app1","image":"nginx:latest","replicas":2,"status":"running"},{"name":"app2","image":"redis:latest","replicas":1,"status":"running"}]`,
			serverStatus:   http.StatusOK,
			expectError:    false,
			expectOutput:   "app1",
		},
		{
			name:           "get specific pod",
			args:           []string{"get", "pods", "app1"},
			serverResponse: `{"name":"app1","image":"nginx:latest","replicas":2,"status":"running","created_at":"2024-01-01T00:00:00Z"}`,
			serverStatus:   http.StatusOK,
			expectError:    false,
			expectOutput:   "Name:       app1",
		},
		{
			name:           "pod not found",
			args:           []string{"get", "pods", "nonexistent"},
			serverResponse: `{"error":"Pod not found"}`,
			serverStatus:   http.StatusNotFound,
			expectError:    true,
			expectOutput:   "not found",
		},
		{
			name:           "empty pod list",
			args:           []string{"get", "pods"},
			serverResponse: `[]`,
			serverStatus:   http.StatusOK,
			expectError:    false,
			expectOutput:   "No pods found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != "GET" {
					t.Errorf("Expected GET, got %s", r.Method)
				}

				// Verify path - if we have 3 args, it's get pods <name>
				if len(tt.args) >= 3 && tt.args[0] == "get" && tt.args[1] == "pods" {
					expectedPath := "/pods/" + tt.args[2]
					if r.URL.Path != expectedPath {
						t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
					}
				} else {
					// Otherwise it should be just /pods
					if r.URL.Path != "/pods" {
						t.Errorf("Expected path /pods, got %s", r.URL.Path)
					}
				}

				w.WriteHeader(tt.serverStatus)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			// Create fresh command for each test
			cmd := &cobra.Command{
				Use: "dubectl",
			}
			cmd.AddCommand(getCmd)
			cmd.PersistentFlags().StringP("server", "s", "http://localhost:8080", "Server URL")
			cmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose output")

			// Set args with server URL
			args := append([]string{"--server", server.URL}, tt.args...)
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