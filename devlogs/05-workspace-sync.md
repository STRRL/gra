# Development Log 05: Workspace Sync Feature

## Overview

This document describes the implementation of the `gractl workspace sync` feature, which enables seamless file synchronization between local development environments and remote runner workspaces using sshfs over kubectl port-forward.

## Feature Requirements

### User Story
As a developer using gra runners for data analytics, I want to mount the remote runner's `/workspace` directory to my local machine so I can:
- Edit files locally using my preferred IDE
- Have changes immediately available in the remote runner
- Maintain a persistent local copy of my work
- Seamlessly switch between local editing and remote execution

### Technical Requirements

1. **Pure Client-Side Implementation**: No server-side changes required beyond existing SSH infrastructure
2. **SSH Key Integration**: Automatically inject user's SSH public key into runners during creation
3. **kubectl Integration**: Use kubectl port-forward to create secure SSH tunnel
4. **Local Directory Management**: Create clean directory structure under `./runners/<runner-id>/workspace`
5. **Graceful Cleanup**: Handle interrupts and properly unmount/cleanup resources
6. **Dependency Checking**: Verify kubectl and sshfs are available

## Architecture

### Component Interaction

The workspace sync feature operates entirely on the client side, leveraging existing infrastructure:

1. **Runner Creation**: Inject user's SSH public key via PUBLIC_KEY environment variable
2. **Port Forwarding**: Use kubectl port-forward to tunnel SSH traffic (pod:22 → localhost:random_port)
3. **sshfs Mounting**: Mount remote `/workspace` to local `./runners/<runner-id>/workspace`
4. **Cleanup**: Gracefully unmount and terminate port-forward on exit

### Dependencies

**Client-side (gractl)**:
- kubectl (for port-forward)
- sshfs (for filesystem mounting)
- SSH key pair (typically ~/.ssh/id_rsa or ~/.ssh/id_ed25519)

**Server-side (runner containers)**:
- SSH daemon (already configured)
- PUBLIC_KEY environment variable handling (already implemented)

## Implementation Plan

### Phase 1: SSH Key Integration
- Modify `internal/grad/service/pod_spec.go` to read and inject user's SSH public key
- Update runner creation to include PUBLIC_KEY environment variable
- Ensure backward compatibility for runners without SSH keys

### Phase 2: Core workspace sync Command
- Create `cmd/gractl/cmd/workspace_sync.go`
- Implement runner status checking
- Add local directory management
- Implement kubectl port-forward subprocess management
- Add sshfs mounting logic
- Handle graceful cleanup and error scenarios

### Phase 3: Client Infrastructure
- Update `cmd/gractl/client/client.go` with SSH key utilities
- Add helper functions for directory management
- Implement dependency checking (kubectl, sshfs availability)

### Phase 4: Integration and Testing
- Integrate SSH key injection with runner creation command
- Test various scenarios (with/without SSH keys, permission issues, etc.)
- Test cleanup on interrupts and errors

## Technical Details

### SSH Key Location Strategy
1. Try `~/.ssh/id_ed25519.pub` (modern default)
2. Fall back to `~/.ssh/id_rsa.pub` (traditional default)
3. If neither exists, create runner without SSH key (backward compatible)

### Local Directory Structure
```
./runners/
├── runner-1/
│   └── workspace/          # sshfs mount point
├── runner-2/
│   └── workspace/
└── ...
```

### Port-Forward Management
- Use random local port to avoid conflicts
- Start kubectl port-forward as subprocess
- Monitor subprocess health
- Clean termination on exit

### sshfs Options
```bash
sshfs runner@localhost:/workspace ./runners/<runner-id>/workspace \
  -p <local_port> \
  -o reconnect \
  -o UserKnownHostsFile=/dev/null \
  -o StrictHostKeyChecking=no \
  -o PasswordAuthentication=no
```

### Error Handling
- Check kubectl availability and cluster connectivity
- Verify sshfs is installed
- Validate runner exists and is running
- Handle mount point conflicts
- Graceful cleanup on all error paths

## Security Considerations

### SSH Key Handling
- Only read public keys (never private keys)
- Use existing user SSH keys (no key generation)
- Keys are injected as environment variables (standard practice)

### Network Security
- All traffic goes through kubectl port-forward (authenticated)
- SSH provides additional encryption layer
- No direct network exposure of runner SSH

### Local Filesystem
- Mount points are under user's current directory
- Standard UNIX permissions apply
- sshfs runs with user privileges (no root required)

## Future Enhancements

### Multi-Runner Sync
- Support mounting multiple runners simultaneously
- Manage port conflicts automatically

### IDE Integration
- VS Code extension for automatic workspace mounting
- Integration with development workflows

### Advanced Mount Options
- Read-only mounting option
- Selective directory mounting (not just /workspace)
- Custom mount point locations

## Testing Strategy

### Unit Tests
- SSH key reading functions
- Directory management utilities
- Error handling paths

### Integration Tests
- End-to-end workspace sync flow
- Runner creation with SSH keys
- Cleanup and error scenarios

### Manual Testing
- Different SSH key configurations
- Various operating systems
- Permission and dependency scenarios

## Implementation Notes

### Backward Compatibility
- Runners without SSH keys continue to work normally
- Existing runner creation API unchanged
- New functionality is opt-in

### Cross-Platform Considerations
- SSH key locations may vary by OS
- sshfs availability on different platforms
- kubectl behavior consistency

### Performance
- sshfs performance is adequate for development workflows
- kubectl port-forward adds minimal overhead
- Cleanup is fast and reliable

## Conclusion

The workspace sync feature provides a seamless bridge between local development and remote execution environments. By leveraging existing SSH infrastructure and kubectl port-forwarding, we can provide a robust, secure, and user-friendly file synchronization solution without requiring server-side changes.

The implementation maintains backward compatibility while adding powerful new capabilities for developer workflows, enabling the full potential of remote execution with local development comfort.