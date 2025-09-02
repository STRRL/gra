# gractl Documentation

## Overview

`gractl` is a command-line tool for managing grad runners and executing remote commands. It provides a simple interface to create runners, execute commands, and sync workspaces.

## Quick Start

```bash
# Execute a command (auto-creates runner if needed)
gractl execute "echo hello world"

# Create a new runner
gractl runners create --name my-runner

# Execute command in specific runner
gractl runners exec runner-123 "ls -la"

# List all runners
gractl runners list

# Sync runner workspace locally
gractl workspace sync runner-123
```

## Main Commands

### `gractl execute`

Executes a command with automatic runner provisioning. If no runners are available, a new one is created automatically.

```bash
gractl execute "python script.py" --workdir /workspace --timeout 60
```

### `gractl runners`

Manage runner instances - create, list, delete, and execute commands in specific runners.

```bash
# Create runner with S3 workspace
gractl runners create --name data-processor --s3-bucket my-bucket --s3-prefix projects/data

# List runners in JSON format
gractl runners list --output json

# Delete a runner
gractl runners delete runner-123
```

### `gractl workspace sync`

Mount runner workspaces locally using sshfs for easy file access and editing.

```bash
# Sync specific runner
gractl workspace sync runner-123

# Sync all running runners
gractl workspace sync
```

## Common Options

- `--server`: gRPC server address (default: localhost:9090)
- `--output`: Output format - table or json (default: table)
- `--timeout`: Command execution timeout in seconds
- `--workdir`: Working directory for command execution
