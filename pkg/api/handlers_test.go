package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockOrchestrator is a mock implementation of the Orchestrator interface
type MockOrchestrator struct {
	mock.Mock
}

func (m *MockOrchestrator) CreatePod(ctx context.Context, req PodRequest) (*PodResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PodResponse), args.Error(1)
}

func (m *MockOrchestrator) GetPod(ctx context.Context, name string) (*PodResponse, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PodResponse), args.Error(1)
}

func (m *MockOrchestrator) ListPods(ctx context.Context) ([]PodSummary, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]PodSummary), args.Error(1)
}

func (m *MockOrchestrator) DeletePod(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func TestHandler_CreatePod(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockOrchestrator)
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name: "successful pod creation",
			requestBody: PodRequest{
				Name:     "test-pod",
				Image:    "nginx:latest",
				Replicas: 2,
			},
			mockSetup: func(m *MockOrchestrator) {
				expectedReq := PodRequest{
					Name:     "test-pod",
					Image:    "nginx:latest",
					Replicas: 2,
				}
				m.On("CreatePod", mock.Anything, expectedReq).Return(&PodResponse{
					Name:     "test-pod",
					Image:    "nginx:latest",
					Replicas: 2,
					Status:   StatusPending,
				}, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody: PodResponse{
				Name:     "test-pod",
				Image:    "nginx:latest",
				Replicas: 2,
				Status:   StatusPending,
			},
		},
		{
			name: "invalid request body",
			requestBody: map[string]interface{}{
				"name": "",
				"image": "nginx:latest",
			},
			mockSetup: func(m *MockOrchestrator) {
				// No mock setup needed as validation should fail first
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: ErrorResponse{
				Error: "validation failed",
				Code:  "VALIDATION_ERROR",
			},
		},
		{
			name:        "malformed JSON",
			requestBody: "invalid json",
			mockSetup: func(m *MockOrchestrator) {
				// No mock setup needed
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody: ErrorResponse{
				Error: "invalid request body",
				Code:  "INVALID_JSON",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockOrch := &MockOrchestrator{}
			tt.mockSetup(mockOrch)

			handler := NewHandler(mockOrch)

			var reqBody []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				reqBody = []byte(str)
			} else {
				reqBody, err = json.Marshal(tt.requestBody)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/pods", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response interface{}
				if _, ok := tt.expectedBody.(PodResponse); ok {
					var podResp PodResponse
					err := json.Unmarshal(w.Body.Bytes(), &podResp)
					require.NoError(t, err)
					response = podResp
				} else {
					var errResp ErrorResponse
					err := json.Unmarshal(w.Body.Bytes(), &errResp)
					require.NoError(t, err)
					response = errResp
				}

				// Compare relevant fields
				if podResp, ok := tt.expectedBody.(PodResponse); ok {
					actualResp := response.(PodResponse)
					assert.Equal(t, podResp.Name, actualResp.Name)
					assert.Equal(t, podResp.Image, actualResp.Image)
					assert.Equal(t, podResp.Replicas, actualResp.Replicas)
					assert.Equal(t, podResp.Status, actualResp.Status)
				} else if errResp, ok := tt.expectedBody.(ErrorResponse); ok {
					actualResp := response.(ErrorResponse)
					assert.Contains(t, actualResp.Error, errResp.Error)
					if errResp.Code != "" {
						assert.Equal(t, errResp.Code, actualResp.Code)
					}
				}
			}

			mockOrch.AssertExpectations(t)
		})
	}
}

func TestHandler_GetPod(t *testing.T) {
	tests := []struct {
		name           string
		podName        string
		mockSetup      func(*MockOrchestrator)
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name:    "successful pod retrieval",
			podName: "test-pod",
			mockSetup: func(m *MockOrchestrator) {
				m.On("GetPod", mock.Anything, "test-pod").Return(&PodResponse{
					Name:     "test-pod",
					Image:    "nginx:latest",
					Replicas: 1,
					Status:   StatusRunning,
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: PodResponse{
				Name:     "test-pod",
				Image:    "nginx:latest",
				Replicas: 1,
				Status:   StatusRunning,
			},
		},
		{
			name:    "pod not found",
			podName: "nonexistent-pod",
			mockSetup: func(m *MockOrchestrator) {
				m.On("GetPod", mock.Anything, "nonexistent-pod").Return(nil, ErrPodNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: ErrorResponse{
				Error: "pod not found",
				Code:  "POD_NOT_FOUND",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockOrch := &MockOrchestrator{}
			tt.mockSetup(mockOrch)

			handler := NewHandler(mockOrch)

			req := httptest.NewRequest(http.MethodGet, "/pods/"+tt.podName, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var response interface{}
				if _, ok := tt.expectedBody.(PodResponse); ok {
					var podResp PodResponse
					err := json.Unmarshal(w.Body.Bytes(), &podResp)
					require.NoError(t, err)
					response = podResp
				} else {
					var errResp ErrorResponse
					err := json.Unmarshal(w.Body.Bytes(), &errResp)
					require.NoError(t, err)
					response = errResp
				}

				// Compare relevant fields
				if podResp, ok := tt.expectedBody.(PodResponse); ok {
					actualResp := response.(PodResponse)
					assert.Equal(t, podResp.Name, actualResp.Name)
					assert.Equal(t, podResp.Image, actualResp.Image)
					assert.Equal(t, podResp.Replicas, actualResp.Replicas)
					assert.Equal(t, podResp.Status, actualResp.Status)
				} else if errResp, ok := tt.expectedBody.(ErrorResponse); ok {
					actualResp := response.(ErrorResponse)
					assert.Contains(t, actualResp.Error, errResp.Error)
					if errResp.Code != "" {
						assert.Equal(t, errResp.Code, actualResp.Code)
					}
				}
			}

			mockOrch.AssertExpectations(t)
		})
	}
}

func TestHandler_ListPods(t *testing.T) {
	tests := []struct {
		name           string
		mockSetup      func(*MockOrchestrator)
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name: "successful pod listing",
			mockSetup: func(m *MockOrchestrator) {
				m.On("ListPods", mock.Anything).Return([]PodSummary{
					{
						Name:     "pod-1",
						Image:    "nginx:latest",
						Replicas: 1,
						Status:   StatusRunning,
					},
					{
						Name:     "pod-2",
						Image:    "redis:alpine",
						Replicas: 2,
						Status:   StatusPending,
					},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: ListPodsResponse{
				Pods: []PodSummary{
					{
						Name:     "pod-1",
						Image:    "nginx:latest",
						Replicas: 1,
						Status:   StatusRunning,
					},
					{
						Name:     "pod-2",
						Image:    "redis:alpine",
						Replicas: 2,
						Status:   StatusPending,
					},
				},
			},
		},
		{
			name: "empty pod list",
			mockSetup: func(m *MockOrchestrator) {
				m.On("ListPods", mock.Anything).Return([]PodSummary{}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: ListPodsResponse{
				Pods: []PodSummary{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockOrch := &MockOrchestrator{}
			tt.mockSetup(mockOrch)

			handler := NewHandler(mockOrch)

			req := httptest.NewRequest(http.MethodGet, "/pods", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response ListPodsResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			expectedResp := tt.expectedBody.(ListPodsResponse)
			assert.Len(t, response.Pods, len(expectedResp.Pods))
			for i, expectedPod := range expectedResp.Pods {
				assert.Equal(t, expectedPod.Name, response.Pods[i].Name)
				assert.Equal(t, expectedPod.Image, response.Pods[i].Image)
				assert.Equal(t, expectedPod.Replicas, response.Pods[i].Replicas)
				assert.Equal(t, expectedPod.Status, response.Pods[i].Status)
			}

			mockOrch.AssertExpectations(t)
		})
	}
}

func TestHandler_DeletePod(t *testing.T) {
	tests := []struct {
		name           string
		podName        string
		mockSetup      func(*MockOrchestrator)
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name:    "successful pod deletion",
			podName: "test-pod",
			mockSetup: func(m *MockOrchestrator) {
				m.On("DeletePod", mock.Anything, "test-pod").Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			expectedBody:   nil,
		},
		{
			name:    "pod not found for deletion",
			podName: "nonexistent-pod",
			mockSetup: func(m *MockOrchestrator) {
				m.On("DeletePod", mock.Anything, "nonexistent-pod").Return(ErrPodNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: ErrorResponse{
				Error: "pod not found",
				Code:  "POD_NOT_FOUND",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockOrch := &MockOrchestrator{}
			tt.mockSetup(mockOrch)

			handler := NewHandler(mockOrch)

			req := httptest.NewRequest(http.MethodDelete, "/pods/"+tt.podName, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedBody != nil {
				var errResp ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &errResp)
				require.NoError(t, err)

				expectedResp := tt.expectedBody.(ErrorResponse)
				assert.Contains(t, errResp.Error, expectedResp.Error)
				if expectedResp.Code != "" {
					assert.Equal(t, expectedResp.Code, errResp.Code)
				}
			}

			mockOrch.AssertExpectations(t)
		})
	}
}

func TestHandler_UnsupportedMethod(t *testing.T) {
	mockOrch := &MockOrchestrator{}
	handler := NewHandler(mockOrch)

	req := httptest.NewRequest(http.MethodPatch, "/pods", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)

	var errResp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)

	assert.Contains(t, errResp.Error, "method not allowed")
}

func TestHandler_InvalidPath(t *testing.T) {
	mockOrch := &MockOrchestrator{}
	handler := NewHandler(mockOrch)

	req := httptest.NewRequest(http.MethodGet, "/invalid", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var errResp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)

	assert.Contains(t, errResp.Error, "not found")
}