package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Common errors
var (
	ErrPodNotFound = errors.New("pod not found")
)

// Orchestrator defines the interface for pod management operations
type Orchestrator interface {
	CreatePod(ctx context.Context, req PodRequest) (*PodResponse, error)
	GetPod(ctx context.Context, name string) (*PodResponse, error)
	ListPods(ctx context.Context) ([]PodSummary, error)
	DeletePod(ctx context.Context, name string) error
}

// Handler handles HTTP requests for the API
type Handler struct {
	orchestrator Orchestrator
}

// NewHandler creates a new API handler
func NewHandler(orchestrator Orchestrator) *Handler {
	return &Handler{
		orchestrator: orchestrator,
	}
}

// ServeHTTP implements http.Handler interface
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set content type for all responses
	w.Header().Set("Content-Type", "application/json")

	// Route based on path and method
	path := strings.TrimPrefix(r.URL.Path, "/")
	pathParts := strings.Split(path, "/")

	if len(pathParts) < 1 || pathParts[0] != "pods" {
		h.writeError(w, http.StatusNotFound, "not found", "ROUTE_NOT_FOUND", "")
		return
	}

	switch r.Method {
	case http.MethodPost:
		if len(pathParts) == 1 && pathParts[0] == "pods" {
			h.createPod(w, r)
		} else {
			h.writeError(w, http.StatusNotFound, "not found", "ROUTE_NOT_FOUND", "")
		}
	case http.MethodGet:
		if len(pathParts) == 1 && pathParts[0] == "pods" {
			h.listPods(w, r)
		} else if len(pathParts) == 2 && pathParts[0] == "pods" {
			h.getPod(w, r, pathParts[1])
		} else {
			h.writeError(w, http.StatusNotFound, "not found", "ROUTE_NOT_FOUND", "")
		}
	case http.MethodDelete:
		if len(pathParts) == 2 && pathParts[0] == "pods" {
			h.deletePod(w, r, pathParts[1])
		} else {
			h.writeError(w, http.StatusNotFound, "not found", "ROUTE_NOT_FOUND", "")
		}
	default:
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed", "METHOD_NOT_ALLOWED", "")
	}
}

// createPod handles POST /pods
func (h *Handler) createPod(w http.ResponseWriter, r *http.Request) {
	var req PodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body", "INVALID_JSON", err.Error())
		return
	}

	// Set defaults
	req.SetDefaults()

	// Validate request
	if err := req.Validate(); err != nil {
		h.writeError(w, http.StatusBadRequest, "validation failed", "VALIDATION_ERROR", err.Error())
		return
	}

	// Create pod
	pod, err := h.orchestrator.CreatePod(r.Context(), req)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to create pod", "CREATE_FAILED", err.Error())
		return
	}

	// Return created pod
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(pod)
}

// getPod handles GET /pods/{name}
func (h *Handler) getPod(w http.ResponseWriter, r *http.Request, name string) {
	pod, err := h.orchestrator.GetPod(r.Context(), name)
	if err != nil {
		if errors.Is(err, ErrPodNotFound) {
			h.writeError(w, http.StatusNotFound, "pod not found", "POD_NOT_FOUND", fmt.Sprintf("pod '%s' not found", name))
			return
		}
		h.writeError(w, http.StatusInternalServerError, "failed to get pod", "GET_FAILED", err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(pod)
}

// listPods handles GET /pods
func (h *Handler) listPods(w http.ResponseWriter, r *http.Request) {
	pods, err := h.orchestrator.ListPods(r.Context())
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to list pods", "LIST_FAILED", err.Error())
		return
	}

	response := ListPodsResponse{
		Pods: pods,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// deletePod handles DELETE /pods/{name}
func (h *Handler) deletePod(w http.ResponseWriter, r *http.Request, name string) {
	err := h.orchestrator.DeletePod(r.Context(), name)
	if err != nil {
		if errors.Is(err, ErrPodNotFound) {
			h.writeError(w, http.StatusNotFound, "pod not found", "POD_NOT_FOUND", fmt.Sprintf("pod '%s' not found", name))
			return
		}
		h.writeError(w, http.StatusInternalServerError, "failed to delete pod", "DELETE_FAILED", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// writeError writes an error response
func (h *Handler) writeError(w http.ResponseWriter, statusCode int, message, code, details string) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   message,
		Code:    code,
		Details: details,
	})
}