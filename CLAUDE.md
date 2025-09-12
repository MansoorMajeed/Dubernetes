# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Dubernetes is a simplified container orchestration system designed for learning how orchestration works behind the scenes. This is an educational project that aims to demonstrate the core concepts of Kubernetes-like container orchestration without enterprise complexity.

## Current Architecture (Finalized)

**Refer to `architecture.md` for complete details.** Key components:

### Core Components
- **CLI Tool (dubectl)**: Declarative YAML interface (`apply`, `get pods`, `delete`)
- **Orchestrator**: Single Go process managing pod lifecycle with replicas
- **SQLite Database**: State persistence for pods and services
- **Nginx Proxy**: Ingress with host-based routing and load balancing
- **Docker**: Container runtime via system commands

### YAML Specification
```yaml
name: string           # Pod name (required)
image: string          # Container image (required) 
replicas: int          # Number of replicas (default: 1)
access:                # Ingress configuration (optional)
  host: string         # Host header for routing (e.g., app1.local)
```

### Key Features
- **Ingress & Load Balancing**: Single entry point (port 80) routing to multiple replicas
- **Service Discovery**: Host-based routing (app1.local → pods)
- **Automatic Restart**: Containers restart with exponential backoff until deleted
- **State Management**: SQLite persistence across orchestrator restarts

## Implementation Status

**Current Phase**: Ready to start implementation
**Approach**: Test-Driven Development (TDD)
**Language**: Go with system Docker commands
**Architecture**: 
- Orchestrator API server (configurable port, default 8080)
- Nginx ingress proxy (configurable port, default 80)

## Development Plan

**Refer to `IMPLEMENTATION_PLAN.md` for complete TDD checklist.** 

### Implementation Phases
1. **Core Infrastructure**: Config management, SQLite database
2. **Docker Integration**: Container lifecycle management  
3. **REST API Server**: HTTP endpoints for CLI communication
4. **Nginx Proxy**: Dynamic config generation and load balancing
5. **Reconciliation Loop**: State reconciliation with restart logic
6. **CLI Client**: dubectl commands with YAML parsing
7. **Integration Testing**: End-to-end validation
8. **Documentation & Polish**: Final touches

### TDD Approach
- Write tests first for each component
- Mock external dependencies (Docker, file system, network)
- Integration tests with real Docker containers
- >80% test coverage requirement

## Session Continuity

**Status File**: `PROJECT_STATUS.md` - Contains current progress and context
**Resume Protocol**: When starting new sessions, check PROJECT_STATUS.md for:
- Current implementation phase
- Completed tasks and checklist status  
- Active development context
- Next steps and priorities

## Key Design Principles

- **Transparency**: Every decision should be visible and explained
- **Simplicity**: Focus on core concepts without enterprise complexity  
- **Educational**: Show the "magic" behind container orchestration
- **TDD**: Write tests first, implement to make tests pass
- **Hands-on**: Users should be able to experiment and see immediate results

When user says "let us wrap up for the day", update PROJECT_STATUS.md with current progress and context for seamless session resumption.