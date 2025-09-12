package api

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
	mockOrch := &MockOrchestrator{}
	
	server := NewServer(":8080", mockOrch)
	
	assert.NotNil(t, server)
	assert.Equal(t, ":8080", server.addr)
	assert.NotNil(t, server.handler)
	assert.NotNil(t, server.server)
}

func TestServer_Start(t *testing.T) {
	mockOrch := &MockOrchestrator{}
	// Set up mock to handle ListPods call
	mockOrch.On("ListPods", mock.Anything).Return([]PodSummary{}, nil)
	
	server := NewServer(":0", mockOrch) // Use :0 for random available port
	
	// Start server in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start(ctx)
	}()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	// Verify server is running by making a request
	resp, err := http.Get(fmt.Sprintf("http://localhost%s/pods", server.GetAddr()))
	require.NoError(t, err)
	resp.Body.Close()
	
	// Server should be running
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	// Cancel context to stop server
	cancel()
	
	// Wait for server to shutdown
	select {
	case err := <-errCh:
		// Server should shut down gracefully without error
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Server did not shut down within timeout")
	}
}

func TestServer_StartAlreadyRunning(t *testing.T) {
	mockOrch := &MockOrchestrator{}
	server := NewServer(":0", mockOrch)
	
	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	
	// Start first server
	errCh1 := make(chan error, 1)
	go func() {
		errCh1 <- server.Start(ctx1)
	}()
	
	// Give first server time to start
	time.Sleep(100 * time.Millisecond)
	
	// Try to start second server on same address - should fail
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	
	server2 := NewServer(server.GetAddr(), mockOrch)
	err := server2.Start(ctx2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "address already in use")
	
	// Clean up first server
	cancel1()
	<-errCh1
}

func TestServer_GracefulShutdown(t *testing.T) {
	mockOrch := &MockOrchestrator{}
	// Set up mock to handle ListPods call
	mockOrch.On("ListPods", mock.Anything).Return([]PodSummary{}, nil)
	
	server := NewServer(":0", mockOrch)
	
	ctx, cancel := context.WithCancel(context.Background())
	
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start(ctx)
	}()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	// Verify server is running
	resp, err := http.Get(fmt.Sprintf("http://localhost%s/pods", server.GetAddr()))
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	// Cancel context to trigger shutdown
	shutdownStart := time.Now()
	cancel()
	
	// Wait for server to shutdown
	select {
	case err := <-errCh:
		shutdownDuration := time.Since(shutdownStart)
		assert.NoError(t, err)
		// Should shutdown quickly for test
		assert.Less(t, shutdownDuration, 2*time.Second)
	case <-time.After(5 * time.Second):
		t.Fatal("Server did not shut down within timeout")
	}
}

func TestServer_HealthEndpoint(t *testing.T) {
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
	
	// Test health endpoint
	resp, err := http.Get(fmt.Sprintf("http://localhost%s/health", server.GetAddr()))
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	
	// Cleanup
	cancel()
	<-errCh
}

func TestServer_CORS(t *testing.T) {
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
	
	// Test CORS headers
	req, _ := http.NewRequest("OPTIONS", fmt.Sprintf("http://localhost%s/pods", server.GetAddr()), nil)
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
	
	// Cleanup
	cancel()
	<-errCh
}

func TestServer_RequestLogging(t *testing.T) {
	mockOrch := &MockOrchestrator{}
	// Set up mock to handle ListPods call
	mockOrch.On("ListPods", mock.Anything).Return([]PodSummary{}, nil)
	
	server := NewServer(":0", mockOrch)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start(ctx)
	}()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	// Make a request to verify logging middleware is working
	resp, err := http.Get(fmt.Sprintf("http://localhost%s/pods", server.GetAddr()))
	require.NoError(t, err)
	resp.Body.Close()
	
	// Just verify the request completed successfully
	// In a real implementation, we might capture logs and verify them
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	// Cleanup
	cancel()
	<-errCh
}

func TestServer_GetAddr(t *testing.T) {
	mockOrch := &MockOrchestrator{}
	
	tests := []struct {
		name     string
		addr     string
		expected string
	}{
		{
			name:     "explicit port",
			addr:     ":8080",
			expected: ":8080",
		},
		{
			name:     "random port",
			addr:     ":0",
			expected: ":0", // Before starting
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(tt.addr, mockOrch)
			assert.Equal(t, tt.expected, server.GetAddr())
		})
	}
}

func TestServer_GetAddrAfterStart(t *testing.T) {
	mockOrch := &MockOrchestrator{}
	server := NewServer(":0", mockOrch) // Random port
	
	// Before starting, should return original address
	assert.Equal(t, ":0", server.GetAddr())
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start(ctx)
	}()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	// After starting, should return actual port
	actualAddr := server.GetAddr()
	assert.NotEqual(t, ":0", actualAddr)
	assert.Contains(t, actualAddr, ":")
	
	// Cleanup
	cancel()
	<-errCh
}