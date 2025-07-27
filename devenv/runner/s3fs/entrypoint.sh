#!/bin/bash
set -e

echo "=== S3FS Sidecar Container ==="
echo "Hostname: $(hostname)"
echo "Mount point: /workspace/dataset"

# Mount S3 datasets if S3 credentials are provided
if [ -n "$AWS_ACCESS_KEY_ID" ] && [ -n "$AWS_SECRET_ACCESS_KEY" ] && [ -n "$S3_BUCKET" ]; then
    echo "Mounting S3 bucket: $S3_BUCKET"
    
    # Create s3fs password file
    echo "$AWS_ACCESS_KEY_ID:$AWS_SECRET_ACCESS_KEY" > /etc/passwd-s3fs
    chmod 600 /etc/passwd-s3fs
    
    # Mount S3 bucket to /workspace/dataset
    s3fs "$S3_BUCKET" /workspace/dataset \
        -o passwd_file=/etc/passwd-s3fs \
        -o allow_other \
        -o ensure_diskfree=100 \
        -o parallel_count=10 \
        -o multireq_max=5 \
        -o url="https://s3.amazonaws.com" \
        -o endpoint="${S3_ENDPOINT:-us-east-1}" \
        || {
            echo "Error: Failed to mount S3 bucket"
            exit 1
        }
    
    echo "S3 bucket mounted successfully at /workspace/dataset"
    
    # Verify mount is working
    if mountpoint -q /workspace/dataset; then
        echo "Mount verification: SUCCESS"
        ls -la /workspace/dataset/ 2>/dev/null || echo "Dataset directory is empty or inaccessible"
    else
        echo "Mount verification: FAILED"
        exit 1
    fi
else
    echo "Warning: S3 credentials not provided, skipping S3 mount"
    echo "Required environment variables: AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, S3_BUCKET"
fi

echo "S3FS sidecar initialization complete"
echo "=================================="

# Execute the main command
exec "$@"
