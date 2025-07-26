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
)

var (
	serverAddress string
	outputFormatStr string
	grpcClient    *client.Client
)

// RunnersCmd represents the runners command
var RunnersCmd = &cobra.Command{
	Use:   "runners",
	Short: "Manage runners",
	Long:  `Manage runner instances including creating, listing, and executing commands.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
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
		
		var err error
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

		// Parse environment variables
		envMap := make(map[string]string)
		for _, env := range envVars {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				envMap[parts[0]] = parts[1]
			}
		}

		req := &gradv1.CreateRunnerRequest{
			Name: name,
			Env:  envMap,
		}
		
		// Add workspace configuration if S3 bucket is specified
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
	Use:   "delete RUNNER_ID",
	Short: "Delete a runner",
	Long:  `Delete a runner instance.`,
	Aliases: []string{"rm"},
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
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