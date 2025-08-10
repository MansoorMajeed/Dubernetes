# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Dubernetes is a simplified container orchestration system designed for learning how orchestration works behind the scenes. This is an educational project that aims to demonstrate the core concepts of Kubernetes-like container orchestration without enterprise complexity.

## Architecture

The system is designed with the following components:

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

## Development Status

This repository is in early planning stages with only architecture documentation present. When implementing:

1. Follow the architectural flow described in `architecture.md:86-94`
2. Prioritize transparency and educational value over performance
3. Keep implementations simple to focus on core orchestration concepts
4. Ensure every decision is visible and can be explained to learners

## Key Design Principles

- **Transparency**: Every decision should be visible and explained
- **Simplicity**: Focus on core concepts without enterprise complexity  
- **Educational**: Show the "magic" behind container orchestration
- **Hands-on**: Users should be able to experiment and see immediate results

Refer to `architecture.md` for the complete system design and component interaction flows.