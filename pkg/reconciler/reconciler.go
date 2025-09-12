package reconciler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mansoormajeed/dubernetes/pkg/config"
	"github.com/mansoormajeed/dubernetes/pkg/database"
	"github.com/mansoormajeed/dubernetes/pkg/docker"
	"github.com/mansoormajeed/dubernetes/pkg/nginx"
)

// Reconciler manages the overall state reconciliation loop
type Reconciler struct {
	config       *config.Config
	db           *database.Database
	docker       *docker.DockerClient
	nginx        *nginx.Manager
	lastReconcile time.Time
}

// NewReconciler creates a new reconciler instance
func NewReconciler(cfg *config.Config, database *database.Database, dockerClient *docker.DockerClient, nginxManager *nginx.Manager) *Reconciler {
	return &Reconciler{
		config: cfg,
		db:     database,
		docker: dockerClient,
		nginx:  nginxManager,
	}
}

// Start begins the reconciliation loop
func (r *Reconciler) Start(ctx context.Context) error {
	log.Println("Starting reconciliation loop...")
	
	ticker := time.NewTicker(r.config.Reconciler.Interval)
	defer ticker.Stop()

	// Run initial reconciliation
	if err := r.ReconcileOnce(); err != nil {
		log.Printf("Initial reconciliation failed: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("Reconciliation loop stopped")
			return ctx.Err()
		case <-ticker.C:
			if err := r.ReconcileOnce(); err != nil {
				log.Printf("Reconciliation failed: %v", err)
			}
		}
	}
}

// ReconcileOnce performs a single reconciliation cycle
func (r *Reconciler) ReconcileOnce() error {
	r.lastReconcile = time.Now()
	log.Printf("Starting reconciliation cycle at %v", r.lastReconcile)

	// Get all pods from database
	pods, err := r.db.ListPods()
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	// Reconcile each pod
	for _, pod := range pods {
		if err := r.reconcilePod(pod); err != nil {
			log.Printf("Failed to reconcile pod %s: %v", pod.Name, err)
			// Continue with other pods even if one fails
		}
	}

	// Update nginx configuration
	if err := r.updateNginxConfig(); err != nil {
		log.Printf("Failed to update nginx config: %v", err)
		// Don't return error as this shouldn't stop reconciliation
	}

	log.Printf("Reconciliation cycle completed in %v", time.Since(r.lastReconcile))
	return nil
}

// reconcilePod reconciles the state of a single pod
func (r *Reconciler) reconcilePod(pod *database.Pod) error {
	log.Printf("Reconciling pod %s (desired: %s, replicas: %d)", pod.Name, pod.DesiredState, pod.Replicas)

	// Get current replicas for the pod
	replicas, err := r.db.ListReplicasForPod(pod.Name)
	if err != nil {
		return fmt.Errorf("failed to list replicas for pod %s: %w", pod.Name, err)
	}

	if pod.DesiredState == "running" {
		return r.reconcileRunningPod(pod, replicas)
	} else if pod.DesiredState == "stopped" {
		return r.reconcileStoppedPod(pod, replicas)
	}

	return nil
}

// reconcileRunningPod ensures a pod has the correct number of running replicas
func (r *Reconciler) reconcileRunningPod(pod *database.Pod, replicas []*database.Replica) error {
	runningReplicas := 0
	var replicasToRestart []*database.Replica

	// Count running replicas and identify failed ones
	for _, replica := range replicas {
		if replica.Status == "running" {
			// Verify container is actually running
			isRunning, err := r.docker.IsContainerRunning(replica.ContainerID)
			if err != nil {
				log.Printf("Failed to check container status for %s: %v", replica.ContainerID, err)
				continue
			}

			if isRunning {
				runningReplicas++
			} else {
				// Container is not actually running, mark for restart
				replica.Status = "failed"
				replicasToRestart = append(replicasToRestart, replica)
			}
		} else if replica.Status == "failed" && r.shouldRestartReplica(replica) {
			replicasToRestart = append(replicasToRestart, replica)
		}
	}

	// Restart failed replicas
	for _, replica := range replicasToRestart {
		if err := r.restartReplica(pod, replica); err != nil {
			log.Printf("Failed to restart replica %s: %v", replica.ReplicaID, err)
		} else {
			runningReplicas++
		}
	}

	// Create additional replicas if needed
	replicasNeeded := pod.Replicas - runningReplicas
	for i := 0; i < replicasNeeded; i++ {
		if err := r.createReplica(pod); err != nil {
			log.Printf("Failed to create replica for pod %s: %v", pod.Name, err)
		}
	}

	// Remove excess replicas if needed
	if runningReplicas > pod.Replicas {
		excessReplicas := runningReplicas - pod.Replicas
		if err := r.removeExcessReplicas(pod, replicas, excessReplicas); err != nil {
			log.Printf("Failed to remove excess replicas for pod %s: %v", pod.Name, err)
		}
	}

	return nil
}

// reconcileStoppedPod ensures all replicas for a pod are stopped
func (r *Reconciler) reconcileStoppedPod(pod *database.Pod, replicas []*database.Replica) error {
	for _, replica := range replicas {
		if replica.Status == "running" {
			if err := r.stopReplica(replica); err != nil {
				log.Printf("Failed to stop replica %s: %v", replica.ReplicaID, err)
			}
		}
	}
	return nil
}

