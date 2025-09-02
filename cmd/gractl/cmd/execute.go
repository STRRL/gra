package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/strrl/gra/cmd/gractl/client"
	"github.com/strrl/gra/cmd/gractl/config"
	gradv1 "github.com/strrl/gra/gen/grad/v1"
)

// ExecuteCmd represents the top-level execute command
var ExecuteCmd = &cobra.Command{
	Use:   "execute [flags] -- COMMAND [args...]",
	Short: "Execute a command (auto-creates runner if needed)",
	Long: `Execute a command with automatic runner provisioning. 
If no runners are available, a new runner will be created automatically.

Use -- to separate gractl flags from the command to execute:
  gractl execute -- python script.py --verbose
  gractl execute --timeout 60 -- ls -la /workspace
  gractl execute --shell sh -- curl -s https://api.example.com`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Load configuration from file and environment
		globalConfig, err := config.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
			os.Exit(1)
		}
		
		// Get flags
		serverAddress, _ := cmd.Flags().GetString("server")
		shell, _ := cmd.Flags().GetString("shell")
		timeout, _ := cmd.Flags().GetInt32("timeout")
		workdir, _ := cmd.Flags().GetString("workdir")
		
		// Use server address from config if not provided via flag
		if serverAddress == "localhost:9090" && globalConfig.Server.Address != "" {
			serverAddress = globalConfig.Server.Address
		}
		
		// Handle double dash separation for command arguments
		var command string
		dashIndex := cmd.ArgsLenAtDash()
		if dashIndex >= 0 {
			// Double dash found, use everything after the dash as the command
			commandArgs := args[dashIndex:]
			if len(commandArgs) == 0 {
				fmt.Fprintf(os.Stderr, "Error: No command specified after --\n")
				os.Exit(1)
			}
			command = strings.Join(commandArgs, " ")
		} else {
			// No double dash, treat all args as the command (backward compatibility)
			command = strings.Join(args, " ")
		}

		// Initialize client
		cfg := &client.Config{
			ServerAddress: serverAddress,
		}
		
		grpcClient, err := client.NewClient(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to connect to server: %v\n", err)
			os.Exit(1)
		}
		defer grpcClient.Close()

		// Prepare environment variables map with AWS credentials from config
		envMap := make(map[string]string)
		if globalConfig.S3.AccessKeyID != "" {
			envMap["AWS_ACCESS_KEY_ID"] = globalConfig.S3.AccessKeyID
		}
		if globalConfig.S3.SecretAccessKey != "" {
			envMap["AWS_SECRET_ACCESS_KEY"] = globalConfig.S3.SecretAccessKey
		}
		if globalConfig.S3.SessionToken != "" {
			envMap["AWS_SESSION_TOKEN"] = globalConfig.S3.SessionToken
		}

		// Automatically inject SSH public key if available
		if sshPublicKey, err := client.GetUserSSHPublicKey(); err == nil && sshPublicKey != "" {
			envMap["PUBLIC_KEY"] = sshPublicKey
		}

		// Create request
		req := &gradv1.ExecuteCommandRequest{
			Command:    command,
			Shell:      shell,
			Timeout:    timeout,
			WorkingDir: workdir,
			Env:        envMap,
		}
		
		// Add workspace configuration if S3 bucket is specified in config
		if globalConfig.S3.Bucket != "" {
			req.Workspace = &gradv1.WorkspaceConfig{
				Bucket:   globalConfig.S3.Bucket,
				Endpoint: globalConfig.S3.Endpoint,
				Prefix:   globalConfig.S3.Prefix,
				Region:   globalConfig.S3.Region,
				ReadOnly: globalConfig.S3.ReadOnly,
			}
		}

		// Execute command with streaming
		stream, err := grpcClient.ExecuteService().ExecuteCommand(context.Background(), req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start command execution: %v\n", err)
			os.Exit(1)
		}

		var exitCode int32 = 0
		for {
			resp, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					break
				}
				fmt.Fprintf(os.Stderr, "Stream error: %v\n", err)
				os.Exit(1)
			}

			switch resp.Type {
			case gradv1.StreamType_STREAM_TYPE_STDOUT:
				os.Stdout.Write(resp.Data)
			case gradv1.StreamType_STREAM_TYPE_STDERR:
				os.Stderr.Write(resp.Data)
			case gradv1.StreamType_STREAM_TYPE_EXIT:
				exitCode = resp.ExitCode
			}
		}

		// Exit with the same code as the command
		if exitCode != 0 {
			os.Exit(int(exitCode))
		}
	},
}

func init() {
	// Command flags
	ExecuteCmd.Flags().StringP("server", "", "localhost:9090", "gRPC server address")
	ExecuteCmd.Flags().StringP("shell", "s", "bash", "Shell to use for command execution")
	ExecuteCmd.Flags().Int32P("timeout", "t", 30, "Command execution timeout in seconds")
	ExecuteCmd.Flags().StringP("workdir", "w", "", "Working directory for command execution")
}