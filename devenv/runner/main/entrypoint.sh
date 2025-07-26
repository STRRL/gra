#!/bin/bash
set -e

# Initialize SSH host keys if they don't exist
if [ ! -f /etc/ssh/ssh_host_rsa_key ]; then
    ssh-keygen -t rsa -f /etc/ssh/ssh_host_rsa_key -N ''
fi

if [ ! -f /etc/ssh/ssh_host_ecdsa_key ]; then
    ssh-keygen -t ecdsa -f /etc/ssh/ssh_host_ecdsa_key -N ''
fi

if [ ! -f /etc/ssh/ssh_host_ed25519_key ]; then
    ssh-keygen -t ed25519 -f /etc/ssh/ssh_host_ed25519_key -N ''
fi

# Setup SSH authorized keys if PUBLIC_KEY is provided
if [ -n "$PUBLIC_KEY" ]; then
    # Setup for runner user
    echo "$PUBLIC_KEY" > /home/runner/.ssh/authorized_keys
    chown runner:runner /home/runner/.ssh/authorized_keys
    chmod 600 /home/runner/.ssh/authorized_keys
    
    # Setup for root user as well (needed for sshfs workspace sync)
    mkdir -p /root/.ssh
    echo "$PUBLIC_KEY" > /root/.ssh/authorized_keys
    chmod 700 /root/.ssh
    chmod 600 /root/.ssh/authorized_keys
fi

# Set proper ownership for workspace
chown -R runner:runner /workspace

# Start SSH daemon in background
/usr/sbin/sshd -D &

# Print runner information
echo "=== Main Runner Environment ==="
echo "Hostname: $(hostname)"
echo "User: runner"
echo "Workspace: /workspace"
echo "Python: $(python3 --version)"
echo "DuckDB: $(duckdb --version 2>/dev/null || echo 'Not available')"
echo "Available Python packages:"
pip list | head -20
echo "================================"

# Execute the main command
exec "$@"