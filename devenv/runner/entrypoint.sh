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
    echo "$PUBLIC_KEY" > /home/runner/.ssh/authorized_keys
    chown runner:runner /home/runner/.ssh/authorized_keys
    chmod 600 /home/runner/.ssh/authorized_keys
fi

# Mount S3 datasets if S3 credentials are provided
if [ -n "$AWS_ACCESS_KEY_ID" ] && [ -n "$AWS_SECRET_ACCESS_KEY" ] && [ -n "$S3_BUCKET" ]; then
    echo "Mounting S3 bucket: $S3_BUCKET"
    
    # Create s3fs password file
    echo "$AWS_ACCESS_KEY_ID:$AWS_SECRET_ACCESS_KEY" > /etc/passwd-s3fs
    chmod 600 /etc/passwd-s3fs
    
    # Mount S3 bucket to /workspace/data
    s3fs "$S3_BUCKET" /workspace/data \
        -o passwd_file=/etc/passwd-s3fs \
        -o allow_other \
        -o use_cache=/tmp/s3fs-cache \
        -o ensure_diskfree=100 \
        -o parallel_count=10 \
        -o multireq_max=5 \
        -o url="https://s3.amazonaws.com" \
        -o endpoint="${S3_ENDPOINT:-us-east-1}" \
        || echo "Warning: Failed to mount S3 bucket, continuing without S3 mount"
fi

# Set proper ownership for workspace
chown -R runner:runner /workspace

# Print runner information
echo "=== Runner Environment ==="
echo "Hostname: $(hostname)"
echo "User: runner"
echo "Workspace: /workspace"
echo "Python: $(python3 --version)"
echo "Available packages:"
pip list | head -20
echo "=========================="

# Execute the main command
exec "$@"