# Runner container with Python, SSH, and S3FS capabilities
FROM python:3.11-slim

# Install system dependencies
RUN apt-get update && apt-get install -y \
    openssh-server \
    s3fs \
    fuse \
    curl \
    git \
    vim \
    htop \
    && rm -rf /var/lib/apt/lists/*

# Install Python packages
RUN pip install --no-cache-dir \
    pandas \
    numpy \
    duckdb \
    pyarrow \
    jupyter \
    matplotlib \
    seaborn \
    scipy \
    scikit-learn

# Create runner user
RUN useradd -m -s /bin/bash runner && \
    echo 'runner:runner' | chpasswd && \
    usermod -aG sudo runner

# Setup SSH
RUN mkdir /var/run/sshd && \
    sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin no/' /etc/ssh/sshd_config && \
    sed -i 's/#PasswordAuthentication yes/PasswordAuthentication yes/' /etc/ssh/sshd_config && \
    sed -i 's/#PubkeyAuthentication yes/PubkeyAuthentication yes/' /etc/ssh/sshd_config

# Create workspace structure
RUN mkdir -p /workspace/{code,outputs,data} && \
    chown -R runner:runner /workspace

# Setup SSH keys directory
RUN mkdir -p /home/runner/.ssh && \
    chown -R runner:runner /home/runner/.ssh && \
    chmod 700 /home/runner/.ssh

# Create s3fs cache directory
RUN mkdir -p /tmp/s3fs-cache && \
    chown -R runner:runner /tmp/s3fs-cache

# Copy entrypoint script
COPY devenv/runner/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Expose SSH port
EXPOSE 22

# Set working directory
WORKDIR /workspace

# Switch to runner user for final setup
USER runner

# Set environment variables
ENV PYTHONPATH=/workspace/code
ENV PATH=/home/runner/.local/bin:$PATH

# Switch back to root for entrypoint (needed for SSH and S3FS)
USER root

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["/usr/sbin/sshd", "-D"]