package database

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase() failed: %v", err)
	}
	defer db.Close()

	// Verify database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}
}

func TestPodCRUD(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase() failed: %v", err)
	}
	defer db.Close()

	// Test Create Pod
	pod := &Pod{
		Name:         "test-app",
		Image:        "nginx:latest",
		Replicas:     3,
		Host:         "test.local",
		DesiredState: "running",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err = db.CreatePod(pod)
	if err != nil {
		t.Fatalf("CreatePod() failed: %v", err)
	}

	// Test Get Pod
	retrieved, err := db.GetPod("test-app")
	if err != nil {
		t.Fatalf("GetPod() failed: %v", err)
	}
	if retrieved.Name != pod.Name || retrieved.Image != pod.Image || retrieved.Replicas != pod.Replicas {
		t.Errorf("Retrieved pod doesn't match created pod")
	}

	// Test Update Pod
	pod.Replicas = 5
	pod.UpdatedAt = time.Now()
	err = db.UpdatePod(pod)
	if err != nil {
		t.Fatalf("UpdatePod() failed: %v", err)
	}

	// Verify update
	retrieved, err = db.GetPod("test-app")
	if err != nil {
		t.Fatalf("GetPod() after update failed: %v", err)
	}
	if retrieved.Replicas != 5 {
		t.Errorf("Pod replicas not updated, expected 5, got %d", retrieved.Replicas)
	}

	// Test List Pods
	pods, err := db.ListPods()
	if err != nil {
		t.Fatalf("ListPods() failed: %v", err)
	}
	if len(pods) != 1 {
		t.Errorf("Expected 1 pod, got %d", len(pods))
	}

	// Test Delete Pod
	err = db.DeletePod("test-app")
	if err != nil {
		t.Fatalf("DeletePod() failed: %v", err)
	}

	// Verify deletion
	_, err = db.GetPod("test-app")
	if err == nil {
		t.Error("Expected error when getting deleted pod")
	}
}

