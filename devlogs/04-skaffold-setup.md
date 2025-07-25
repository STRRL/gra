# Skaffold Development Setup

## Overview

Skaffold is used to accelerate the development workflow of grad service in minikube. It provides continuous development capabilities with automatic building, testing, and deployment.

## What is Skaffold

Skaffold is an open-source command-line tool from Google that:

- Handles building, pushing, and deploying applications to Kubernetes
- Provides instant development feedback through continuous workflows
- Works lightweight (client-side only, no cluster overhead)
- Supports multiple build and deployment tools

## Key Benefits for Grad Development

1. **Fast Iteration**: Changes take seconds instead of minutes
2. **Easy Sharing**: "git clone, then skaffold run" workflow
3. **Multi-Environment**: Works across different Kubernetes environments
4. **Integrated Workflow**: Combines build, test, and deploy in one tool

## Installation

```bash
# Install Skaffold
curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/latest/skaffold-linux-amd64
sudo install skaffold /usr/local/bin/

# Verify installation
skaffold version
```

## Development Workflow

### 1. Initialize Project

```bash
# Generate skaffold.yaml automatically
skaffold init

# Or use our pre-configured skaffold.yaml
```

### 2. Continuous Development

```bash
# Start development mode - watches files and auto-deploys
skaffold dev

# Development with port forwarding
skaffold dev --port-forward
```

### 3. Build and Test

```bash
# Build container images
skaffold build

# Run tests
skaffold test

# Build and output artifacts
skaffold build --file-output=artifacts.json
```

### 4. Deploy

```bash
# Deploy to cluster
skaffold deploy

# Render manifests (GitOps workflow)
skaffold render --output=manifests.yaml
skaffold apply manifests.yaml
```

## Skaffold Pipeline Stages

### 1. File Watching

- Monitors source code changes
- Triggers rebuilds automatically
- Supports different watch modes (filesystem notifications, polling, manual)

### 2. Build

- Uses Docker builder for grad service
- Builds container images from Dockerfile
- Supports build arguments and custom contexts

### 3. Test (Optional)

- Container structure tests
- Custom test scripts
- Validates built images before deployment

### 4. Deploy

- Uses kubectl deployer
- Applies Kubernetes manifests
- Supports custom namespaces and configurations

### 5. File Sync (Development)

- Hot-reloads code changes
- Syncs files directly to running pods
- Faster than full rebuild/redeploy cycle

## Configuration Structure

The `skaffold.yaml` file defines:

```yaml
apiVersion: skaffold/v4beta7
kind: Config
metadata:
  name: grad-development

# Build configuration
build:
  artifacts:
    - image: grad
      docker:
        dockerfile: cmd/grad/Dockerfile

# Test configuration (optional)
test:
  - image: grad
    structureTests:
      - ./test/structure-test.yaml

# Deploy configuration
deploy:
  kubectl:
    manifests:
      - k8s/grad-deployment.yaml
      - k8s/grad-service.yaml

# Development profiles
profiles:
  - name: development
    build:
      local:
        push: false
  - name: production
    build:
      local:
        push: true
```

## Best Practices for Grad Development

### 1. Use Development Profile

```bash
# Use development profile (no image push)
skaffold dev -p development
```

### 2. Port Forwarding

```bash
# Forward grad gRPC service port
skaffold dev --port-forward=true
```

### 3. File Sync Configuration

```yaml
build:
  artifacts:
    - image: grad
      sync:
        manual:
          - src: "**/*.go"
            dest: /app
            strip: ""
```

### 4. Extended Development (with gractl)

```yaml
# For grad service + gractl development
build:
  artifacts:
    - image: grad
      context: .
      docker:
        dockerfile: cmd/grad/Dockerfile
    - image: gractl
      context: .
      docker:
        dockerfile: cmd/gractl/Dockerfile
```

**Note**: While runners are dynamically created by the grad service as Kubernetes pods, their container images are pre-built by Skaffold to ensure they're available when needed.

## Common Commands

```bash
# Start development with file watching
skaffold dev

# Run once (build, test, deploy)
skaffold run

# Clean up deployed resources
skaffold delete

# Debug mode with verbose output
skaffold dev -v debug

# Use specific profile
skaffold dev -p development

# Watch specific files only
skaffold dev --trigger=manual
```

## Integration with Minikube

### Prerequisites

```bash
# Start minikube with required resources
minikube start --cpus=4 --memory=16384

# Point Docker to minikube's Docker daemon
eval $(minikube docker-env)

# Verify cluster connection
kubectl cluster-info
```

### Development Workflow

#### Using Makefile (Recommended)
```bash
# Start complete development environment
make dev

# Stop development mode
make dev-stop

# Start with debug output
make dev-debug

# Check minikube status
make minikube-status
```

#### Manual Steps
1. **Start minikube**: `minikube start --cpus=4 --memory=16384`
2. **Set Docker context**: `eval $(minikube docker-env)`
3. **Start Skaffold dev**: `skaffold dev -p development`
4. **Make code changes**: Files are watched and auto-deployed
5. **Test locally**: Use port-forward to access services

## Troubleshooting

### Common Issues

- **Build failures**: Check Dockerfile and build context
- **Deploy failures**: Verify Kubernetes manifests and cluster access
- **File sync issues**: Ensure sync configuration matches source structure
- **Port conflicts**: Use different ports or clean up existing deployments

### Debug Commands

```bash
# Verbose output
skaffold dev -v debug

# Check configuration
skaffold config list

# Validate skaffold.yaml
skaffold validate
```

This setup enables rapid iteration on grad service development while maintaining consistency with production deployment patterns.
