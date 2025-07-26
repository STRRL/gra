package client

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GetUserSSHPublicKey reads the user's SSH public key from standard locations
// Returns the public key content or empty string if no key is found
func GetUserSSHPublicKey() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Try common SSH public key locations in order of preference
	keyPaths := []string{
		filepath.Join(homeDir, ".ssh", "id_ed25519.pub"),
		filepath.Join(homeDir, ".ssh", "id_rsa.pub"),
		filepath.Join(homeDir, ".ssh", "id_ecdsa.pub"),
	}

	for _, keyPath := range keyPaths {
		if content, err := readSSHPublicKey(keyPath); err == nil {
			return content, nil
		}
	}

	// No SSH public key found - this is not an error, just return empty string
	return "", nil
}

// readSSHPublicKey reads and validates an SSH public key file
func readSSHPublicKey(keyPath string) (string, error) {
	// Check if file exists
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return "", fmt.Errorf("key file does not exist: %s", keyPath)
	}

	// Read the file
	content, err := os.ReadFile(keyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read SSH key file %s: %w", keyPath, err)
	}

	// Basic validation - SSH public keys should start with known prefixes
	keyContent := strings.TrimSpace(string(content))
	if keyContent == "" {
		return "", fmt.Errorf("SSH key file is empty: %s", keyPath)
	}

	// Validate that it looks like an SSH public key
	validPrefixes := []string{
		"ssh-rsa ",
		"ssh-ed25519 ",
		"ssh-ecdsa ",
		"ecdsa-sha2-nistp256 ",
		"ecdsa-sha2-nistp384 ",
		"ecdsa-sha2-nistp521 ",
	}

	isValid := false
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(keyContent, prefix) {
			isValid = true
			break
		}
	}

	if !isValid {
		return "", fmt.Errorf("file does not appear to contain a valid SSH public key: %s", keyPath)
	}

	return keyContent, nil
}

// CreateLocalDirectory creates a directory if it doesn't exist
func CreateLocalDirectory(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	return nil
}

// GetRunnerWorkspaceDir returns the local workspace directory path for a runner
func GetRunnerWorkspaceDir(runnerID string) string {
	return filepath.Join("runners", runnerID, "workspace")
}

// CheckCommandAvailable checks if a command is available in PATH
func CheckCommandAvailable(command string) error {
	_, err := os.Stat("/usr/bin/" + command)
	if err == nil {
		return nil
	}
	
	_, err = os.Stat("/usr/local/bin/" + command)
	if err == nil {
		return nil
	}

	// Check if command exists in PATH
	if _, err := exec.LookPath(command); err != nil {
		return fmt.Errorf("command '%s' not found in PATH", command)
	}
	
	return nil
}