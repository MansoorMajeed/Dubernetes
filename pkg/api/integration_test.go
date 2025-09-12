package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestAPIIntegration tests the full API with real HTTP requests
func TestAPIIntegration(t *testing.T) {
	// Create mock orchestrator
	mockOrch := &MockOrchestrator{}
	
	// Set up mock expectations for the entire test flow
	mockOrch.On("CreatePod", mock.Anything, mock.MatchedBy(func(req PodRequest) bool {
		return req.Name == "test-app" && req.Image == "nginx:latest"
	})).Return(&PodResponse{
		Name:     "test-app",
		Image:    "nginx:latest", 
		Replicas: 2,
		Status:   StatusPending,
		Access: &AccessConfig{
			Host: "test-app.local",
		},
		Instances: []PodInstance{},
	}, nil)
	
	mockOrch.On("GetPod", mock.Anything, "test-app").Return(&PodResponse{
		Name:     "test-app",
		Image:    "nginx:latest",
		Replicas: 2,
		Status:   StatusRunning,
		Access: &AccessConfig{
			Host: "test-app.local",
		},
		Instances: []PodInstance{
			{ID: "container-1", Port: 32001, Status: StatusRunning},
			{ID: "container-2", Port: 32002, Status: StatusRunning},
		},
	}, nil)
	
	mockOrch.On("ListPods", mock.Anything).Return([]PodSummary{
		{
			Name:     "test-app",
			Image:    "nginx:latest",
			Replicas: 2,
			Status:   StatusRunning,
		},
	}, nil)
	
	mockOrch.On("DeletePod", mock.Anything, "test-app").Return(nil)
	
	// Start server
	server := NewServer(":0", mockOrch)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start(ctx)
	}()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	baseURL := fmt.Sprintf("http://localhost%s", server.GetAddr())
	
	// Test 1: Create Pod
	t.Run("CreatePod", func(t *testing.T) {
		podReq := PodRequest{
			Name:     "test-app",
			Image:    "nginx:latest",
			Replicas: 2,
			Access: &AccessConfig{
				Host: "test-app.local",
			},
		}
		
		reqBody, err := json.Marshal(podReq)
		require.NoError(t, err)
		
		resp, err := http.Post(baseURL+"/pods", "application/json", bytes.NewReader(reqBody))
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
		
		var podResp PodResponse
		err = json.NewDecoder(resp.Body).Decode(&podResp)
		require.NoError(t, err)
		
		assert.Equal(t, "test-app", podResp.Name)
		assert.Equal(t, "nginx:latest", podResp.Image)
		assert.Equal(t, 2, podResp.Replicas)
		assert.Equal(t, StatusPending, podResp.Status)
		assert.NotNil(t, podResp.Access)
		assert.Equal(t, "test-app.local", podResp.Access.Host)
	})
	
	// Test 2: Get Pod
	t.Run("GetPod", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/pods/test-app")
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
		
		var podResp PodResponse
		err = json.NewDecoder(resp.Body).Decode(&podResp)
		require.NoError(t, err)
		
		assert.Equal(t, "test-app", podResp.Name)
		assert.Equal(t, "nginx:latest", podResp.Image)
		assert.Equal(t, 2, podResp.Replicas)
		assert.Equal(t, StatusRunning, podResp.Status)
		assert.Len(t, podResp.Instances, 2)
		assert.Equal(t, "container-1", podResp.Instances[0].ID)
		assert.Equal(t, 32001, podResp.Instances[0].Port)
	})
	
	// Test 3: List Pods
	t.Run("ListPods", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/pods")
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
		
		var listResp ListPodsResponse
		err = json.NewDecoder(resp.Body).Decode(&listResp)
		require.NoError(t, err)
		
		assert.Len(t, listResp.Pods, 1)
		assert.Equal(t, "test-app", listResp.Pods[0].Name)
		assert.Equal(t, "nginx:latest", listResp.Pods[0].Image)
		assert.Equal(t, 2, listResp.Pods[0].Replicas)
		assert.Equal(t, StatusRunning, listResp.Pods[0].Status)
	})
	
	// Test 4: Delete Pod
	t.Run("DeletePod", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodDelete, baseURL+"/pods/test-app", nil)
		require.NoError(t, err)
		
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})
	
	// Test 5: Health Endpoint
	t.Run("HealthCheck", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
		
		var healthResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&healthResp)
		require.NoError(t, err)
		
		assert.Equal(t, "healthy", healthResp["status"])
		assert.Equal(t, "dubernetes-api", healthResp["service"])
		assert.NotEmpty(t, healthResp["timestamp"])
	})
	
	// Test 6: Error Handling
	t.Run("ErrorHandling", func(t *testing.T) {
		// Test invalid JSON
		resp, err := http.Post(baseURL+"/pods", "application/json", bytes.NewReader([]byte("invalid json")))
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		
		var errResp ErrorResponse
		err = json.NewDecoder(resp.Body).Decode(&errResp)
		require.NoError(t, err)
		
		assert.Contains(t, errResp.Error, "invalid request body")
		assert.Equal(t, "INVALID_JSON", errResp.Code)
		
		// Test validation error
		invalidPod := PodRequest{
			Name:  "", // Invalid: empty name
			Image: "nginx:latest",
		}
		
		reqBody, err := json.Marshal(invalidPod)
		require.NoError(t, err)
		
		resp, err = http.Post(baseURL+"/pods", "application/json", bytes.NewReader(reqBody))
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		
		err = json.NewDecoder(resp.Body).Decode(&errResp)
		require.NoError(t, err)
		
		assert.Contains(t, errResp.Error, "validation failed")
		assert.Equal(t, "VALIDATION_ERROR", errResp.Code)
	})
	
	// Test 7: CORS Headers
	t.Run("CORS", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodOptions, baseURL+"/pods", nil)
		require.NoError(t, err)
		req.Header.Set("Origin", "http://localhost:3000")
		req.Header.Set("Access-Control-Request-Method", "POST")
		
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		assert.Contains(t, resp.Header.Get("Access-Control-Allow-Methods"), "POST")
		assert.Contains(t, resp.Header.Get("Access-Control-Allow-Headers"), "Content-Type")
	})
	
	// Cleanup
	cancel()
	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Server did not shut down within timeout")
	}
	
	// Verify all mock expectations were met
	mockOrch.AssertExpectations(t)
}

// TestAPINotFound tests 404 error handling
func TestAPINotFound(t *testing.T) {
	mockOrch := &MockOrchestrator{}
	server := NewServer(":0", mockOrch)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start(ctx)
	}()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	baseURL := fmt.Sprintf("http://localhost%s", server.GetAddr())
	
	// Test invalid route
	resp, err := http.Get(baseURL + "/invalid")
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	
	var errResp ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errResp)
	require.NoError(t, err)
	
	assert.Contains(t, errResp.Error, "not found")
	
	// Cleanup
	cancel()
	<-errCh
}

// TestAPIMethodNotAllowed tests 405 error handling
func TestAPIMethodNotAllowed(t *testing.T) {
	mockOrch := &MockOrchestrator{}
	server := NewServer(":0", mockOrch)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start(ctx)
	}()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	baseURL := fmt.Sprintf("http://localhost%s", server.GetAddr())
	
	// Test unsupported method
	req, err := http.NewRequest(http.MethodPatch, baseURL+"/pods", nil)
	require.NoError(t, err)
	
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	
	var errResp ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errResp)
	require.NoError(t, err)
	
	assert.Contains(t, errResp.Error, "method not allowed")
	
	// Cleanup
	cancel()
	<-errCh
}