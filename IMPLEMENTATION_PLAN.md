# Dubernetes Implementation Plan

## Overview
Building Dubernetes using Test-Driven Development (TDD) approach. Each phase starts with writing tests, then implementing the functionality to make tests pass.

## Project Structure
```
dubernetes/
├── cmd/
│   ├── dubectl/          # CLI client
│   │   └── main.go
│   └── orchestrator/     # Server/orchestrator
│       └── main.go
├── pkg/
│   ├── api/             # REST API definitions
│   │   ├── handlers.go
│   │   ├── handlers_test.go
│   │   ├── types.go
│   │   └── types_test.go
│   ├── config/          # Configuration management
│   │   ├── config.go
│   │   └── config_test.go
│   ├── database/        # SQLite operations
│   │   ├── db.go
│   │   └── db_test.go
│   ├── docker/          # Docker wrapper
│   │   ├── client.go
│   │   └── client_test.go
│   ├── nginx/           # Nginx config generation
│   │   ├── proxy.go
│   │   └── proxy_test.go
│   └── reconciler/      # State reconciliation loop
│       ├── reconciler.go
│       └── reconciler_test.go
├── test/
│   ├── integration/     # End-to-end tests
│   └── fixtures/        # Test data
├── configs/
│   └── default.yaml     # Default configuration
├── examples/
│   ├── nginx.yaml
│   └── multi-app.yaml
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Phase 1: Core Infrastructure & Configuration

### 1.1 Project Setup
- [ ] Initialize Go module (`go mod init github.com/mansoormajeed/dubernetes`)
- [ ] Set up basic project structure (directories)
- [ ] Create initial Makefile with test/build/run commands
- [ ] Set up GitHub Actions for CI/CD (optional)

### 1.2 Configuration Management (TDD)
- [ ] **Write tests first**: `pkg/config/config_test.go`
  - [ ] Test loading config from YAML file
  - [ ] Test environment variable overrides
  - [ ] Test default values when no config provided
  - [ ] Test validation of port ranges
- [ ] **Implement**: `pkg/config/config.go`
  - [ ] Config struct with all required fields
  - [ ] LoadConfig function with YAML parsing
  - [ ] Environment variable override logic
  - [ ] Configuration validation
- [ ] **Verify tests pass**

### 1.3 Database Layer (TDD)
- [ ] **Write tests first**: `pkg/database/db_test.go`
  - [ ] Test database initialization and schema creation
  - [ ] Test CRUD operations for pods table
  - [ ] Test CRUD operations for services table
  - [ ] Test unique constraints (name, replica_id)
  - [ ] Test database cleanup/reset for tests
- [ ] **Implement**: `pkg/database/db.go`
  - [ ] SQLite connection management
  - [ ] Schema creation and migrations
  - [ ] Pod and Service structs
  - [ ] Database interface with all CRUD methods
- [ ] **Verify tests pass**

## Phase 2: Docker Integration (TDD)

### 2.1 Docker Client (TDD)
- [ ] **Write tests first**: `pkg/docker/client_test.go`
  - [ ] Test container creation with proper labels
  - [ ] Test container status checking
  - [ ] Test container stopping and removal
  - [ ] Test port allocation from range
  - [ ] Test port conflict detection
  - [ ] Test error handling for Docker command failures
  - [ ] Mock Docker commands for testing
- [ ] **Implement**: `pkg/docker/client.go`
  - [ ] DockerClient struct and interface
  - [ ] RunContainer function with port mapping
  - [ ] StopContainer and RemoveContainer functions
  - [ ] IsContainerRunning status check
  - [ ] Port allocation logic with conflict detection
  - [ ] Proper Docker labels for container tracking
- [ ] **Verify tests pass**

### 2.2 Integration Tests for Docker
- [ ] **Write integration tests**: `test/integration/docker_test.go`
  - [ ] Test actual Docker container lifecycle
  - [ ] Test port accessibility from host
  - [ ] Test container restart scenarios
  - [ ] Cleanup containers after tests

## Phase 3: REST API Server (TDD)

### 3.1 API Types (TDD)
- [ ] **Write tests first**: `pkg/api/types_test.go`
  - [ ] Test YAML parsing of PodSpec
  - [ ] Test JSON marshaling/unmarshaling
  - [ ] Test validation of required fields
  - [ ] Test default values (replicas = 1)
- [ ] **Implement**: `pkg/api/types.go`
  - [ ] PodSpec, AccessSpec, PodStatus structs
  - [ ] JSON/YAML tags
  - [ ] Validation methods
  - [ ] Default value handling
- [ ] **Verify tests pass**

### 3.2 API Handlers (TDD)
- [ ] **Write tests first**: `pkg/api/handlers_test.go`
  - [ ] Test POST /api/apply endpoint
  - [ ] Test GET /api/pods endpoint
  - [ ] Test DELETE /api/delete endpoint
  - [ ] Test error responses (400, 404, 500)
  - [ ] Test request/response JSON format
  - [ ] Mock database and docker client for testing
- [ ] **Implement**: `pkg/api/handlers.go`
  - [ ] HTTP handlers for all endpoints
  - [ ] Request validation and error handling
  - [ ] Integration with database and docker client
  - [ ] Proper HTTP status codes and responses
- [ ] **Verify tests pass**

### 3.3 HTTP Server
- [ ] **Write tests**: Test server startup and shutdown
- [ ] **Implement**: `cmd/orchestrator/main.go`
  - [ ] HTTP server setup with all routes
  - [ ] Graceful shutdown handling
  - [ ] Configuration loading
  - [ ] Dependency injection (db, docker, etc.)

## Phase 4: Nginx Proxy Management (TDD)

### 4.1 Nginx Config Generator (TDD)
- [ ] **Write tests first**: `pkg/nginx/proxy_test.go`
  - [ ] Test nginx config template generation
  - [ ] Test upstream block creation for multiple backends
  - [ ] Test server block with host routing
  - [ ] Test config file writing
  - [ ] Test nginx process management (start/stop/reload)
  - [ ] Mock file system operations for testing
- [ ] **Implement**: `pkg/nginx/proxy.go`
  - [ ] Nginx config templates
  - [ ] Config generation from service data
  - [ ] File writing and nginx process management
  - [ ] Error handling for nginx operations
- [ ] **Verify tests pass**

### 4.2 Integration Tests for Nginx
- [ ] **Write integration tests**: `test/integration/nginx_test.go`
  - [ ] Test actual nginx config generation and reload
  - [ ] Test HTTP requests routing through nginx
  - [ ] Test load balancing across multiple backends

## Phase 5: Reconciliation Loop (TDD)

### 5.1 State Reconciler (TDD)
- [ ] **Write tests first**: `pkg/reconciler/reconciler_test.go`
  - [ ] Test reconciliation of desired vs actual state
  - [ ] Test container restart with exponential backoff
  - [ ] Test failed container detection and handling
  - [ ] Test nginx config updates when pods change
  - [ ] Test reconciliation loop timing and cancellation
  - [ ] Mock all dependencies (docker, database, nginx)
- [ ] **Implement**: `pkg/reconciler/reconciler.go`
  - [ ] Reconciler struct with all dependencies
  - [ ] Main reconciliation loop with context cancellation
  - [ ] Pod state reconciliation logic
  - [ ] Container restart logic with backoff
  - [ ] Nginx config update triggers
- [ ] **Verify tests pass**

### 5.2 Restart Logic (TDD)
- [ ] **Write tests**: Test exponential backoff calculation
- [ ] **Implement**: Restart backoff algorithm
- [ ] **Write tests**: Test max restart limits
- [ ] **Implement**: Restart limit enforcement

## Phase 6: CLI Client (TDD)

### 6.1 CLI Commands (TDD)
- [ ] **Write tests first**: `cmd/dubectl/main_test.go`
  - [ ] Test YAML file parsing
  - [ ] Test HTTP API communication
  - [ ] Test command-line argument parsing
  - [ ] Test output formatting (tables)
  - [ ] Test error handling and user messages
  - [ ] Mock HTTP client for testing
- [ ] **Implement**: `cmd/dubectl/main.go`
  - [ ] Command-line interface with cobra/flag
  - [ ] HTTP client for API communication
  - [ ] YAML file reading and parsing
  - [ ] Pretty-printed table output
  - [ ] User-friendly error messages
- [ ] **Verify tests pass**

### 6.2 CLI Integration Tests
- [ ] **Write integration tests**: Test full CLI workflow
  - [ ] Test apply -> get -> delete flow
  - [ ] Test error scenarios
  - [ ] Test output formatting

## Phase 7: End-to-End Integration & Testing

### 7.1 Integration Test Suite
- [ ] **Write comprehensive integration tests**: `test/integration/e2e_test.go`
  - [ ] Test complete deployment workflow
  - [ ] Test load balancing across replicas
  - [ ] Test container restart scenarios
  - [ ] Test nginx config updates
  - [ ] Test state persistence across orchestrator restarts
  - [ ] Test cleanup and resource management

### 7.2 Example Applications
- [ ] Create sample YAML files in `examples/`
  - [ ] Simple nginx deployment
  - [ ] Multi-replica application
  - [ ] Multiple services with different hosts
- [ ] Test all examples work end-to-end

### 7.3 Performance & Reliability Tests
- [ ] **Write performance tests**
  - [ ] Test with multiple replicas (10+)
  - [ ] Test rapid apply/delete cycles
  - [ ] Test resource cleanup
- [ ] **Write reliability tests**
  - [ ] Test orchestrator restart scenarios
  - [ ] Test database corruption recovery
  - [ ] Test docker daemon restart handling

## Phase 8: Documentation & Polish

### 8.1 Documentation
- [ ] Update README.md with installation and usage
- [ ] Add inline code documentation
- [ ] Create user guide with examples
- [ ] Document configuration options

### 8.2 Build & Distribution
- [ ] Finalize Makefile with all commands
- [ ] Add version information to binaries
- [ ] Create release builds for different platforms
- [ ] Set up proper logging throughout the application

## Testing Strategy

### Unit Tests
- All packages must have >80% test coverage
- Use table-driven tests where appropriate
- Mock external dependencies (Docker, file system, network)
- Fast execution (<5 seconds for all unit tests)

### Integration Tests
- Test real Docker containers
- Test actual nginx configuration and routing
- Test database operations with real SQLite
- Proper cleanup after each test

### Test Commands
```makefile
test:           # Run all unit tests
test-integration: # Run integration tests (requires Docker)
test-coverage:  # Generate coverage report
test-race:      # Run tests with race detection
```

## Success Criteria
- [ ] All tests pass consistently
- [ ] Can deploy nginx with 3 replicas
- [ ] Can access service via host-based routing
- [ ] Containers restart automatically when killed
- [ ] Load balancing works across replicas
- [ ] State persists across orchestrator restarts
- [ ] Clean resource cleanup on delete