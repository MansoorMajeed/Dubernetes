package main

import (
	"context"
	"testing"
	"time"

	"github.com/mansoormajeed/dubernetes/pkg/api"
	"github.com/mansoormajeed/dubernetes/pkg/config"
	"github.com/mansoormajeed/dubernetes/pkg/database"
	"github.com/mansoormajeed/dubernetes/pkg/docker"
	"github.com/mansoormajeed/dubernetes/pkg/nginx"
)

func TestOrchestratorImpl_CreatePod(t *testing.T) {
	// Setup
	db, err := database.NewDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	orchestrator := &OrchestratorImpl{
		db:           db,
		dockerClient: docker.NewDockerClient(&docker.RealCommandExecutor{}),
		nginxManager: nginx.NewManager(&config.Config{}, docker.NewDockerClient(&docker.RealCommandExecutor{})),
	}

	// Test data
	req := api.PodRequest{
		Name:     "test-pod",
		Image:    "nginx:latest",
		Replicas: 2,
		Access: &api.AccessConfig{
			Host: "test.local",
		},
	}

	// Test CreatePod
	response, err := orchestrator.CreatePod(context.Background(), req)
	if err != nil {
		t.Fatalf("Failed to create pod: %v", err)
	}

	// Verify response
	if response.Name != "test-pod" {
		t.Errorf("Expected response name 'test-pod', got '%s'", response.Name)
	}

	// Verify pod was created
	dbPod, err := db.GetPod("test-pod")
	if err != nil {
		t.Fatalf("Failed to get created pod: %v", err)
	}

	if dbPod.Name != "test-pod" {
		t.Errorf("Expected pod name 'test-pod', got '%s'", dbPod.Name)
	}
	if dbPod.Image != "nginx:latest" {
		t.Errorf("Expected image 'nginx:latest', got '%s'", dbPod.Image)
	}
	if dbPod.Replicas != 2 {
		t.Errorf("Expected 2 replicas, got %d", dbPod.Replicas)
	}
	if dbPod.Host != "test.local" {
		t.Errorf("Expected access host 'test.local', got '%s'", dbPod.Host)
	}
	if dbPod.DesiredState != "running" {
		t.Errorf("Expected desired state 'running', got '%s'", dbPod.DesiredState)
	}
}

