package service

import (
	"os"
	"strconv"
)

// Config holds the configuration for the grad service
type Config struct {
	Kubernetes *KubernetesConfig
}

// LoadConfig loads configuration from environment variables and defaults
func LoadConfig() *Config {
	return &Config{
		Kubernetes: loadKubernetesConfig(),
	}
}

// loadKubernetesConfig loads Kubernetes configuration from environment variables
func loadKubernetesConfig() *KubernetesConfig {
	config := DefaultKubernetesConfig()

	// Override with environment variables if provided
	if namespace := os.Getenv("KUBERNETES_NAMESPACE"); namespace != "" {
		config.Namespace = namespace
	}

	// Override runner image if provided (handles skaffold dynamic tags)
	if runnerImage := os.Getenv("RUNNER_IMAGE"); runnerImage != "" {
		config.RunnerImage = runnerImage
	}

	if sshPortStr := os.Getenv("SSH_PORT"); sshPortStr != "" {
		if port, err := strconv.ParseInt(sshPortStr, 10, 32); err == nil {
			config.SSHPort = int32(port)
		}
	}

	return config
}
