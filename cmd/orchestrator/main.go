package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/mansoormajeed/dubernetes/pkg/api"
	"github.com/mansoormajeed/dubernetes/pkg/config"
	"github.com/mansoormajeed/dubernetes/pkg/database"
	"github.com/mansoormajeed/dubernetes/pkg/docker"
	"github.com/mansoormajeed/dubernetes/pkg/nginx"
	"github.com/mansoormajeed/dubernetes/pkg/reconciler"
)

func main() {
	// Parse command line flags
	var (
		configPath = flag.String("config", "configs/default.yaml", "Path to configuration file")
		verbose    = flag.Bool("verbose", false, "Enable verbose logging")
	)
	flag.Parse()

	// Set up logging
	if *verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	log.Println("Starting Dubernetes Orchestrator...")

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := database.NewDatabase(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize Docker client
	dockerClient := docker.NewDockerClient(&docker.RealCommandExecutor{})

	// Initialize Nginx manager
	nginxManager := nginx.NewManager(cfg, dockerClient)

	// Initialize orchestrator (for API)
	orchestrator := &OrchestratorImpl{
		db:           db,
		dockerClient: dockerClient,
		nginxManager: nginxManager,
	}

	// Initialize API server
	apiAddr := fmt.Sprintf("%s:%d", cfg.Orchestrator.Host, cfg.Orchestrator.Port)
	server := api.NewServer(apiAddr, orchestrator)

	// Initialize reconciler
	rec := reconciler.NewReconciler(cfg, db, dockerClient, nginxManager)

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start components
	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	// Start API server
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("Starting API server on %s", apiAddr)
		if err := server.Start(ctx); err != nil && err != context.Canceled {
			errCh <- fmt.Errorf("API server error: %w", err)
		}
	}()

	// Start reconciler
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("Starting reconciler")
		if err := rec.Start(ctx); err != nil && err != context.Canceled {
			errCh <- fmt.Errorf("reconciler error: %w", err)
		}
	}()

	log.Println("Dubernetes Orchestrator started successfully")
	log.Printf("API server: http://%s", server.GetAddr())
	log.Printf("Nginx proxy: http://%s:%d", cfg.Proxy.Host, cfg.Proxy.Port)

	// Wait for shutdown signal or error
	select {
	case sig := <-sigCh:
		log.Printf("Received signal %v, shutting down...", sig)
	case err := <-errCh:
		log.Printf("Component error: %v", err)
	}

	// Graceful shutdown
	log.Println("Initiating graceful shutdown...")
	cancel()

	// Wait for components to stop with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All components stopped successfully")
	case <-time.After(10 * time.Second):
		log.Println("Shutdown timeout reached, forcing exit")
	}

	log.Println("Dubernetes Orchestrator stopped")
}

// OrchestratorImpl implements the api.Orchestrator interface
type OrchestratorImpl struct {
	db           *database.Database
	dockerClient *docker.DockerClient
	nginxManager *nginx.Manager
}

// CreatePod creates a new pod
func (o *OrchestratorImpl) CreatePod(ctx context.Context, req api.PodRequest) (*api.PodResponse, error) {
	// Set default replicas if not specified
	replicas := req.Replicas
	if replicas == 0 {
		replicas = 1
	}

	// Convert API request to database pod
	dbPod := &database.Pod{
		Name:         req.Name,
		Image:        req.Image,
		Replicas:     replicas,
		DesiredState: "running",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Set access host if provided
	if req.Access != nil {
		dbPod.Host = req.Access.Host
	}

	err := o.db.CreatePod(dbPod)
	if err != nil {
		return nil, err
	}

	// Return the created pod as response
	return o.convertToResponse(dbPod, []*database.Replica{}), nil
}

// GetPod retrieves a pod by name
func (o *OrchestratorImpl) GetPod(ctx context.Context, name string) (*api.PodResponse, error) {
	dbPod, err := o.db.GetPod(name)
	if err != nil {
		return nil, err
	}

	// Get replicas for the pod
	replicas, err := o.db.ListReplicasForPod(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get replicas: %w", err)
	}

	return o.convertToResponse(dbPod, replicas), nil
}

// ListPods retrieves all pods
func (o *OrchestratorImpl) ListPods(ctx context.Context) ([]api.PodSummary, error) {
	dbPods, err := o.db.ListPods()
	if err != nil {
		return nil, err
	}

	pods := make([]api.PodSummary, len(dbPods))
	for i, dbPod := range dbPods {
		pods[i] = api.PodSummary{
			Name:     dbPod.Name,
			Image:    dbPod.Image,
			Replicas: dbPod.Replicas,
			Status:   dbPod.DesiredState,
		}
	}

	return pods, nil
}


// DeletePod deletes a pod
func (o *OrchestratorImpl) DeletePod(ctx context.Context, name string) error {
	// First, mark the pod as stopped to trigger cleanup
	dbPod, err := o.db.GetPod(name)
	if err != nil {
		return err
	}

	dbPod.DesiredState = "stopped"
	dbPod.UpdatedAt = time.Now()

	if err := o.db.UpdatePod(dbPod); err != nil {
		return fmt.Errorf("failed to mark pod as stopped: %w", err)
	}

	// Give reconciler a moment to clean up containers
	time.Sleep(2 * time.Second)

	// Delete all replicas first
	replicas, err := o.db.ListReplicasForPod(name)
	if err != nil {
		return fmt.Errorf("failed to list replicas: %w", err)
	}

	for _, replica := range replicas {
		if err := o.db.DeleteReplica(replica.ReplicaID); err != nil {
			log.Printf("Failed to delete replica %s: %v", replica.ReplicaID, err)
		}
	}

	// Delete the pod
	return o.db.DeletePod(name)
}

// countRunningReplicas counts the number of running replicas
func countRunningReplicas(replicas []*database.Replica) int {
	count := 0
	for _, replica := range replicas {
		if replica.Status == "running" {
			count++
		}
	}
	return count
}

// convertToResponse converts database pod and replicas to API response
func (o *OrchestratorImpl) convertToResponse(dbPod *database.Pod, replicas []*database.Replica) *api.PodResponse {
	response := &api.PodResponse{
		Name:      dbPod.Name,
		Image:     dbPod.Image,
		Replicas:  dbPod.Replicas,
		Status:    dbPod.DesiredState,
		CreatedAt: dbPod.CreatedAt,
		Instances: make([]api.PodInstance, len(replicas)),
	}

	// Set access config if available
	if dbPod.Host != "" {
		response.Access = &api.AccessConfig{
			Host: dbPod.Host,
		}
	}

	// Convert replicas to instances
	for i, replica := range replicas {
		response.Instances[i] = api.PodInstance{
			ID:     replica.ReplicaID,
			Port:   replica.Port,
			Status: replica.Status,
		}
	}

	return response
}