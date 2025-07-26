# Development Environment Guide for Claude Code

This file provides specific guidance for Claude Code when working with the development environment setup and configuration in the gra project.

## Development Environment Overview

The devenv directory contains all configurations needed to run the gra system in a local Kubernetes development environment using Skaffold, Helm, and Minikube.

### Directory Structure

```
devenv/
├── helm/grad/          # Helm chart for grad service deployment
│   ├── Chart.yaml      # Helm chart metadata
│   ├── values.yaml     # Default configuration values
│   └── templates/      # Kubernetes manifest templates
└── runner/             # Runner container configuration
    ├── Dockerfile      # Runner image definition
    └── entrypoint.sh   # Runner startup script
```

## Key Components

### Helm Chart Configuration

- **Location**: `devenv/helm/grad/`
- **Purpose**: Defines how the grad service is deployed to Kubernetes
- **Key Files**:
  - `values.yaml`: Contains all configurable parameters (ports, resources, images)
  - `templates/`: Kubernetes manifests (deployment, service, configmap, RBAC)

### Runner Container

- **Base Image**: `python:3.11-slim`
- **Capabilities**: SSH access, S3FS mounting, Python data analytics stack
- **Python Packages**: pandas, numpy, duckdb, pyarrow, jupyter, matplotlib, seaborn, scipy, scikit-learn
- **User Setup**: Non-root `runner` user with workspace at `/workspace/`
- **SSH**: Enabled on port 22 for future file synchronization features
- **S3FS**: Pre-installed for future S3 data mounting

### Skaffold Profiles

- **development**: Local builds, no image push, default for `skaffold dev`
- **production**: Builds and pushes images to registry
- **debug**: Enables verbose logging and debug builds
- **extended**: Includes gractl binary in addition to grad service

## Configuration Values

### Service Configuration (values.yaml)

```yaml
# Main service ports
service:
  http:
    port: 8080      # Health/metrics endpoint
  grpc:
    port: 9090      # Main gRPC API

# Resource allocation
resources:
  requests:
    memory: "128Mi"
    cpu: "100m"
  limits:
    memory: "512Mi"
    cpu: "500m"

# Health check endpoints
probes:
  liveness:
    path: /health   # HTTP health check
  readiness:
    path: /ready    # HTTP readiness check
```

### Image Configuration

- **grad service**: Built from `cmd/grad/Dockerfile`
- **runner**: Built from `devenv/runner/main/Dockerfile`
- **s3fs sidecar**: Built from `devenv/runner/s3fs/Dockerfile`
- **Tags**: Automatically generated from git commit hash via Skaffold
- **Architecture**: Multi-arch support (ARM64/AMD64) via Docker buildkit TARGETARCH

## Development Workflow Commands

### Essential Commands

```bash
# Start development environment
make dev                    # Starts skaffold dev with port forwarding
make dev-debug             # Development with debug logging
make dev-stop              # Stop skaffold development

# Build runner image separately (if needed)
docker build -f devenv/runner/Dockerfile -t grad-runner .

# Test Helm chart locally
helm template grad devenv/helm/grad --values devenv/helm/grad/values.yaml
```

### Port Forwarding (Automatic via Skaffold)

- HTTP/Health: `localhost:8080` → `grad-service:8080`
- gRPC API: `localhost:9090` → `grad-service:9090`

### Skaffold Configuration

- **Config**: `skaffold.yaml` in project root
- **Build Strategy**: Git commit-based tagging
- **Local Development**: No image push, Docker CLI builds
- **Helm Integration**: Automatic value injection for image tags

## Environment Requirements

### Minikube Setup

```bash
# Required resources for development
minikube start --cpus=4 --memory=16384

# Enable required addons
minikube addons enable ingress
```

### Required Tools

- Docker (for building images)
- Minikube (local Kubernetes cluster)
- Skaffold (development workflow)
- Helm (package manager)
- kubectl (Kubernetes CLI)

## Security Configuration

### Service Account

- **Name**: `grad-service-account`
- **Purpose**: Kubernetes API access for runner pod management
- **RBAC**: Defined in `templates/rbac.yaml`

### Container Security

- **Non-root execution**: Service runs as user 1000
- **File system group**: 2000
- **SSH access**: Enabled in runner containers only

## Troubleshooting Common Issues

### Runner Image Issues

```bash
# Rebuild runner image if Python packages are outdated
skaffold build --profile=development

# Check runner container logs
kubectl logs -l app=grad-runner
```

### Port Forwarding Issues

```bash
# Verify services are running
kubectl get svc grad-service

# Manual port forwarding if skaffold fails
kubectl port-forward svc/grad-service 8080:8080 9090:9090
```

### Helm Template Debugging

```bash
# Validate Helm templates
helm template grad devenv/helm/grad --debug

# Check for template syntax errors
helm lint devenv/helm/grad
```

## Configuration Customization

### Overriding Values

Create a custom values file for local development:

```bash
# Create custom-values.yaml
cp devenv/helm/grad/values.yaml custom-values.yaml

# Modify custom-values.yaml as needed, then:
helm template grad devenv/helm/grad -f custom-values.yaml
```

### Environment Variables

Key environment variables in grad service:

- `PORT`: HTTP server port (default: 8080)
- `GRPC_PORT`: gRPC server port (default: 9090)
- `LOG_LEVEL`: Logging verbosity (info, debug, error)
- `RUNNER_IMAGE`: Override runner container image

### Adding New Dependencies to Runner

To add Python packages or system dependencies:

1. Edit `devenv/runner/Dockerfile`
2. Add packages to appropriate RUN command
3. Rebuild with `skaffold build`

## Integration Points

### Grad Service Integration

- Uses Helm chart values for configuration
- Kubernetes API access via service account
- Creates runner pods dynamically using runner image

### Runner Pod Integration

- Launched by grad service on-demand
- SSH access for file synchronization (future)
- S3FS mounting for data access (future)
- Isolated workspace at `/workspace/`

### Development Integration

- Skaffold handles build and deployment
- Port forwarding for local testing
- Git-based image tagging for consistency

## Future Enhancements

### Planned Features

- S3 bucket mounting via s3fs in runner containers
- SSH key management for secure file transfer
- Multiple runner presets (small, medium, large)
- Registry configuration for runner image distribution

### Configuration Extensions

- Support for custom runner base images
- Environment-specific value overrides
- Resource scaling based on workload requirements