func TestReplicaCRUD(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase() failed: %v", err)
	}
	defer db.Close()

	// First create a pod
	pod := &Pod{
		Name:         "test-app",
		Image:        "nginx:latest",
		Replicas:     2,
		DesiredState: "running",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err = db.CreatePod(pod)
	if err != nil {
		t.Fatalf("CreatePod() failed: %v", err)
	}

	// Test Create Replica
	replica := &Replica{
		PodName:      "test-app",
		ReplicaID:    "test-app-1",
		ContainerID:  "container123",
		Port:         32001,
		Status:       "running",
		RestartCount: 0,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err = db.CreateReplica(replica)
	if err != nil {
		t.Fatalf("CreateReplica() failed: %v", err)
	}

	// Test Get Replica
	retrieved, err := db.GetReplica("test-app-1")
	if err != nil {
		t.Fatalf("GetReplica() failed: %v", err)
	}
	if retrieved.ReplicaID != replica.ReplicaID || retrieved.Port != replica.Port {
		t.Errorf("Retrieved replica doesn't match created replica")
	}

	// Test Update Replica
	replica.Status = "failed"
	replica.RestartCount = 1
	replica.UpdatedAt = time.Now()
	err = db.UpdateReplica(replica)
	if err != nil {
		t.Fatalf("UpdateReplica() failed: %v", err)
	}

	// Verify update
	retrieved, err = db.GetReplica("test-app-1")
	if err != nil {
		t.Fatalf("GetReplica() after update failed: %v", err)
	}
	if retrieved.Status != "failed" || retrieved.RestartCount != 1 {
		t.Errorf("Replica not updated correctly")
	}

	// Test List Replicas for Pod
	replicas, err := db.ListReplicasForPod("test-app")
	if err != nil {
		t.Fatalf("ListReplicasForPod() failed: %v", err)
	}
	if len(replicas) != 1 {
		t.Errorf("Expected 1 replica, got %d", len(replicas))
	}

	// Test Delete Replica
	err = db.DeleteReplica("test-app-1")
	if err != nil {
		t.Fatalf("DeleteReplica() failed: %v", err)
	}

	// Verify deletion
	_, err = db.GetReplica("test-app-1")
	if err == nil {
		t.Error("Expected error when getting deleted replica")
	}
}

func TestUniqueConstraints(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase() failed: %v", err)
	}
	defer db.Close()

	// Test unique pod name constraint
	pod1 := &Pod{
		Name:         "test-app",
		Image:        "nginx:latest",
		Replicas:     1,
		DesiredState: "running",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err = db.CreatePod(pod1)
	if err != nil {
		t.Fatalf("CreatePod() failed: %v", err)
	}

	// Try to create another pod with same name
	pod2 := &Pod{
		Name:         "test-app",
		Image:        "redis:latest",
		Replicas:     1,
		DesiredState: "running",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err = db.CreatePod(pod2)
	if err == nil {
		t.Error("Expected error when creating pod with duplicate name")
	}

	// Test unique replica ID constraint
	replica1 := &Replica{
		PodName:     "test-app",
		ReplicaID:   "test-app-1",
		ContainerID: "container123",
		Port:        32001,
		Status:      "running",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err = db.CreateReplica(replica1)
	if err != nil {
		t.Fatalf("CreateReplica() failed: %v", err)
	}

	// Try to create another replica with same ID
	replica2 := &Replica{
		PodName:     "test-app",
		ReplicaID:   "test-app-1",
		ContainerID: "container456",
		Port:        32002,
		Status:      "running",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err = db.CreateReplica(replica2)
	if err == nil {
		t.Error("Expected error when creating replica with duplicate ID")
	}
}

func TestDatabaseCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase() failed: %v", err)
	}

	// Create test data
	pod := &Pod{
		Name:         "test-app",
		Image:        "nginx:latest",
		Replicas:     1,
		DesiredState: "running",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err = db.CreatePod(pod)
	if err != nil {
		t.Fatalf("CreatePod() failed: %v", err)
	}

	replica := &Replica{
		PodName:     "test-app",
		ReplicaID:   "test-app-1",
		ContainerID: "container123",
		Port:        32001,
		Status:      "running",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err = db.CreateReplica(replica)
	if err != nil {
		t.Fatalf("CreateReplica() failed: %v", err)
	}

	// Test cleanup
	err = db.Reset()
	if err != nil {
		t.Fatalf("Reset() failed: %v", err)
	}

	// Verify data is gone
	pods, err := db.ListPods()
	if err != nil {
		t.Fatalf("ListPods() after reset failed: %v", err)
	}
	if len(pods) != 0 {
		t.Errorf("Expected 0 pods after reset, got %d", len(pods))
	}

	replicas, err := db.ListReplicasForPod("test-app")
	if err != nil {
		t.Fatalf("ListReplicasForPod() after reset failed: %v", err)
	}
	if len(replicas) != 0 {
		t.Errorf("Expected 0 replicas after reset, got %d", len(replicas))
	}

	db.Close()
}

func TestPortAllocation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase() failed: %v", err)
	}
	defer db.Close()

	// Test getting used ports when no replicas exist
	ports, err := db.GetUsedPorts()
	if err != nil {
		t.Fatalf("GetUsedPorts() failed: %v", err)
	}
	if len(ports) != 0 {
		t.Errorf("Expected 0 used ports, got %d", len(ports))
	}

	// Create pod and replica with port
	pod := &Pod{
		Name:         "test-app",
		Image:        "nginx:latest",
		Replicas:     1,
		DesiredState: "running",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err = db.CreatePod(pod)
	if err != nil {
		t.Fatalf("CreatePod() failed: %v", err)
	}

	replica := &Replica{
		PodName:     "test-app",
		ReplicaID:   "test-app-1",
		ContainerID: "container123",
		Port:        32001,
		Status:      "running",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err = db.CreateReplica(replica)
	if err != nil {
		t.Fatalf("CreateReplica() failed: %v", err)
	}

	// Test getting used ports
	ports, err = db.GetUsedPorts()
	if err != nil {
		t.Fatalf("GetUsedPorts() failed: %v", err)
	}
	if len(ports) != 1 || ports[0] != 32001 {
		t.Errorf("Expected used ports [32001], got %v", ports)
	}
}