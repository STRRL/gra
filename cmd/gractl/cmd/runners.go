package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	gradv1 "github.com/strrl/gra/gen/grad/v1"
	"github.com/strrl/gra/cmd/gractl/client"
	"github.com/strrl/gra/cmd/gractl/config"
)

var (
	serverAddress string
	outputFormatStr string
	grpcClient    *client.Client
	globalConfig  *config.Config
)

// RunnersCmd represents the runners command
var RunnersCmd = &cobra.Command{
	Use:   "runners",
	Short: "Manage runners",
	Long:  `Manage runner instances including creating, listing, and executing commands.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Load configuration from file and environment
		var err error
		globalConfig, err = config.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
			os.Exit(1)
		}

		// Use server address from config if not provided via flag
		if serverAddress == "localhost:9090" && globalConfig.Server.Address != "" {
			serverAddress = globalConfig.Server.Address
		}

		// Set output format
		switch outputFormatStr {
		case "json":
			outputFormat = OutputFormatJSON
		case "table":
			outputFormat = OutputFormatTable
		default:
			fmt.Fprintf(os.Stderr, "Invalid output format: %s (supported: table, json)\n", outputFormatStr)
			os.Exit(1)
		}

		// Initialize client for all subcommands
		cfg := &client.Config{
			ServerAddress: serverAddress,
		}
		
		grpcClient, err = client.NewClient(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to connect to server: %v\n", err)
			os.Exit(1)
		}
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		// Clean up client connection
		if grpcClient != nil {
			grpcClient.Close()
		}
	},
}

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new runner",
	Long:  `Create a new runner instance with optional name and environment variables.`,
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		envVars, _ := cmd.Flags().GetStringSlice("env")
		
		// S3 workspace configuration flags
		s3Bucket, _ := cmd.Flags().GetString("s3-bucket")
		s3Endpoint, _ := cmd.Flags().GetString("s3-endpoint")
		s3Prefix, _ := cmd.Flags().GetString("s3-prefix")
		s3Region, _ := cmd.Flags().GetString("s3-region")
		readOnly, _ := cmd.Flags().GetBool("read-only")

		// Use config values as defaults if flags are not provided
		if s3Bucket == "" && globalConfig.S3.Bucket != "" {
			s3Bucket = globalConfig.S3.Bucket
		}
		// Always use config values for other S3 settings if not explicitly provided via flags
		// This allows mixing --s3-bucket flag with config file settings
		if s3Endpoint == "" && globalConfig.S3.Endpoint != "" {
			s3Endpoint = globalConfig.S3.Endpoint
		}
		if s3Prefix == "" && globalConfig.S3.Prefix != "" {
			s3Prefix = globalConfig.S3.Prefix
		}
		if s3Region == "" && globalConfig.S3.Region != "" {
			s3Region = globalConfig.S3.Region
		}
		if !cmd.Flags().Changed("read-only") && globalConfig.S3.ReadOnly {
			readOnly = globalConfig.S3.ReadOnly
		}

		// Parse environment variables
		envMap := make(map[string]string)
		for _, env := range envVars {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				envMap[parts[0]] = parts[1]
			}
		}

		// Always auto-inject AWS credentials from config if available (regardless of bucket source)
		// This allows using --s3-bucket flag while still getting credentials from config
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

		req := &gradv1.CreateRunnerRequest{
			Name: name,
			Env:  envMap,
		}
		
		// Add workspace configuration if S3 bucket is specified (either via flag or config)
		if s3Bucket != "" {
			req.Workspace = &gradv1.WorkspaceConfig{
				Bucket:    s3Bucket,
				Endpoint:  s3Endpoint,
				Prefix:    s3Prefix,
				Region:    s3Region,
				ReadOnly:  readOnly,
			}
		}

		resp, err := grpcClient.RunnerService().CreateRunner(context.Background(), req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create runner: %v\n", err)
			os.Exit(1)
		}

		if err := PrintRunner(resp.Runner); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to print runner: %v\n", err)
			os.Exit(1)
		}
	},
}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List runners",
	Long:  `List all runners with optional filtering by status.`,
	Aliases: []string{"ls"},
	Run: func(cmd *cobra.Command, args []string) {
		statusStr, _ := cmd.Flags().GetString("status")
		limit, _ := cmd.Flags().GetInt32("limit")
		offset, _ := cmd.Flags().GetInt32("offset")

		status, err := ParseRunnerStatus(statusStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid status: %v\n", err)
			os.Exit(1)
		}

		req := &gradv1.ListRunnersRequest{
			Status: status,
			Limit:  limit,
			Offset: offset,
		}

		resp, err := grpcClient.RunnerService().ListRunners(context.Background(), req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list runners: %v\n", err)
			os.Exit(1)
		}

		if err := PrintRunnerList(resp.Runners); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to print runners: %v\n", err)
			os.Exit(1)
		}
	},
}

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get RUNNER_ID",
	Short: "Get runner details",
	Long:  `Get detailed information about a specific runner.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runnerID := args[0]

		req := &gradv1.GetRunnerRequest{
			RunnerId: runnerID,
		}

		resp, err := grpcClient.RunnerService().GetRunner(context.Background(), req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get runner: %v\n", err)
			os.Exit(1)
		}

		if err := PrintRunner(resp.Runner); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to print runner: %v\n", err)
			os.Exit(1)
		}
	},
}

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete [RUNNER_ID]",
	Short: "Delete a runner or all runners",
	Long:  `Delete a runner instance by ID, or delete all runners with --all flag.`,
	Aliases: []string{"rm"},
	Args: func(cmd *cobra.Command, args []string) error {
		all, _ := cmd.Flags().GetBool("all")
		if all && len(args) > 0 {
			return fmt.Errorf("cannot specify runner ID when using --all flag")
		}
		if !all && len(args) != 1 {
			return fmt.Errorf("requires exactly one RUNNER_ID when not using --all flag")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		all, _ := cmd.Flags().GetBool("all")
		
		if all {
			// Delete all runners
			// First, list all runners
			listReq := &gradv1.ListRunnersRequest{
				Status: gradv1.RunnerStatus_RUNNER_STATUS_UNSPECIFIED, // Get all runners regardless of status
				Limit:  0, // No limit
				Offset: 0,
			}

			listResp, err := grpcClient.RunnerService().ListRunners(context.Background(), listReq)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to list runners: %v\n", err)
				os.Exit(1)
			}

			if len(listResp.Runners) == 0 {
				fmt.Printf("No runners found to delete\n")
				return
			}

			// Delete each runner
			successCount := 0
			for _, runner := range listResp.Runners {
				deleteReq := &gradv1.DeleteRunnerRequest{
					RunnerId: runner.Id,
				}

				_, err := grpcClient.RunnerService().DeleteRunner(context.Background(), deleteReq)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to delete runner %s: %v\n", runner.Id, err)
				} else {
					fmt.Printf("Deleted runner: %s\n", runner.Id)
					successCount++
				}
			}

			fmt.Printf("Successfully deleted %d out of %d runners\n", successCount, len(listResp.Runners))
		} else {
			// Delete single runner
			runnerID := args[0]

			req := &gradv1.DeleteRunnerRequest{
				RunnerId: runnerID,
			}

			resp, err := grpcClient.RunnerService().DeleteRunner(context.Background(), req)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to delete runner: %v\n", err)
				os.Exit(1)
			}

			if err := PrintMessage(resp.Message); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to print message: %v\n", err)
				os.Exit(1)
			}
		}
	},
}