func TestOrchestratorImpl_GetPod(t *testing.T) {
	// Setup
	db, err := database.NewDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	orchestrator := &OrchestratorImpl{
		db:           db,
		dockerClient: docker.NewDockerClient(&docker.RealCommandExecutor{}),
		nginxManager: nginx.NewManager(&config.Config{}, docker.NewDockerClient(&docker.RealCommandExecutor{})),
	}

	// Create a pod in database
	dbPod := &database.Pod{
		Name:         "test-pod",
		Image:        "nginx:latest",
		Replicas:     2,
		DesiredState: "running",
		Host:         "test.local",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err = db.CreatePod(dbPod)
	if err != nil {
		t.Fatalf("Failed to create pod in database: %v", err)
	}

	// Create some replicas
	replica1 := &database.Replica{
		PodName:      "test-pod",
		ReplicaID:    "test-pod-1",
		ContainerID:  "container1",
		Port:         32001,
		Status:       "running",
		RestartCount: 0,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	replica2 := &database.Replica{
		PodName:      "test-pod",
		ReplicaID:    "test-pod-2",
		ContainerID:  "container2",
		Port:         32002,
		Status:       "failed",
		RestartCount: 1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err = db.CreateReplica(replica1)
	if err != nil {
		t.Fatalf("Failed to create replica1: %v", err)
	}
	err = db.CreateReplica(replica2)
	if err != nil {
		t.Fatalf("Failed to create replica2: %v", err)
	}

	// Test GetPod
	response, err := orchestrator.GetPod(context.Background(), "test-pod")
	if err != nil {
		t.Fatalf("Failed to get pod: %v", err)
	}

	// Verify pod details
	if response.Name != "test-pod" {
		t.Errorf("Expected pod name 'test-pod', got '%s'", response.Name)
	}
	if response.Image != "nginx:latest" {
		t.Errorf("Expected image 'nginx:latest', got '%s'", response.Image)
	}
	if response.Replicas != 2 {
		t.Errorf("Expected 2 replicas, got %d", response.Replicas)
	}
	if response.Access.Host != "test.local" {
		t.Errorf("Expected access host 'test.local', got '%s'", response.Access.Host)
	}
	if response.Status != "running" {
		t.Errorf("Expected status 'running', got '%s'", response.Status)
	}
	if len(response.Instances) != 2 {
		t.Errorf("Expected 2 instances, got %d", len(response.Instances))
	}
	// Check for 1 running instance
	runningCount := 0
	for _, instance := range response.Instances {
		if instance.Status == "running" {
			runningCount++
		}
	}
	if runningCount != 1 {
		t.Errorf("Expected 1 running instance, got %d", runningCount)
	}
}

func TestOrchestratorImpl_ListPods(t *testing.T) {
	// Setup
	db, err := database.NewDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	orchestrator := &OrchestratorImpl{
		db:           db,
		dockerClient: docker.NewDockerClient(&docker.RealCommandExecutor{}),
		nginxManager: nginx.NewManager(&config.Config{}, docker.NewDockerClient(&docker.RealCommandExecutor{})),
	}

	// Create multiple pods
	pod1 := &database.Pod{
		Name:         "pod1",
		Image:        "nginx:latest",
		Replicas:     1,
		DesiredState: "running",
		Host:         "pod1.local",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	pod2 := &database.Pod{
		Name:         "pod2",
		Image:        "redis:latest",
		Replicas:     2,
		DesiredState: "running",
		Host:         "pod2.local",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err = db.CreatePod(pod1)
	if err != nil {
		t.Fatalf("Failed to create pod1: %v", err)
	}
	err = db.CreatePod(pod2)
	if err != nil {
		t.Fatalf("Failed to create pod2: %v", err)
	}

	// Test ListPods
	pods, err := orchestrator.ListPods(context.Background())
	if err != nil {
		t.Fatalf("Failed to list pods: %v", err)
	}

	// Verify results
	if len(pods) != 2 {
		t.Fatalf("Expected 2 pods, got %d", len(pods))
	}

	// Check pods
	found1 := false
	found2 := false
	for _, pod := range pods {
		if pod.Name == "pod1" {
			found1 = true
			if pod.Image != "nginx:latest" {
				t.Errorf("Expected pod1 image 'nginx:latest', got '%s'", pod.Image)
			}
		}
		if pod.Name == "pod2" {
			found2 = true
			if pod.Image != "redis:latest" {
				t.Errorf("Expected pod2 image 'redis:latest', got '%s'", pod.Image)
			}
		}
	}

	if !found1 {
		t.Error("pod1 not found in list")
	}
	if !found2 {
		t.Error("pod2 not found in list")
	}
}


func TestOrchestratorImpl_DeletePod(t *testing.T) {
	// Setup
	db, err := database.NewDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	orchestrator := &OrchestratorImpl{
		db:           db,
		dockerClient: docker.NewDockerClient(&docker.RealCommandExecutor{}),
		nginxManager: nginx.NewManager(&config.Config{}, docker.NewDockerClient(&docker.RealCommandExecutor{})),
	}

	// Create pod
	dbPod := &database.Pod{
		Name:         "test-pod",
		Image:        "nginx:latest",
		Replicas:     1,
		DesiredState: "running",
		Host:         "test.local",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err = db.CreatePod(dbPod)
	if err != nil {
		t.Fatalf("Failed to create pod: %v", err)
	}

	// Create replica
	replica := &database.Replica{
		PodName:      "test-pod",
		ReplicaID:    "test-pod-1",
		ContainerID:  "container1",
		Port:         32001,
		Status:       "running",
		RestartCount: 0,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err = db.CreateReplica(replica)
	if err != nil {
		t.Fatalf("Failed to create replica: %v", err)
	}

	// Test DeletePod
	err = orchestrator.DeletePod(context.Background(), "test-pod")
	if err != nil {
		t.Fatalf("Failed to delete pod: %v", err)
	}

	// Verify pod is deleted
	_, err = db.GetPod("test-pod")
	if err == nil {
		t.Error("Expected pod to be deleted, but it still exists")
	}

	// Verify replica is deleted
	replicas, err := db.ListReplicasForPod("test-pod")
	if err != nil {
		t.Fatalf("Failed to list replicas: %v", err)
	}
	if len(replicas) != 0 {
		t.Errorf("Expected 0 replicas after pod deletion, got %d", len(replicas))
	}
}

func TestCountRunningReplicas(t *testing.T) {
	replicas := []*database.Replica{
		{Status: "running"},
		{Status: "running"},
		{Status: "failed"},
		{Status: "creating"},
		{Status: "running"},
	}

	count := countRunningReplicas(replicas)
	if count != 3 {
		t.Errorf("Expected 3 running replicas, got %d", count)
	}
}