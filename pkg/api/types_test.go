package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPodRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     PodRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid pod request",
			req: PodRequest{
				Name:     "test-pod",
				Image:    "nginx:latest",
				Replicas: 2,
				Access: &AccessConfig{
					Host: "app.local",
				},
			},
			wantErr: false,
		},
		{
			name: "valid pod request minimal",
			req: PodRequest{
				Name:  "test-pod",
				Image: "nginx:latest",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			req: PodRequest{
				Image: "nginx:latest",
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "missing image",
			req: PodRequest{
				Name: "test-pod",
			},
			wantErr: true,
			errMsg:  "image is required",
		},
		{
			name: "invalid name",
			req: PodRequest{
				Name:  "Test Pod",
				Image: "nginx:latest",
			},
			wantErr: true,
			errMsg:  "name must be lowercase alphanumeric with hyphens",
		},
		{
			name: "negative replicas",
			req: PodRequest{
				Name:     "test-pod",
				Image:    "nginx:latest",
				Replicas: -1,
			},
			wantErr: true,
			errMsg:  "replicas must be >= 0",
		},
		{
			name: "invalid host",
			req: PodRequest{
				Name:     "test-pod",
				Image:    "nginx:latest",
				Replicas: 1,
				Access: &AccessConfig{
					Host: "invalid host name",
				},
			},
			wantErr: true,
			errMsg:  "host must be a valid hostname",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPodRequest_DefaultValues(t *testing.T) {
	req := PodRequest{
		Name:  "test-pod",
		Image: "nginx:latest",
	}

	req.SetDefaults()

	assert.Equal(t, 1, req.Replicas)
}

func TestPodResponse_Structure(t *testing.T) {
	now := time.Now()
	
	resp := PodResponse{
		Name:      "test-pod",
		Image:     "nginx:latest",
		Replicas:  2,
		Status:    "running",
		CreatedAt: now,
		Access: &AccessConfig{
			Host: "app.local",
		},
		Instances: []PodInstance{
			{
				ID:     "container-1",
				Port:   32001,
				Status: "running",
			},
			{
				ID:     "container-2",
				Port:   32002,
				Status: "running",
			},
		},
	}

	assert.Equal(t, "test-pod", resp.Name)
	assert.Equal(t, "nginx:latest", resp.Image)
	assert.Equal(t, 2, resp.Replicas)
	assert.Equal(t, "running", resp.Status)
	assert.Equal(t, now, resp.CreatedAt)
	assert.NotNil(t, resp.Access)
	assert.Equal(t, "app.local", resp.Access.Host)
	assert.Len(t, resp.Instances, 2)
	assert.Equal(t, "container-1", resp.Instances[0].ID)
	assert.Equal(t, 32001, resp.Instances[0].Port)
}

func TestListPodsResponse_Structure(t *testing.T) {
	resp := ListPodsResponse{
		Pods: []PodSummary{
			{
				Name:     "pod-1",
				Image:    "nginx:latest",
				Replicas: 2,
				Status:   "running",
			},
			{
				Name:     "pod-2",
				Image:    "redis:alpine",
				Replicas: 1,
				Status:   "pending",
			},
		},
	}

	assert.Len(t, resp.Pods, 2)
	assert.Equal(t, "pod-1", resp.Pods[0].Name)
	assert.Equal(t, "nginx:latest", resp.Pods[0].Image)
	assert.Equal(t, 2, resp.Pods[0].Replicas)
	assert.Equal(t, "running", resp.Pods[0].Status)
}

func TestErrorResponse_Structure(t *testing.T) {
	resp := ErrorResponse{
		Error:   "validation failed",
		Code:    "VALIDATION_ERROR",
		Details: "name is required",
	}

	assert.Equal(t, "validation failed", resp.Error)
	assert.Equal(t, "VALIDATION_ERROR", resp.Code)
	assert.Equal(t, "name is required", resp.Details)
}

func TestAccessConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  AccessConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid hostname",
			config: AccessConfig{
				Host: "app.local",
			},
			wantErr: false,
		},
		{
			name: "valid subdomain",
			config: AccessConfig{
				Host: "api.app.local",
			},
			wantErr: false,
		},
		{
			name: "empty host",
			config: AccessConfig{
				Host: "",
			},
			wantErr: true,
			errMsg:  "host is required",
		},
		{
			name: "invalid hostname with spaces",
			config: AccessConfig{
				Host: "app local",
			},
			wantErr: true,
			errMsg:  "host must be a valid hostname",
		},
		{
			name: "invalid hostname with uppercase",
			config: AccessConfig{
				Host: "App.Local",
			},
			wantErr: true,
			errMsg:  "host must be lowercase",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPodInstance_Structure(t *testing.T) {
	instance := PodInstance{
		ID:     "container-abc123",
		Port:   32001,
		Status: "running",
	}

	assert.Equal(t, "container-abc123", instance.ID)
	assert.Equal(t, 32001, instance.Port)
	assert.Equal(t, "running", instance.Status)
}

func TestPodSummary_Structure(t *testing.T) {
	summary := PodSummary{
		Name:     "test-pod",
		Image:    "nginx:latest",
		Replicas: 3,
		Status:   "running",
	}

	assert.Equal(t, "test-pod", summary.Name)
	assert.Equal(t, "nginx:latest", summary.Image)
	assert.Equal(t, 3, summary.Replicas)
	assert.Equal(t, "running", summary.Status)
}

func TestStatusConstants(t *testing.T) {
	// Test that status constants are defined
	assert.NotEmpty(t, StatusPending)
	assert.NotEmpty(t, StatusRunning)
	assert.NotEmpty(t, StatusStopped)
	assert.NotEmpty(t, StatusFailed)
	
	// Test expected values
	assert.Equal(t, "pending", StatusPending)
	assert.Equal(t, "running", StatusRunning)
	assert.Equal(t, "stopped", StatusStopped)
	assert.Equal(t, "failed", StatusFailed)
}