// createReplica creates a new replica for a pod
func (r *Reconciler) createReplica(pod *database.Pod) error {
	// Allocate port
	usedPorts, err := r.db.GetUsedPorts()
	if err != nil {
		return fmt.Errorf("failed to get used ports: %w", err)
	}

	port, err := r.docker.AllocatePort(usedPorts, r.config.Containers.PortRangeStart, r.config.Containers.PortRangeEnd)
	if err != nil {
		return fmt.Errorf("failed to allocate port: %w", err)
	}

	// Generate replica ID
	replicaID := fmt.Sprintf("%s-%d", pod.Name, time.Now().UnixNano())

	// Create replica record first
	replica := &database.Replica{
		PodName:      pod.Name,
		ReplicaID:    replicaID,
		ContainerID:  "", // Will be set after container creation
		Port:         port,
		Status:       "creating",
		RestartCount: 0,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := r.db.CreateReplica(replica); err != nil {
		return fmt.Errorf("failed to create replica record: %w", err)
	}

	// Create container
	req := &docker.RunContainerRequest{
		Image:  pod.Image,
		Name:   replicaID,
		Port:   port,
		Labels: map[string]string{
			"dubernetes.pod":     pod.Name,
			"dubernetes.replica": replicaID,
		},
	}

	containerID, err := r.docker.RunContainer(req)
	if err != nil {
		// Update replica status to failed
		replica.Status = "failed"
		r.db.UpdateReplica(replica)
		return fmt.Errorf("failed to create container: %w", err)
	}

	// Update replica with container ID and status
	replica.ContainerID = containerID
	replica.Status = "running"
	replica.UpdatedAt = time.Now()

	if err := r.db.UpdateReplica(replica); err != nil {
		log.Printf("Failed to update replica %s status: %v", replicaID, err)
	}

	log.Printf("Created replica %s for pod %s (container: %s, port: %d)", replicaID, pod.Name, containerID, port)
	return nil
}

// restartReplica restarts a failed replica
func (r *Reconciler) restartReplica(pod *database.Pod, replica *database.Replica) error {
	log.Printf("Restarting replica %s (restart count: %d)", replica.ReplicaID, replica.RestartCount)

	// Stop and remove existing container if it exists
	if replica.ContainerID != "" {
		r.docker.StopContainer(replica.ContainerID)
		r.docker.RemoveContainer(replica.ContainerID)
	}

	// Wait for backoff period
	backoff := CalculateRestartBackoff(replica.RestartCount)
	if backoff > 0 {
		log.Printf("Waiting %v before restarting replica %s", backoff, replica.ReplicaID)
		time.Sleep(backoff)
	}

	// Create new container
	req := &docker.RunContainerRequest{
		Image:  pod.Image,
		Name:   replica.ReplicaID,
		Port:   replica.Port,
		Labels: map[string]string{
			"dubernetes.pod":     pod.Name,
			"dubernetes.replica": replica.ReplicaID,
		},
	}

	containerID, err := r.docker.RunContainer(req)
	if err != nil {
		// Update restart count and status
		replica.RestartCount++
		replica.Status = "failed"
		replica.UpdatedAt = time.Now()
		r.db.UpdateReplica(replica)
		return fmt.Errorf("failed to restart container: %w", err)
	}

	// Update replica
	replica.ContainerID = containerID
	replica.Status = "running"
	replica.RestartCount++
	replica.UpdatedAt = time.Now()

	if err := r.db.UpdateReplica(replica); err != nil {
		log.Printf("Failed to update replica %s after restart: %v", replica.ReplicaID, err)
	}

	log.Printf("Restarted replica %s (new container: %s)", replica.ReplicaID, containerID)
	return nil
}

// stopReplica stops a running replica
func (r *Reconciler) stopReplica(replica *database.Replica) error {
	log.Printf("Stopping replica %s", replica.ReplicaID)

	if replica.ContainerID != "" {
		if err := r.docker.StopContainer(replica.ContainerID); err != nil {
			log.Printf("Failed to stop container %s: %v", replica.ContainerID, err)
		}

		if err := r.docker.RemoveContainer(replica.ContainerID); err != nil {
			log.Printf("Failed to remove container %s: %v", replica.ContainerID, err)
		}
	}

	// Update replica status
	replica.Status = "stopped"
	replica.UpdatedAt = time.Now()

	if err := r.db.UpdateReplica(replica); err != nil {
		log.Printf("Failed to update replica %s status: %v", replica.ReplicaID, err)
	}

	return nil
}

// removeExcessReplicas removes running replicas when scaling down
func (r *Reconciler) removeExcessReplicas(pod *database.Pod, replicas []*database.Replica, count int) error {
	removed := 0
	for _, replica := range replicas {
		if removed >= count {
			break
		}
		if replica.Status == "running" {
			if err := r.stopReplica(replica); err != nil {
				log.Printf("Failed to remove excess replica %s: %v", replica.ReplicaID, err)
			} else {
				removed++
			}
		}
	}
	return nil
}

// shouldRestartReplica determines if a failed replica should be restarted
func (r *Reconciler) shouldRestartReplica(replica *database.Replica) bool {
	// Calculate backoff time
	backoff := CalculateRestartBackoff(replica.RestartCount)
	
	// Check if enough time has passed since last update
	timeSinceUpdate := time.Since(replica.UpdatedAt)
	return timeSinceUpdate >= backoff
}

// updateNginxConfig updates the nginx configuration based on current pod state
func (r *Reconciler) updateNginxConfig() error {
	if err := nginx.UpdateManagerFromDatabase(r.nginx, r.db); err != nil {
		return fmt.Errorf("failed to update nginx config: %w", err)
	}
	return nil
}

// CalculateRestartBackoff calculates the backoff time for container restarts
func CalculateRestartBackoff(restartCount int) time.Duration {
	if restartCount == 0 {
		return 1 * time.Second
	}

	// Exponential backoff: 2^restartCount seconds, max 60 seconds
	backoff := time.Duration(1<<uint(restartCount)) * time.Second
	maxBackoff := 60 * time.Second
	
	if backoff > maxBackoff {
		return maxBackoff
	}
	
	return backoff
}