# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is the "gra" project - a cloud-native remote code execution system that enables users to run data analytics workloads in dynamically provisioned Kubernetes containers. The system consists of:

- **grad**: A gRPC service that manages runner lifecycle in Kubernetes
- **gractl**: A CLI tool for interacting with grad (both human and AI-friendly)
- **runners**: Dynamically created Kubernetes pods that execute user code with access to S3 data

## Development Commands

### Building Artifacts
```bash
# Always use make commands, NEVER use go build directly
make build          # Build both grad and gractl binaries
make build-gractl   # Build gractl CLI tool only
make build-all      # Build for multiple platforms (Linux, Darwin, Windows)
make test           # Run all tests
make clean          # Clean build artifacts

# Build artifacts are placed in the out/ directory
```

### Architecture Support
```bash
# Multi-architecture support (ARM64/AMD64)
# Containers automatically detect target architecture via Docker buildkit
# Supports both Intel/AMD (amd64) and Apple Silicon (arm64) development

# DuckDB CLI binary automatically downloads correct architecture:
# - arm64 → downloads arm64 binary  
# - amd64 → downloads amd64 binary
```

**Multi-Architecture Implementation**:

```dockerfile
# Docker multi-arch support (cmd/grad/Dockerfile:13-15)
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-amd64} go build -o grad ./cmd/grad
```

```dockerfile
# DuckDB CLI with correct ARM64 naming (devenv/runner/main/Dockerfile:16-22)
ARG TARGETARCH
RUN ARCH=${TARGETARCH:-amd64} && \
    curl -L https://github.com/duckdb/duckdb/releases/latest/download/duckdb_cli-linux-${ARCH}.zip -o /tmp/duckdb.zip && \
    unzip /tmp/duckdb.zip -d /tmp && \
    mv /tmp/duckdb /usr/local/bin/duckdb && \
    chmod +x /usr/local/bin/duckdb && \
    rm -f /tmp/duckdb.zip
```

**Note**: DuckDB uses "arm64" naming (not "aarch64") for ARM64 Linux binaries.

### Development Workflow
```bash
# Start development environment (includes minikube start)
make dev            # Starts skaffold dev with port forwarding
make dev-debug      # Development mode with debug output
make dev-stop       # Stop skaffold development

# Minikube management
make minikube-start     # Start minikube with 4C16G config
make minikube-stop      # Stop minikube
make minikube-status    # Show minikube and cluster status

# Verify skaffold configuration
skaffold diagnose
```

### Protocol Buffer Generation
```bash
make generate       # Regenerate protobuf code after changes to .proto files
buf generate        # Alternative: direct buf command (same as make generate)
```

## Architecture Overview

### Service Structure
```
/cmd/grad/          - Main gRPC service (deployed to Kubernetes)
/cmd/gractl/        - CLI tool for interacting with grad
/internal/grad/     - Core business logic
  /grpc/           - gRPC server implementation (thin controller layer)
  /service/        - Business logic and Kubernetes integration
/proto/grad/v1/    - Protocol buffer definitions
/gen/grad/v1/      - Generated protobuf code
```

### Key Design Patterns

1. **Clean Architecture**: 
   - gRPC layer (controller) → Service layer (business logic) → Kubernetes layer (infrastructure)
   - Domain types separate from protobuf types with conversion functions
   - RunnerStatus uses string constants ("creating", "running", "stopped", etc.)

2. **Resource Management**:
   - Hardcoded "small" preset (2c2g40g) for all runners
   - Runner images dynamically tagged by skaffold, use RUNNER_IMAGE env var to override

3. **Error Handling**:
   - Domain-specific errors mapped to gRPC status codes
   - Consistent error propagation through layers
   
4. **Kubernetes-Native Storage**:
   - No in-memory state - uses Kubernetes API as source of truth
   - Annotations for runner metadata (runner-id, status, created-at)
   - Labels for resource discovery and filtering
   - Finalizers for proper resource cleanup
   - Simple incrementing runner IDs (runner-1, runner-2, etc.)

### Core Components

**grad Service**:
- Manages runner lifecycle (create, delete, list, execute commands)
- Integrates with Kubernetes API to create/manage pods
- Exposes gRPC API on port 9090 and HTTP health/metrics on port 8080
- Supports streaming command execution with real-time stdout/stderr output
- Follows Go channel best practices (only sender closes channels)

**Runner Pods**:
- Dynamically created as Kubernetes pods
- Execute user commands in isolated environments
- Will support SSH access for file synchronization (future)
- Will mount S3 data via s3fs (future)

**gractl CLI**:
- Human-friendly commands with structured output
- Designed to be AI-tool friendly for integration with Gemini CLI
- Supports workspace management and runner operations
- Features streaming command execution with `--stream` flag

## Important Constraints

### Build Rules
- ❌ NEVER use `go build` directly - always use make commands
- ❌ NEVER use `go run -c` for testing code snippets
- ❌ NEVER build the main grad service - it's handled by skaffold dev
- ✅ Use `make build-gractl` for building the CLI tool
- ✅ Use `make test` for running tests

### Code Style
- ❌ NEVER use line-tail comments
- ✅ Use block comments above the code
- ✅ Follow existing patterns in the codebase

### Development Environment
- The main grad service runs in Kubernetes via skaffold
- Runners are created dynamically as pods (not managed by skaffold)
- Use Helm charts for deployment configuration
- Minikube requires 4 CPUs and 16GB RAM

## Current API Structure

