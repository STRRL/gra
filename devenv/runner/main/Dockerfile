# Main runner container with Python, SSH, and DuckDB CLI
FROM python:3.11-slim

# Install system dependencies
RUN apt-get update && apt-get install -y \
    openssh-server \
    curl \
    git \
    vim \
    htop \
    wget \
    unzip \
    && rm -rf /var/lib/apt/lists/*

# Install DuckDB CLI binary
RUN curl -L https://github.com/duckdb/duckdb/releases/latest/download/duckdb_cli-linux-amd64.zip -o /tmp/duckdb.zip && \
    unzip /tmp/duckdb.zip -d /tmp && \
    mv /tmp/duckdb /usr/local/bin/duckdb && \
    chmod +x /usr/local/bin/duckdb && \
    rm -f /tmp/duckdb.zip

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
RUN mkdir -p /workspace/{code,outputs,dataset} && \
    chown -R runner:runner /workspace

# Setup SSH keys directory
RUN mkdir -p /home/runner/.ssh && \
    chown -R runner:runner /home/runner/.ssh && \
    chmod 700 /home/runner/.ssh

# Copy entrypoint script
COPY devenv/runner/main/entrypoint.sh /usr/local/bin/entrypoint.sh
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

# Switch back to root for entrypoint (needed for SSH)
USER root

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["sleep", "infinity"]