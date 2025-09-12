package api

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Status constants for pods and instances
const (
	StatusPending = "pending"
	StatusRunning = "running"
	StatusStopped = "stopped"
	StatusFailed  = "failed"
)

// PodRequest represents a request to create or update a pod
type PodRequest struct {
	Name     string        `json:"name" yaml:"name"`
	Image    string        `json:"image" yaml:"image"`
	Replicas int           `json:"replicas,omitempty" yaml:"replicas,omitempty"`
	Access   *AccessConfig `json:"access,omitempty" yaml:"access,omitempty"`
}

// AccessConfig defines ingress configuration for a pod
type AccessConfig struct {
	Host string `json:"host" yaml:"host"`
}

// PodResponse represents the full details of a pod
type PodResponse struct {
	Name      string        `json:"name"`
	Image     string        `json:"image"`
	Replicas  int           `json:"replicas"`
	Status    string        `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
	Access    *AccessConfig `json:"access,omitempty"`
	Instances []PodInstance `json:"instances"`
}

// PodInstance represents a single running container instance
type PodInstance struct {
	ID     string `json:"id"`
	Port   int    `json:"port"`
	Status string `json:"status"`
}

// PodSummary represents basic pod information for list operations
type PodSummary struct {
	Name     string `json:"name"`
	Image    string `json:"image"`
	Replicas int    `json:"replicas"`
	Status   string `json:"status"`
}

// ListPodsResponse represents the response for listing pods
type ListPodsResponse struct {
	Pods []PodSummary `json:"pods"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// Validate validates a PodRequest
func (r *PodRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}

	if r.Image == "" {
		return fmt.Errorf("image is required")
	}

	// Validate name format (lowercase alphanumeric with hyphens)
	nameRegex := regexp.MustCompile(`^[a-z0-9-]+$`)
	if !nameRegex.MatchString(r.Name) {
		return fmt.Errorf("name must be lowercase alphanumeric with hyphens")
	}

	if r.Replicas < 0 {
		return fmt.Errorf("replicas must be >= 0")
	}

	if r.Access != nil {
		if err := r.Access.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// SetDefaults sets default values for a PodRequest
func (r *PodRequest) SetDefaults() {
	if r.Replicas == 0 {
		r.Replicas = 1
	}
}

// Validate validates an AccessConfig
func (a *AccessConfig) Validate() error {
	if a.Host == "" {
		return fmt.Errorf("host is required")
	}

	// Check for spaces
	if strings.Contains(a.Host, " ") {
		return fmt.Errorf("host must be a valid hostname")
	}

	// Check for uppercase
	if a.Host != strings.ToLower(a.Host) {
		return fmt.Errorf("host must be lowercase")
	}

	// Basic hostname validation (alphanumeric, dots, hyphens)
	hostnameRegex := regexp.MustCompile(`^[a-z0-9.-]+$`)
	if !hostnameRegex.MatchString(a.Host) {
		return fmt.Errorf("host must be a valid hostname")
	}

	return nil
}