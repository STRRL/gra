# Implementation Plan

## POC Scope and Goals

**Primary Goal**: Demonstrate basic container execution via Gemini CLI using gractl
**Success Criteria**: Execute simple Python code in a Kubernetes pod via `gractl execute-code`

## Architecture Flow

Gemini CLI → `gractl execute-code` → grad (gRPC) → Kubernetes API → Pod

## Core Infrastructure Setup

### Development Environment

- [ ] Set up local Kubernetes cluster (minikube with 4C16G configuration)
- [ ] Configure Skaffold for rapid grad development workflow
- [ ] Use Skaffold to deploy and iterate on grad service in minikube

### Runner Container

- [ ] Create basic Python runner container image with pandas, sshd, and s3fs
- [ ] Configure runner container with SSH access for workspace synchronization
- [ ] Set up dedicated workspace directory structure in runner container

### gRPC Service Implementation

- [ ] Implement minimal gRPC server (grad) for runner management
- [ ] Basic Kubernetes runner pod creation/deletion via Go client
- [ ] Define execute-code gRPC service with buf
- [ ] Implement execute-code method in grad server

### CLI Development

- [ ] Enhance gractl to be both human and AI friendly
- [ ] Add execute-code subcommand to gractl for both humans and AI
- [ ] Implement gractl execute-code subcommand integration
- [ ] Simple result retrieval mechanism
- [ ] Test with hardcoded Python script

## Integration Tasks

### End-to-End Integration

- [ ] Modify existing Gemini CLI to call `gractl execute-code`
- [ ] Basic S3 file access (if time permits, otherwise use local files)
- [ ] End-to-end testing
- [ ] Basic error handling
- [ ] Manual testing validation

## Minimal Feature Set

**Included:**

- Local Kubernetes cluster (minikube 4C16G)
- Python runner containers with sshd and s3fs
- Simple runner lifecycle management
- gRPC API (grad) deployed in minikube with buf-generated clients
- Enhanced gractl with execute-code and workspace sync subcommands
- Gemini CLI integration via gractl

**Excluded (Future Work):**

- S3 integration (use local files)
- Advanced container lifecycle management
- Spark/distributed computing
- Monitoring and observability
- Security features
- Production deployment
- Auto-scaling
- Multiple concurrent users

## Technical Approach

### Runner Strategy

- Pre-built Python runner container with pandas, sshd, s3fs
- SSH access enabled for workspace synchronization
- Dedicated workspace directory structure
- Simple exec and SSH-based access patterns

### Storage Strategy

- Local files only for POC
- Skip S3 integration for time constraints

### gractl Integration

- Add execute-code subcommand to existing gractl
- Support both human-readable and AI-friendly output formats
- Handle code input via parameters or files

## Success Metrics

### Core Infrastructure Complete

- [ ] Can create/destroy runner pods via gRPC API deployed in minikube
- [ ] Basic code execution working with protobuf messages
- [ ] gractl execute-code subcommand functional
- [ ] SSH access to runners established

### Full Integration Complete

- [ ] Gemini CLI successfully executes Python code via gractl
- [ ] Results returned and displayed
- [ ] Demo-ready POC

This plan focuses on proving the core concept with gractl as the unified interface.
