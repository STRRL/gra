# System Architecture

## Core Concepts

### User Workspace
- Local directory on user's machine under the working directory
- Contains workspace configuration files
- Stores dataset connection descriptions
- Mount point for remote runner workspaces via sshfs

### Runner
- Remote compute instance controlled by grad
- Mounts datasets via s3fs
- Runs sshd for remote access
- Provides isolated execution environment

### Workspace Sync
- Frontend process initiated by `gractl workspace sync`
- Uses sshfs to mount runner workspace
- Mounted under `workspace/runners/<runner-name>/`
- Enables seamless file access between local and remote

## Component Overview

### Local Components

- **Gemini CLI**: Natural language interface for users
- **gractl**: CLI interface to grad service (human and AI friendly)
- **User Workspace**: Local workspace directory with dataset configurations

### Remote Components  

- **grad**: gRPC service managing runner lifecycle (deployed in Kubernetes)
- **Kubernetes**: Container orchestration platform (minikube with 4C16G)
- **Runners**: Remote compute instances with sshd and s3fs (Kubernetes pods)
- **S3 Storage**: Data repository for parquet files

## Data Flow

**User Request Flow:**

1. User issues natural language query to Gemini CLI
2. Gemini CLI calls `gractl execute-code` with code and parameters
3. gractl sends gRPC request to grad service
4. grad creates/reuses runner (Kubernetes pod) for execution
5. Runner executes code with S3 data access via s3fs
6. Results flow back: grad → gractl → Gemini CLI → user

**Workspace Sync Flow:**

1. User executes `gractl workspace sync`
2. gractl identifies available runners from grad
3. sshfs mounts runner workspaces to local `workspace/runners/<runner-name>/`
4. User can access remote files as if they were local
5. Changes are synchronized in real-time via sshfs

## Component Responsibilities

### gractl (CLI Interface)

- **Human-friendly**: Clear commands and readable output
- **AI-friendly**: Structured input/output for tool integration
- **Functions**: Execute code, manage workspace, sync runners
- **Communication**: gRPC client to grad service
- **Workspace Management**: Initialize and configure local workspace
- **Sync Operations**: Mount remote runner workspaces via sshfs

### grad (gRPC Service)

- **Runner Lifecycle**: Create, pause, resume, destroy runners
- **Resource Management**: CPU, memory, storage allocation
- **Session Tracking**: User context and state persistence
- **Kubernetes Integration**: Pod management via K8s API
- **Runner Registry**: Track available runners and their SSH endpoints

### Runners (Remote Compute Instances)

- **Runtime Environment**: Python with pandas, DuckDB
- **Data Access**: S3 mounting via s3fs for dataset access
- **SSH Server**: sshd for remote filesystem access
- **Workspace**: Dedicated directory for user files and outputs
- **Isolation**: Separate execution environments per user
- **Result Collection**: Output capture and synchronization

## Workspace Structure

### Local User Workspace
```
workspace/
├── config.yaml              # Workspace configuration
├── datasets/                # Dataset connection descriptions
│   ├── dataset1.yaml
│   └── dataset2.yaml
└── runners/                 # Mounted runner workspaces
    ├── runner-1/           # sshfs mount point
    │   ├── code/
    │   └── outputs/
    └── runner-2/
        ├── code/
        └── outputs/
```

### Remote Runner Workspace
```
/workspace/                  # Runner workspace root
├── code/                    # User code files
├── outputs/                 # Execution outputs
└── data/                    # s3fs mount points
    ├── dataset1/           # Mounted from S3
    └── dataset2/
```

## Development Environment

### Minikube Configuration
- **Resources**: 4 CPUs, 16GB RAM
- **Purpose**: Local Kubernetes cluster for development and testing
- **Components**: Hosts both grad service and runner pods

### Setup Requirements
```bash
# Quick start with Makefile (recommended)
make dev

# Or manual setup
minikube start --cpus=4 --memory=16384
eval $(minikube docker-env)
skaffold dev -p development --port-forward
```

### Deployment Architecture
- **grad service**: Deployed via Skaffold using Kubernetes deployment manifests
- **Runners**: Dynamically created by grad as individual Kubernetes pods (not managed by Skaffold)
- **Development Workflow**: Skaffold provides continuous development with file watching for grad service
- **Networking**: Internal cluster networking for grad-runner communication
- **External Access**: grad service exposed via NodePort with Skaffold port-forwarding

### Development Tools
- **Skaffold**: Continuous development workflow with automatic build/deploy for grad service
- **File Sync**: Hot-reload Go code changes without full rebuild
- **Port Forwarding**: Direct access to grad service during development
- **Profiles**: Different configurations for development, debug, and production

### Runner Management
- **Dynamic Creation**: grad service creates runner pods on-demand via Kubernetes API
- **Pre-built Images**: Runner container images are pre-built and available in registry
- **Lifecycle Management**: grad manages runner creation, execution, and cleanup
- **Resource Allocation**: Each runner gets dedicated compute resources and workspace

## Key Design Principles

1. **Clean Separation**: Gemini CLI never directly contacts grad
2. **Dual Interface**: gractl serves both human users and AI tools
3. **Unified Workspace**: Local and remote workspaces synchronized via sshfs
4. **Data Locality**: Datasets mounted directly in runners via s3fs
5. **Transparent Access**: Users work with remote resources as if local
6. **Stateless Design**: grad service handles runner state management
7. **Resource Efficiency**: Runner reuse and automatic cleanup
8. **Extensibility**: Pluggable execution environments and data sources