The service exposes these gRPC methods:
- `CreateRunner` - Create a new runner instance with S3FS mount support and SSH key injection
- `DeleteRunner` - Remove a runner
- `ListRunners` - List all runners with optional filtering
- `GetRunner` - Get details of a specific runner
- `ExecuteCommand` - Execute a command in a runner (was ExecuteCode)
- `ExecuteCommandStream` - Execute a command with real-time stdout/stderr streaming

### Workspace Sync Feature

**NEW**: `gractl workspace sync` command enables local file synchronization with remote runners (all runners or specific runner).

**SSH Key Integration**: Runner creation automatically injects user's SSH public key:
```go
// Auto-inject SSH key in CreateRunner (cmd/gractl/cmd/runners.go:84-87)
if sshPublicKey, err := client.GetUserSSHPublicKey(); err == nil && sshPublicKey != "" {
    envMap["PUBLIC_KEY"] = sshPublicKey
}
```

**Local Workspace Mounting**: Mount remote `/workspace` to local directory:
```bash
# Example usage
gractl workspace sync runner-1    # Sync specific runner
gractl workspace sync             # Sync all running runners

# Creates and mounts to:
./runners/runner-1/workspace/
```

**Architecture**: Uses `kubectl port-forward` + `sshfs` for secure file synchronization.

### S3FS Integration

**Hardcoded Mount Path**: All S3 datasets are mounted at `/workspace/dataset` (not configurable)

```go
// S3FS sidecar configuration (pod_spec.go:243-249)
s3fsEnv = append(s3fsEnv, corev1.EnvVar{
    Name:  "MOUNT_PATH", 
    Value: "/workspace/dataset",  // Always hardcoded
})
```

**WorkspaceConfig Changes**: Removed mount_path field from protobuf definition:
```proto
message WorkspaceConfig {
    string bucket = 1;
    string endpoint = 2; 
    string prefix = 3;
    string region = 4;
    bool read_only = 5;  // mount_path removed in favor of hardcoded path
}
```

### gRPC Streaming Error Handling

**EOF Handling**: Fixed proper EOF detection in both client and server:

```go
// Client-side EOF handling (cmd/gractl/cmd/runners.go:89-95)
for {
    resp, err := stream.Recv()
    if err != nil {
        if err == io.EOF {
            break  // Normal stream termination
        }
        fmt.Fprintf(os.Stderr, "Stream error: %v\n", err)
        os.Exit(1)
    }
    // Process response...
}
```

```go
// Server-side channel closure handling (internal/grad/grpc/server.go:191-196)
case err, ok := <-errCh:
    if !ok {
        // errCh was closed, no error to handle
        continue
    }
    return s.mapServiceError(err)
```

## Testing

The project separates unit tests from integration tests to improve development velocity and CI/CD pipelines.

### Unit Tests (Fast, No Dependencies)
```bash
# Run unit tests only (default)
make test

# Tests pure business logic without external dependencies
# Located alongside source files (*_test.go)
# Key unit test files:
# - internal/grad/service/types_test.go (domain type conversions)
# - internal/grad/service/pod_spec_test.go (pod specification generation)
# - internal/grad/service/runner_test.go (domain object tests)
# - internal/grad/service/activity_test.go (activity tracking)
# - internal/grad/service/cleanup_test.go (cleanup service)
```

### Integration Tests (Require Kubernetes)
```bash
# Run integration tests (requires Kubernetes cluster)
make test-integration

# Tests that interact with real Kubernetes API
# Files marked with //go:build integration tag
# Key integration test files:
# - internal/grad/service/runner_integration_test.go (TestRunnerServiceBasics)
```

### All Tests
```bash
# Run both unit and integration tests
make test-all
```

### Test Strategy
- **Unit tests**: Fast feedback, no external dependencies, run in CI/CD
- **Integration tests**: Comprehensive end-to-end validation, require cluster
- **Build tags**: `//go:build integration` separates test types
- **CI Strategy**: Run unit tests on every commit, integration tests on deployment

## Collaboration Patterns

### Claude's Responsibilities:
- ✅ Code writing and file operations
- ✅ Short-term builds (`buf generate`, `make build-gractl`)
- ✅ Quick testing commands (`curl`, `grpcurl`)
- ✅ File system operations
- ✅ Code generation and configuration

### User's Responsibilities:
- 🚀 Starting long-running services (`skaffold dev`)
- 🚀 Building the main grad service (handled by skaffold)
- 🚀 Environment setup (`minikube start`)
- 🚀 Interactive operations
- 🚀 Port forwarding and network configuration

### Testing Requirements:
- ⚠️ Before testing gractl commands, always ask user to start grad server with `skaffold dev`
- ⚠️ Tests that require Kubernetes connectivity will fail without running server

## Common Tasks

### Adding a New gRPC Method
1. Update `proto/grad/v1/runner_service.proto`
2. Run `buf generate` to regenerate code
3. Implement the method in `internal/grad/grpc/server.go`
4. Add business logic in `internal/grad/service/`
5. Update domain types if needed in `internal/grad/service/types.go`
6. Add tests for new functionality

### Modifying Runner Resources
- Edit the `RunnerSpecPreset` in `internal/grad/service/kubernetes.go`
- Currently only "small" preset is used (2c2g40g)
- Update `createPodSpec` in `internal/grad/service/pod_spec.go` if needed

### Working with Skaffold
- Configuration is in `skaffold.yaml`
- Uses Helm for deployment (`devenv/helm/grad/`)
- Dynamic image tags require RUNNER_IMAGE env var override
- Port forwarding automatically configured for local development

