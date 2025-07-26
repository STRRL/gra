# CLAUDE.md - Command Line Components

This file provides specific guidance for working with the command-line components (`grad` and `gractl`) in the gra project.

## Component Overview

### grad (Server)

**Location**: `/cmd/grad/`

- **Purpose**: Main gRPC service that manages runner lifecycle in Kubernetes
- **Architecture**: Dual HTTP/gRPC server with Prometheus metrics
- **Ports**:
  - gRPC API: 9090 (configurable via `--grpc-port`)  
  - HTTP health/metrics: 8080 (configurable via `--http-port`)
- **Deployment**: Runs in Kubernetes via skaffold, NOT built locally

### gractl (Client)

**Location**: `/cmd/gractl/`

- **Purpose**: CLI tool for interacting with grad service
- **Framework**: Cobra CLI framework
- **Usage**: Both human-friendly and AI-tool friendly
- **Build**: Use `make build-gractl` for local development

## Development Guidelines

### Building Rules

```bash
# ✅ CORRECT - Build gractl for local testing
make build-gractl

# ✅ CORRECT - Build both components  
make build

# ❌ WRONG - Never build grad directly (handled by skaffold)
go build ./cmd/grad

# ❌ WRONG - Never use go build directly for any component
go build ./cmd/gractl
```

### grad Service Development

**Key Characteristics**:

- Runs as dual HTTP/gRPC server with structured logging (slog)
- Exposes Prometheus metrics on HTTP endpoints
- Uses Gin for HTTP routing and standard gRPC server
- Implements graceful shutdown with signal handling
- Loads configuration from environment and service layer

**Important Notes**:

- ❌ Never run or build grad locally - it's managed by skaffold dev
- ✅ Configuration comes from `service.LoadConfig()`
- ✅ Kubernetes client initialized via `service.NewKubernetesClient()`
- ✅ Runner service dependency injected into gRPC server
- ✅ All logging uses structured logging (slog) with JSON output

**HTTP Endpoints**:

- `/health` - Health check (200 OK)
- `/ready` - Readiness check (200 OK)  
- `/metrics` - Prometheus metrics endpoint

**gRPC Features**:

- Implements `gradv1.RunnerServiceServer` interface
- Reflection enabled for grpcurl testing
- Prometheus metrics for request counting and duration

### gractl CLI Development

**Key Characteristics**:

- Built with Cobra framework for command structure
- Uses gRPC client for grad service communication
- Supports streaming command execution with `--stream` flag
- Designed for both human and AI tool integration

**Command Structure**:

```
gractl
└── runners (main command group)
    ├── create
    ├── delete  
    ├── list
    ├── get
    ├── exec
    └── workspace-sync (NEW: mount remote workspace locally)
```

**Client Architecture**:

- Client logic in `/cmd/gractl/client/client.go`
- SSH utilities in `/cmd/gractl/client/ssh.go` (NEW: SSH key management, local directory handling)
- Command implementations in `/cmd/gractl/cmd/`
- Workspace sync in `/cmd/gractl/cmd/workspace_sync.go` (NEW: sshfs + kubectl port-forward)
- Main entry point in `/cmd/gractl/main.go`

## Configuration Patterns

### grad Service Configuration

- Environment-based configuration via `service.LoadConfig()`
- Runner image configured via `RUNNER_IMAGE` environment variable
- Kubernetes configuration for cluster connectivity
- Structured logging with JSON output format

### gractl Client Configuration  

- Connection to grad service (typically localhost:9090 in dev)
- Output formatting options for human vs programmatic use
- Streaming vs batch execution modes
- Proper EOF handling for gRPC streaming (fixed spurious "Stream error" messages)

## Testing Patterns

### Integration Testing Requirements

- ⚠️ grad service must be running via `skaffold dev` before testing gractl
- ⚠️ Kubernetes cluster (minikube) must be available for grad service
- ✅ Use `grpcurl` for direct gRPC API testing
- ✅ Use `curl` for HTTP endpoint testing

### Common Test Commands

```bash
# Test HTTP health endpoint (requires grad service running)
curl http://localhost:8080/health

# Test gRPC reflection (requires grad service running)  
grpcurl -plaintext localhost:9090 list

# Test gractl commands (requires grad service running)
./out/gractl runners list
```

## Error Handling Patterns

### grad Service

- Structured logging with context using slog
- gRPC status codes mapped from domain errors
- Graceful shutdown handling for both HTTP and gRPC servers
- Prometheus metrics for monitoring error rates

### gractl Client

- Cobra command error handling with os.Exit(1)
- gRPC client error handling and user-friendly error messages
- Streaming error handling for real-time command execution
- Fixed EOF detection: `err == io.EOF` (not string comparison) for proper stream termination

## Dependencies and Imports

### grad Service Key Dependencies

```go
"github.com/gin-gonic/gin"           // HTTP router
"github.com/prometheus/client_golang" // Metrics
"google.golang.org/grpc"             // gRPC server
"github.com/spf13/cobra"             // CLI framework
```

### gractl Client Key Dependencies  

```go
"github.com/spf13/cobra"             // CLI framework
// gRPC client dependencies for grad communication
```

## Common Development Tasks

### Adding New gractl Command

1. Create command in `/cmd/gractl/cmd/`
2. Register command in `/cmd/gractl/main.go` init function
3. Implement client logic in `/cmd/gractl/client/client.go` if needed
4. Build with `make build-gractl` and test

### Adding grad Service Endpoint

1. Update protobuf definitions if needed (`buf generate`)
2. Implement gRPC method in `internal/grad/grpc/server.go`
3. Add business logic in `internal/grad/service/`
4. Test via skaffold dev (never build grad locally)

### Debugging Connection Issues

- Verify grad service is running: `curl http://localhost:8080/health`
- Check gRPC connectivity: `grpcurl -plaintext localhost:9090 list`
- Verify port forwarding in skaffold configuration
- Check Kubernetes connectivity and minikube status
