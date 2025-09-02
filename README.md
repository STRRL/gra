# Gra

A cloud-native remote code execution system for running data analytics workloads in Kubernetes.

## What it does

- **grad**: gRPC service that manages runner lifecycle in Kubernetes
- **gractl**: CLI tool to create runners and execute commands
- **runners**: Isolated Kubernetes pods for code execution with S3 data access

## Quick Start

```bash
# Build the CLI
make build-gractl

# Start development environment
make dev

# Create a runner
./out/gractl runners create

# Execute commands
./out/gractl runners execute runner-1 "python analysis.py"

# Sync files locally
./out/gractl workspace sync runner-1
```

## Requirements

- Kubernetes cluster (minikube supported)
- 4 CPUs, 16GB RAM for development

## Development

```bash
make build-gractl   # Build CLI only
make test          # Run unit tests
make generate      # Regenerate protobuf code
make dev           # Start with skaffold (includes minikube)
```

## Architecture

- gRPC API on port 9090
- HTTP health/metrics on port 8080
- Runners mount S3 data at `/workspace/dataset`
- SSH access for file synchronization