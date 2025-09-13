package nginx

import (
	"os"
	"testing"
	"time"

	"github.com/mansoormajeed/dubernetes/pkg/database"
)

func TestNginxConfigIntegration(t *testing.T) {
	// Create temporary database for testing
	tmpDB := "/tmp/test_nginx_integration.db"
	defer os.Remove(tmpDB)

	db, err := database.NewDatabase(tmpDB)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Reset database to clean state
	if err := db.Reset(); err != nil {
		t.Fatalf("Failed to reset database: %v", err)
	}

	t.Run("generate config from database pods", func(t *testing.T) {
		// Create test pods
		pod1 := &database.Pod{
			Name:         "web-app",
			Image:        "nginx:latest",
			Replicas:     2,
			Host:         "web.local",
			DesiredState: "running",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		pod2 := &database.Pod{
			Name:         "api-app",
			Image:        "node:14",
			Replicas:     1,
			Host:         "api.local",
			DesiredState: "running",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		// Pod without ingress (no host)
		pod3 := &database.Pod{
			Name:         "worker-app",
			Image:        "worker:latest",
			Replicas:     1,
			Host:         "",
			DesiredState: "running",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		// Create pods in database
		if err := db.CreatePod(pod1); err != nil {
			t.Fatalf("Failed to create pod1: %v", err)
		}
		if err := db.CreatePod(pod2); err != nil {
			t.Fatalf("Failed to create pod2: %v", err)
		}
		if err := db.CreatePod(pod3); err != nil {
			t.Fatalf("Failed to create pod3: %v", err)
		}

		// Create replicas for pods
		replicas := []*database.Replica{
			{PodName: "web-app", ReplicaID: "web-app-1", ContainerID: "container1", IPAddress: "172.17.0.2", Port: 32001, Status: "running", CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{PodName: "web-app", ReplicaID: "web-app-2", ContainerID: "container2", IPAddress: "172.17.0.3", Port: 32002, Status: "running", CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{PodName: "api-app", ReplicaID: "api-app-1", ContainerID: "container3", IPAddress: "172.17.0.4", Port: 32003, Status: "running", CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{PodName: "worker-app", ReplicaID: "worker-app-1", ContainerID: "container4", IPAddress: "172.17.0.5", Port: 32004, Status: "running", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		}

		for _, replica := range replicas {
			if err := db.CreateReplica(replica); err != nil {
				t.Fatalf("Failed to create replica %s: %v", replica.ReplicaID, err)
			}
		}

		// Get pods with their replicas
		allPods, err := db.ListPods()
		if err != nil {
			t.Fatalf("Failed to list pods: %v", err)
		}

		var podsWithReplicas []PodWithReplicas
		for _, pod := range allPods {
			replicas, err := db.ListReplicasForPod(pod.Name)
			if err != nil {
				t.Fatalf("Failed to get replicas for pod %s: %v", pod.Name, err)
			}

			// Convert to []database.Replica
			var replicaList []database.Replica
			for _, replica := range replicas {
				replicaList = append(replicaList, *replica)
			}

			podsWithReplicas = append(podsWithReplicas, PodWithReplicas{
				Pod:      *pod,
				Replicas: replicaList,
			})
		}

		// Generate nginx config
		config, err := GenerateNginxConfig(podsWithReplicas)
		if err != nil {
			t.Fatalf("Failed to generate nginx config: %v", err)
		}

		// Validate config
		if err := ValidateNginxConfig(config); err != nil {
			t.Fatalf("Generated config is invalid: %v", err)
		}

		// Check that config contains expected content
		expected := []string{
			"upstream web-app",
			"server 172.17.0.2:80",
			"server 172.17.0.3:80",
			"upstream api-app",
			"server 172.17.0.4:80",
			"server_name web.local",
			"server_name api.local",
		}

		for _, expectedContent := range expected {
			if !contains(config, expectedContent) {
				t.Errorf("Expected config to contain '%s', but it didn't.\nConfig:\n%s", expectedContent, config)
			}
		}

		// Check that worker-app (no ingress) is not in config
		if contains(config, "worker-app") {
			t.Errorf("Config should not contain worker-app (no ingress), but it did.\nConfig:\n%s", config)
		}
	})

	t.Run("generate config with failed replicas", func(t *testing.T) {
		// Reset database
		if err := db.Reset(); err != nil {
			t.Fatalf("Failed to reset database: %v", err)
		}

		// Create pod with mixed replica states
		pod := &database.Pod{
			Name:         "mixed-app",
			Image:        "nginx:latest",
			Replicas:     3,
			Host:         "mixed.local",
			DesiredState: "running",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if err := db.CreatePod(pod); err != nil {
			t.Fatalf("Failed to create pod: %v", err)
		}

		// Create replicas with different states
		replicas := []*database.Replica{
			{PodName: "mixed-app", ReplicaID: "mixed-app-1", ContainerID: "container1", IPAddress: "172.17.0.2", Port: 32001, Status: "running", CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{PodName: "mixed-app", ReplicaID: "mixed-app-2", ContainerID: "container2", IPAddress: "172.17.0.3", Port: 32002, Status: "failed", CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{PodName: "mixed-app", ReplicaID: "mixed-app-3", ContainerID: "container3", IPAddress: "172.17.0.4", Port: 32003, Status: "running", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		}

		for _, replica := range replicas {
			if err := db.CreateReplica(replica); err != nil {
				t.Fatalf("Failed to create replica %s: %v", replica.ReplicaID, err)
			}
		}

		// Get pods with their replicas
		allPods, err := db.ListPods()
		if err != nil {
			t.Fatalf("Failed to list pods: %v", err)
		}

		var podsWithReplicas []PodWithReplicas
		for _, pod := range allPods {
			replicas, err := db.ListReplicasForPod(pod.Name)
			if err != nil {
				t.Fatalf("Failed to get replicas for pod %s: %v", pod.Name, err)
			}

			var replicaList []database.Replica
			for _, replica := range replicas {
				replicaList = append(replicaList, *replica)
			}

			podsWithReplicas = append(podsWithReplicas, PodWithReplicas{
				Pod:      *pod,
				Replicas: replicaList,
			})
		}

		// Generate nginx config
		config, err := GenerateNginxConfig(podsWithReplicas)
		if err != nil {
			t.Fatalf("Failed to generate nginx config: %v", err)
		}

		// Validate config
		if err := ValidateNginxConfig(config); err != nil {
			t.Fatalf("Generated config is invalid: %v", err)
		}

		// Check that only running replicas are included
		if !contains(config, "server 172.17.0.2:80") {
			t.Errorf("Expected config to contain running replica with IP 172.17.0.2")
		}
		if !contains(config, "server 172.17.0.4:80") {
			t.Errorf("Expected config to contain running replica with IP 172.17.0.4")
		}
		if contains(config, "server 172.17.0.3:80") {
			t.Errorf("Config should not contain failed replica with IP 172.17.0.3")
		}
	})
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || 
		s[len(s)-len(substr):] == substr || 
		containsInMiddle(s, substr))))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}