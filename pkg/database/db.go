package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Database represents the SQLite database connection and operations
type Database struct {
	conn *sql.DB
}

// Pod represents a pod in the database
type Pod struct {
	Name         string    `json:"name"`
	Image        string    `json:"image"`
	Replicas     int       `json:"replicas"`
	Host         string    `json:"host,omitempty"`
	DesiredState string    `json:"desired_state"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Replica represents a replica/container instance in the database
type Replica struct {
	PodName      string    `json:"pod_name"`
	ReplicaID    string    `json:"replica_id"`
	ContainerID  string    `json:"container_id"`
	Port         int       `json:"port"`
	Status       string    `json:"status"`
	RestartCount int       `json:"restart_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// NewDatabase creates a new database connection and initializes the schema
func NewDatabase(dbPath string) (*Database, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &Database{conn: conn}

	// Create tables
	if err := db.createTables(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *Database) Close() error {
	return db.conn.Close()
}

// createTables creates the necessary database tables
func (db *Database) createTables() error {
	// Create pods table
	podsSchema := `
	CREATE TABLE IF NOT EXISTS pods (
		name TEXT PRIMARY KEY,
		image TEXT NOT NULL,
		replicas INTEGER NOT NULL DEFAULT 1,
		host TEXT,
		desired_state TEXT NOT NULL DEFAULT 'running',
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);`

	if _, err := db.conn.Exec(podsSchema); err != nil {
		return fmt.Errorf("failed to create pods table: %w", err)
	}

	// Create replicas table
	replicasSchema := `
	CREATE TABLE IF NOT EXISTS replicas (
		replica_id TEXT PRIMARY KEY,
		pod_name TEXT NOT NULL,
		container_id TEXT,
		port INTEGER NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		restart_count INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		FOREIGN KEY (pod_name) REFERENCES pods(name) ON DELETE CASCADE
	);`

	if _, err := db.conn.Exec(replicasSchema); err != nil {
		return fmt.Errorf("failed to create replicas table: %w", err)
	}

	return nil
}

// Pod CRUD operations

// CreatePod creates a new pod in the database
func (db *Database) CreatePod(pod *Pod) error {
	query := `
	INSERT INTO pods (name, image, replicas, host, desired_state, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := db.conn.Exec(query, pod.Name, pod.Image, pod.Replicas, pod.Host, pod.DesiredState, pod.CreatedAt, pod.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create pod: %w", err)
	}

	return nil
}

// GetPod retrieves a pod by name
func (db *Database) GetPod(name string) (*Pod, error) {
	query := `
	SELECT name, image, replicas, host, desired_state, created_at, updated_at
	FROM pods WHERE name = ?`

	row := db.conn.QueryRow(query, name)

	pod := &Pod{}
	err := row.Scan(&pod.Name, &pod.Image, &pod.Replicas, &pod.Host, &pod.DesiredState, &pod.CreatedAt, &pod.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("pod not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get pod: %w", err)
	}

	return pod, nil
}

// UpdatePod updates an existing pod
func (db *Database) UpdatePod(pod *Pod) error {
	query := `
	UPDATE pods SET image = ?, replicas = ?, host = ?, desired_state = ?, updated_at = ?
	WHERE name = ?`

	result, err := db.conn.Exec(query, pod.Image, pod.Replicas, pod.Host, pod.DesiredState, pod.UpdatedAt, pod.Name)
	if err != nil {
		return fmt.Errorf("failed to update pod: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("pod not found: %s", pod.Name)
	}

	return nil
}

// DeletePod deletes a pod and all its replicas
func (db *Database) DeletePod(name string) error {
	query := `DELETE FROM pods WHERE name = ?`

	result, err := db.conn.Exec(query, name)
	if err != nil {
		return fmt.Errorf("failed to delete pod: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("pod not found: %s", name)
	}

	return nil
}

// ListPods retrieves all pods
func (db *Database) ListPods() ([]*Pod, error) {
	query := `
	SELECT name, image, replicas, host, desired_state, created_at, updated_at
	FROM pods ORDER BY name`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}
	defer rows.Close()

	var pods []*Pod
	for rows.Next() {
		pod := &Pod{}
		err := rows.Scan(&pod.Name, &pod.Image, &pod.Replicas, &pod.Host, &pod.DesiredState, &pod.CreatedAt, &pod.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pod: %w", err)
		}
		pods = append(pods, pod)
	}

	return pods, nil
}

// Replica CRUD operations

// CreateReplica creates a new replica in the database
func (db *Database) CreateReplica(replica *Replica) error {
	query := `
	INSERT INTO replicas (replica_id, pod_name, container_id, port, status, restart_count, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.conn.Exec(query, replica.ReplicaID, replica.PodName, replica.ContainerID, replica.Port, replica.Status, replica.RestartCount, replica.CreatedAt, replica.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create replica: %w", err)
	}

	return nil
}

// GetReplica retrieves a replica by ID
func (db *Database) GetReplica(replicaID string) (*Replica, error) {
	query := `
	SELECT replica_id, pod_name, container_id, port, status, restart_count, created_at, updated_at
	FROM replicas WHERE replica_id = ?`

	row := db.conn.QueryRow(query, replicaID)

	replica := &Replica{}
	err := row.Scan(&replica.ReplicaID, &replica.PodName, &replica.ContainerID, &replica.Port, &replica.Status, &replica.RestartCount, &replica.CreatedAt, &replica.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("replica not found: %s", replicaID)
		}
		return nil, fmt.Errorf("failed to get replica: %w", err)
	}

	return replica, nil
}

// UpdateReplica updates an existing replica
func (db *Database) UpdateReplica(replica *Replica) error {
	query := `
	UPDATE replicas SET container_id = ?, port = ?, status = ?, restart_count = ?, updated_at = ?
	WHERE replica_id = ?`

	result, err := db.conn.Exec(query, replica.ContainerID, replica.Port, replica.Status, replica.RestartCount, replica.UpdatedAt, replica.ReplicaID)
	if err != nil {
		return fmt.Errorf("failed to update replica: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("replica not found: %s", replica.ReplicaID)
	}

	return nil
}

// DeleteReplica deletes a replica
func (db *Database) DeleteReplica(replicaID string) error {
	query := `DELETE FROM replicas WHERE replica_id = ?`

	result, err := db.conn.Exec(query, replicaID)
	if err != nil {
		return fmt.Errorf("failed to delete replica: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("replica not found: %s", replicaID)
	}

	return nil
}

// ListReplicasForPod retrieves all replicas for a specific pod
func (db *Database) ListReplicasForPod(podName string) ([]*Replica, error) {
	query := `
	SELECT replica_id, pod_name, container_id, port, status, restart_count, created_at, updated_at
	FROM replicas WHERE pod_name = ? ORDER BY replica_id`

	rows, err := db.conn.Query(query, podName)
	if err != nil {
		return nil, fmt.Errorf("failed to list replicas for pod: %w", err)
	}
	defer rows.Close()

	var replicas []*Replica
	for rows.Next() {
		replica := &Replica{}
		err := rows.Scan(&replica.ReplicaID, &replica.PodName, &replica.ContainerID, &replica.Port, &replica.Status, &replica.RestartCount, &replica.CreatedAt, &replica.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan replica: %w", err)
		}
		replicas = append(replicas, replica)
	}

	return replicas, nil
}

// GetUsedPorts returns all ports currently in use by replicas
func (db *Database) GetUsedPorts() ([]int, error) {
	query := `SELECT DISTINCT port FROM replicas ORDER BY port`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get used ports: %w", err)
	}
	defer rows.Close()

	var ports []int
	for rows.Next() {
		var port int
		if err := rows.Scan(&port); err != nil {
			return nil, fmt.Errorf("failed to scan port: %w", err)
		}
		ports = append(ports, port)
	}

	return ports, nil
}

// Reset clears all data from the database (useful for testing)
func (db *Database) Reset() error {
	// Delete all replicas first (due to foreign key constraint)
	if _, err := db.conn.Exec("DELETE FROM replicas"); err != nil {
		return fmt.Errorf("failed to clear replicas: %w", err)
	}

	// Delete all pods
	if _, err := db.conn.Exec("DELETE FROM pods"); err != nil {
		return fmt.Errorf("failed to clear pods: %w", err)
	}

	return nil
}