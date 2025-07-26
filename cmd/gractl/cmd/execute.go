package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/strrl/gra/cmd/gractl/client"
	gradv1 "github.com/strrl/gra/gen/grad/v1"
)

// ExecuteCmd represents the top-level execute command
var ExecuteCmd = &cobra.Command{
	Use:   "execute COMMAND [args...]",
	Short: "Execute a command (auto-creates runner if needed)",
	Long: `Execute a command with automatic runner provisioning. 
If no runners are available, a new runner will be created automatically.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Get flags
		serverAddress, _ := cmd.Flags().GetString("server")
		shell, _ := cmd.Flags().GetString("shell")
		timeout, _ := cmd.Flags().GetInt32("timeout")
		workdir, _ := cmd.Flags().GetString("workdir")
		
		// Join command arguments
		command := strings.Join(args, " ")

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

		// Create request
		req := &gradv1.ExecuteCommandRequest{
			Command:    command,
			Shell:      shell,
			Timeout:    timeout,
			WorkingDir: workdir,
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
				if err.Error() == "EOF" {
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