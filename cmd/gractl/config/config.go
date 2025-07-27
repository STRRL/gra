package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the complete configuration for gractl
type Config struct {
	// S3 configuration for default workspace
	S3 S3Config `mapstructure:"s3"`
	
	// Server configuration
	Server ServerConfig `mapstructure:"server"`
}

// S3Config holds S3 workspace configuration
type S3Config struct {
	Bucket          string `mapstructure:"bucket"`
	Endpoint        string `mapstructure:"endpoint"`
	Prefix          string `mapstructure:"prefix"`
	Region          string `mapstructure:"region"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	SessionToken    string `mapstructure:"session_token"`
	ReadOnly        bool   `mapstructure:"read_only"`
}

// ServerConfig holds server connection configuration
type ServerConfig struct {
	Address string `mapstructure:"address"`
}

// LoadConfig loads configuration from .gractl.toml file and environment variables
func LoadConfig() (*Config, error) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Initialize viper
	v := viper.New()
	
	// Set config file name and type
	v.SetConfigName(".gractl")
	v.SetConfigType("toml")
	
	// Add search paths
	v.AddConfigPath(cwd)              // Current working directory
	v.AddConfigPath(".")              // Fallback to current directory
	v.AddConfigPath(getHomeDir())     // User home directory

	// Set environment variable prefix
	v.SetEnvPrefix("GRACTL")
	v.AutomaticEnv()

	// Set default values
	setDefaults(v)

	// Read config file (optional)
	if err := v.ReadInConfig(); err != nil {
		// It's okay if config file doesn't exist, we'll use defaults and env vars
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Unmarshal into config struct
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.address", "localhost:9090")
	
	// S3 defaults
	v.SetDefault("s3.region", "us-east-1")
	v.SetDefault("s3.read_only", false)
}

// getHomeDir returns the user's home directory
func getHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

// GetConfigPath returns the path where the config file should be located
func GetConfigPath() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ".gractl.toml"
	}
	return filepath.Join(cwd, ".gractl.toml")
}