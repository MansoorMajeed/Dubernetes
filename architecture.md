# Dubernetes Architecture

## Overview
Dubernetes is a simplified container orchestration system designed for learning how orchestration works behind the scenes.

## Architecture Diagram

```mermaid
graph TB
    subgraph "Dubernetes Control Plane"
        API["API Server<br/>(REST endpoints)"]
        SCHED["Scheduler<br/>(Pod → Node assignment)"]
        CTRL["Controller Manager<br/>(Deployment reconciliation)"]
        STORE["State Store<br/>(JSON/SQLite)"]
    end
    
    subgraph "Single Node"
        KUBELET["Node Agent<br/>(Pod lifecycle)"]
        DOCKER["Container Runtime<br/>(Docker/containerd)"]
        PROXY["Service Proxy<br/>(Load balancing)"]
        
        subgraph "Running Pods"
            POD1["Pod 1<br/>app=web"]
            POD2["Pod 2<br/>app=web"]
            POD3["Pod 3<br/>app=db"]
        end
    end
    
    subgraph "User Interface"
        CLI["CLI Tool<br/>(kubectl-like)"]
        DASH["Web Dashboard<br/>(Cluster view)"]
    end
    
    %% User interactions
    CLI -->|deploy, scale, expose| API
    DASH -->|view cluster state| API
    
    %% Control plane flow
    API -->|store state| STORE
    API -->|trigger scheduling| SCHED
    SCHED -->|assign pods| STORE
    CTRL -->|watch desired state| STORE
    CTRL -->|reconcile| API
    
    %% Node operations
    API -->|pod specs| KUBELET
    KUBELET -->|create/destroy| DOCKER
    DOCKER -->|manage| POD1
    DOCKER -->|manage| POD2
    DOCKER -->|manage| POD3
    
    %% Service networking
    API -->|service config| PROXY
    PROXY -->|route traffic| POD1
    PROXY -->|route traffic| POD2
    
    %% Feedback loop
    KUBELET -->|pod status| API
    
    classDef controlPlane fill:#e1f5fe
    classDef node fill:#f3e5f5
    classDef ui fill:#e8f5e8
    
    class API,SCHED,CTRL,STORE controlPlane
    class KUBELET,DOCKER,PROXY,POD1,POD2,POD3 node
    class CLI,DASH ui
```

## Core Components

### Control Plane
- **API Server**: REST endpoints for all operations
- **Scheduler**: Assigns pods to nodes based on resources
- **Controller Manager**: Maintains desired state (deployments, replicas)
- **State Store**: Persistent storage for cluster state

### Node Components
- **Node Agent**: Manages pod lifecycle on the node
- **Container Runtime**: Actually runs containers (Docker/containerd)
- **Service Proxy**: Handles load balancing and service discovery

### User Interface
- **CLI Tool**: kubectl-like interface for deployments
- **Web Dashboard**: Visual representation of cluster state

## Key Learning Flow

1. **User creates Deployment** via CLI
2. **API Server** stores desired state
3. **Controller** notices gap between desired/actual state
4. **Scheduler** assigns Pods to Node
5. **Node Agent** creates containers
6. **Service Proxy** handles load balancing
7. **Dashboard** shows real-time state changes

## Design Goals

- **Transparency**: Every decision is visible and explained
- **Simplicity**: Focus on core concepts without enterprise complexity
- **Educational**: Show the "magic" behind container orchestration
- **Hands-on**: Users can experiment and see immediate results