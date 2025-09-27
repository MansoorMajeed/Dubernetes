package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"gopkg.in/yaml.v2"
)

// PodSpec represents the YAML specification for a pod
type PodSpec struct {
	Name     string      `yaml:"name" json:"name"`
	Image    string      `yaml:"image" json:"image"`
	Replicas int         `yaml:"replicas,omitempty" json:"replicas,omitempty"`
	Access   *AccessSpec `yaml:"access,omitempty" json:"access,omitempty"`
}

// AccessSpec represents the access configuration for a pod
type AccessSpec struct {
	Host string `yaml:"host" json:"host"`
}

// PodResponse represents the API response for a pod
type PodResponse struct {
	Name      string      `json:"name"`
	Image     string      `json:"image"`
	Replicas  int         `json:"replicas"`
	Access    *AccessSpec `json:"access,omitempty"`
	Status    string      `json:"status"`
	CreatedAt string      `json:"created_at"`
}

// PodSummary represents a simplified pod view for listing
type PodSummary struct {
	Name     string `json:"name"`
	Image    string `json:"image"`
	Replicas int    `json:"replicas"`
	Status   string `json:"status"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// APIClient handles communication with the Dubernetes orchestrator API
type APIClient struct {
	BaseURL string
	Client  *http.Client
}

// NewAPIClient creates a new API client
func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		BaseURL: baseURL,
		Client:  &http.Client{},
	}
}

// CreatePod creates a new pod via the API
func (c *APIClient) CreatePod(podSpec *PodSpec) error {
	// Convert PodSpec to API request format
	requestData := map[string]interface{}{
		"name":     podSpec.Name,
		"image":    podSpec.Image,
		"replicas": podSpec.Replicas,
	}

	if podSpec.Access != nil {
		requestData["access"] = map[string]interface{}{
			"host": podSpec.Access.Host,
		}
	}

	// Default replicas to 1 if not specified
	if podSpec.Replicas == 0 {
		requestData["replicas"] = 1
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return fmt.Errorf("failed to marshal pod spec: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/pods", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}

// ListPodsResponse represents the API response for listing pods
type ListPodsResponse struct {
	Pods []PodSummary `json:"pods"`
}

// ListPods retrieves all pods from the API
func (c *APIClient) ListPods() ([]PodSummary, error) {
	req, err := http.NewRequest("GET", c.BaseURL+"/pods", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var response ListPodsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Pods, nil
}

// GetPod retrieves a specific pod from the API
func (c *APIClient) GetPod(name string) (*PodResponse, error) {
	req, err := http.NewRequest("GET", c.BaseURL+"/pods/"+name, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("pod '%s' not found", name)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var pod PodResponse
	if err := json.NewDecoder(resp.Body).Decode(&pod); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &pod, nil
}

// DeletePod deletes a pod via the API
func (c *APIClient) DeletePod(name string) error {
	req, err := http.NewRequest("DELETE", c.BaseURL+"/pods/"+name, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("pod '%s' not found", name)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}

// parseYAMLFile parses a YAML file and returns a PodSpec
func parseYAMLFile(filename string) (*PodSpec, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var podSpec PodSpec
	if err := yaml.Unmarshal(data, &podSpec); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate required fields
	if podSpec.Name == "" {
		return nil, fmt.Errorf("pod name is required")
	}

	if podSpec.Image == "" {
		return nil, fmt.Errorf("pod image is required")
	}

	// Set default replicas if not specified
	if podSpec.Replicas == 0 {
		podSpec.Replicas = 1
	}

	return &podSpec, nil
}