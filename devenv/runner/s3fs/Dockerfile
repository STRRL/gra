# S3FS sidecar container for mounting S3 datasets
FROM debian:bookworm-slim

# Install s3fs and required dependencies
RUN apt-get update && apt-get install -y \
    s3fs \
    fuse \
    curl \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Create mount point for S3 dataset
RUN mkdir -p /workspace/dataset

# Create s3fs cache directory
RUN mkdir -p /tmp/s3fs-cache

# Copy entrypoint script
COPY devenv/runner/s3fs/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Set working directory
WORKDIR /workspace

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["sleep", "infinity"]