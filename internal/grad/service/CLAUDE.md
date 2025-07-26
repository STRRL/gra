# Service Layer - CLAUDE.md

This file provides guidance to Claude Code when working with the service layer of the gra project.

## Service Layer Overview

The `internal/grad/service` package contains the core business logic for the gra system. It acts as the domain layer between the gRPC controllers and the Kubernetes infrastructure.

## Key Files and Responsibilities

### Core Domain Logic

- **types.go**: Domain types and models (Runner, RunnerStatus, etc.)
- **runner.go**: Core runner business logic and lifecycle management
- **config.go**: Service configuration and settings
- **activity.go**: In-memory activity tracking for runner cleanup
- **cleanup.go**: Background service for inactive runner cleanup

### Kubernetes Integration

- **kubernetes.go**: Kubernetes client wrapper and resource management
- **pod_spec.go**: Pod specification generation and configuration

### Testing

- **types_test.go**: Domain type conversion and validation tests
- **runner_test.go**: Runner business logic tests
- **pod_spec_test.go**: Pod specification generation tests
- **activity_test.go**: Activity tracker unit tests
- **cleanup_test.go**: Cleanup service unit tests

## Important Design Patterns

### Domain-Driven Design

- Domain types are separate from protobuf types
- Conversion functions handle type mapping
- Business logic is isolated from infrastructure concerns

### Status Management

- RunnerStatus uses string constants: "creating", "running", "stopped", "failed"
- Status transitions are handled at the service layer
- Kubernetes annotations store runner metadata

### Resource Management

- No persistent in-memory state - Kubernetes API is the source of truth for runner state
- Runner IDs are simple incrementing integers (runner-1, runner-2, etc.)
- Hardcoded "small" preset (2c2g40g) for all runners
- Activity tracking maintained in memory for cleanup purposes

### Activity Tracking and Cleanup

- **ActivityTracker**: Thread-safe in-memory tracking of runner last active times
- **CleanupService**: Background service that runs every 1 minute to check for inactive runners
- **Cleanup Policy**: Deletes runners inactive for >5 minutes, but only running/creating runners
- **Safety**: Stopped/error runners are skipped, runners with no activity are ignored

### Error Handling

- Domain-specific errors are defined in this layer
- Errors are mapped to appropriate gRPC status codes at the controller layer
- Consistent error propagation through all service methods

## Key Constraints

### Testing Requirements

- All business logic must have unit tests
- Use table-driven tests for multiple scenarios
- Mock Kubernetes client for testing
- Test error conditions and edge cases

### Code Style

- Follow clean architecture principles
- Keep business logic separate from infrastructure
- Use dependency injection for testability
- Document complex business rules

### Kubernetes Integration

- Use labels for resource discovery
- Use annotations for metadata storage
- Use finalizers for cleanup guarantees
- Handle Kubernetes API errors gracefully

## Common Tasks

### Adding New Runner Operations

1. Define domain types in `types.go`
2. Add business logic to `runner.go`
3. Create unit tests in `runner_test.go`
4. Update Kubernetes integration if needed

### Modifying Resource Specifications

1. Update pod specification in `pod_spec.go`
2. Add tests in `pod_spec_test.go`
3. Consider backwards compatibility
4. Update resource presets if needed

### Adding Configuration Options

1. Define config structs in `config.go`
2. Add validation logic
3. Update service initialization
4. Add environment variable support

### Working with Activity Tracking

1. Activity is automatically tracked when `ExecuteCommandStream` is called
2. Use `ActivityTracker.UpdateLastActiveTime()` to record activity
3. `ActivityTracker.GetInactiveRunners()` returns runners inactive for a given duration
4. Always call `ActivityTracker.RemoveRunner()` when deleting runners

### Modifying Cleanup Behavior

1. Cleanup intervals are configured in `CleanupService` (1 min check, 5 min timeout)
2. Cleanup logic is in `CleanupService.deleteInactiveRunner()`
3. Only running/creating runners are deleted (stopped/error runners are skipped)
4. Add tests in `cleanup_test.go` for new cleanup scenarios

## Testing Guidelines

```bash
# Run service layer tests
go test ./internal/grad/service/...

# Run with coverage
go test -cover ./internal/grad/service/...

# Run specific test file
go test ./internal/grad/service/runner_test.go

# Run activity tracker tests
go test ./internal/grad/service/activity_test.go ./internal/grad/service/activity.go -v

# Run cleanup service tests
go test ./internal/grad/service/cleanup_test.go ./internal/grad/service/cleanup.go ./internal/grad/service/activity.go ./internal/grad/service/types.go -v
```

## Integration Points

### With gRPC Layer

- Service methods are called by gRPC handlers
- Domain types are converted to/from protobuf types
- Service errors are mapped to gRPC status codes

### With Kubernetes

- Uses Kubernetes client-go library
- Manages pod lifecycle through Kubernetes API
- Stores state in Kubernetes resources (annotations, labels)

### With Runners

- Creates and manages runner pods
- Executes commands via Kubernetes exec API
- Handles real-time streaming of command output
- Tracks runner activity for automatic cleanup

### With Cleanup System

- ActivityTracker integrates with ExecuteCommandStream to record activity
- CleanupService runs as background goroutine in main service
- Proper shutdown handling for cleanup service lifecycle
- Integration with runner service for safe deletion

## Service Initialization Pattern

The service layer follows a specific initialization pattern with dependency injection:

```go
// Initialize Kubernetes client
k8sClient, err := service.NewKubernetesClient(config.Kubernetes)

// Initialize activity tracker for runner cleanup
activityTracker := service.NewActivityTracker()

// Initialize runner service with dependencies
runnerService := service.NewRunnerService(k8sClient, activityTracker)

// Initialize cleanup service
cleanupService := service.NewCleanupService(runnerService, activityTracker)

// Start cleanup as background goroutine
ctx, cancelCleanup := context.WithCancel(context.Background())
go func() {
    cleanupService.Start(ctx)
}()

// Proper shutdown
cancelCleanup()
cleanupService.Stop()
```

### Key Dependencies

- **RunnerService** requires: KubernetesClient, ActivityTracker
- **CleanupService** requires: RunnerService, ActivityTracker  
- **ActivityTracker** has no dependencies (standalone)
- **KubernetesClient** requires: KubernetesConfig

## Security Considerations

- Validate all input parameters
- Sanitize command execution inputs
- Use proper RBAC for Kubernetes access
- Never log sensitive information
- Implement proper resource quotas and limits
- Activity tracking data is kept in memory only (not persisted)
- Cleanup service only deletes runners it can verify are safe to delete