// execCmd represents the exec command
var execCmd = &cobra.Command{
	Use:   "exec RUNNER_ID COMMAND [args...]",
	Short: "Execute a command in a runner",
	Long:  `Execute a command in a specific runner instance with streaming output.`,
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		runnerID := args[0]
		command := strings.Join(args[1:], " ")

		shell, _ := cmd.Flags().GetString("shell")
		timeout, _ := cmd.Flags().GetInt32("timeout")
		workdir, _ := cmd.Flags().GetString("workdir")

		req := &gradv1.ExecuteCommandRequest{
			RunnerId:   runnerID,
			Command:    command,
			Shell:      shell,
			Timeout:    timeout,
			WorkingDir: workdir,
		}

		// Use streaming execution (only option available)
		stream, err := grpcClient.RunnerService().ExecuteCommandStream(context.Background(), req)
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
			case gradv1.StreamType_STREAM_TYPE_STDOUT, gradv1.StreamType_STREAM_TYPE_STDERR:
				if err := PrintStreamData(resp.Type, resp.Data); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to print stream data: %v\n", err)
					os.Exit(1)
				}
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
	// Global flags
	RunnersCmd.PersistentFlags().StringVar(&serverAddress, "server", "localhost:9090", "gRPC server address")
	RunnersCmd.PersistentFlags().StringVarP(&outputFormatStr, "output", "o", "table", "Output format (table, json)")

	// Create command flags
	createCmd.Flags().StringP("name", "n", "", "Runner name (optional)")
	createCmd.Flags().StringSliceP("env", "e", []string{}, "Environment variables (KEY=VALUE)")
	
	// S3 workspace configuration flags
	createCmd.Flags().String("s3-bucket", "", "S3 bucket name for workspace")
	createCmd.Flags().String("s3-endpoint", "", "S3 endpoint URL (optional, defaults to AWS S3)")
	createCmd.Flags().String("s3-prefix", "", "S3 path prefix within the bucket (optional)")
	createCmd.Flags().String("s3-region", "", "AWS region (optional, defaults to us-east-1)")
	createCmd.Flags().Bool("read-only", false, "Mount S3 bucket as read-only")

	// List command flags
	listCmd.Flags().StringP("status", "s", "", "Filter by status (creating, running, stopping, stopped, error)")
	listCmd.Flags().Int32P("limit", "l", 0, "Limit number of results")
	listCmd.Flags().Int32("offset", 0, "Offset for pagination")

	// Delete command flags
	deleteCmd.Flags().Bool("all", false, "Delete all runners")

	// Exec command flags
	execCmd.Flags().StringP("shell", "s", "bash", "Shell to use for command execution")
	execCmd.Flags().Int32P("timeout", "t", 30, "Command execution timeout in seconds")
	execCmd.Flags().StringP("workdir", "w", "", "Working directory for command execution")

	// Add subcommands
	RunnersCmd.AddCommand(createCmd)
	RunnersCmd.AddCommand(listCmd)
	RunnersCmd.AddCommand(getCmd)
	RunnersCmd.AddCommand(deleteCmd)
	RunnersCmd.AddCommand(execCmd)
}