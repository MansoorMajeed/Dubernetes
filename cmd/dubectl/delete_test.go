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

func TestDeleteCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		serverResponse string
		serverStatus   int
		expectError    bool
		expectOutput   string
	}{
		{
			name:           "delete single pod",
			args:           []string{"delete", "pods", "app1"},
			serverResponse: `{"message":"Pod deleted"}`,
			serverStatus:   http.StatusOK,
			expectError:    false,
			expectOutput:   "pod/app1 deleted",
		},
		{
			name:           "delete multiple pods",
			args:           []string{"delete", "pods", "app1", "app2"},
			serverResponse: `{"message":"Pod deleted"}`,
			serverStatus:   http.StatusOK,
			expectError:    false,
			expectOutput:   "pod/app1 deleted",
		},
		{
			name:           "pod not found",
			args:           []string{"delete", "pods", "nonexistent"},
			serverResponse: `{"error":"Pod not found"}`,
			serverStatus:   http.StatusNotFound,
			expectError:    true,
			expectOutput:   "not found",
		},
		{
			name:        "no pod names provided",
			args:        []string{"delete", "pods"},
			expectError: true,
			expectOutput: "requires at least 1 arg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server if needed
			var server *httptest.Server
			if tt.serverStatus != 0 {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Verify request method
					if r.Method != "DELETE" {
						t.Errorf("Expected DELETE, got %s", r.Method)
					}

					// Verify path contains pod name
					if !strings.Contains(r.URL.Path, "/pods/") {
						t.Errorf("Expected path to contain /pods/, got %s", r.URL.Path)
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
			cmd.AddCommand(deleteCmd)
			cmd.PersistentFlags().StringP("server", "s", "http://localhost:8080", "Server URL")
			cmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose output")

			// Set args
			args := tt.args
			if server != nil {
				args = append([]string{"--server", server.URL}, tt.args...)
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