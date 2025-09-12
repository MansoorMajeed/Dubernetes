package nginx

import (
	"fmt"

	"github.com/mansoormajeed/dubernetes/pkg/database"
)

// GenerateConfigFromDatabase generates nginx configuration from database pods and replicas
func GenerateConfigFromDatabase(db *database.Database) (string, error) {
	// Get all pods from database
	pods, err := db.ListPods()
	if err != nil {
		return "", fmt.Errorf("failed to list pods from database: %w", err)
	}

	var podsWithReplicas []PodWithReplicas
	for _, pod := range pods {
		// Get replicas for each pod
		replicas, err := db.ListReplicasForPod(pod.Name)
		if err != nil {
			return "", fmt.Errorf("failed to get replicas for pod %s: %w", pod.Name, err)
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
	return GenerateNginxConfig(podsWithReplicas)
}

// UpdateManagerFromDatabase updates nginx manager configuration from database
func UpdateManagerFromDatabase(manager *Manager, db *database.Database) error {
	// Generate config from database
	config, err := GenerateConfigFromDatabase(db)
	if err != nil {
		return fmt.Errorf("failed to generate config from database: %w", err)
	}

	// Update nginx with new config
	return manager.UpdateConfig(config)
}