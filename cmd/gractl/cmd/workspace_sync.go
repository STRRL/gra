package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	gradv1 "github.com/strrl/gra/gen/grad/v1"
	"github.com/strrl/gra/cmd/gractl/client"
)

// WorkspaceSyncCmd represents the workspace-sync command  
var WorkspaceSyncCmd = &cobra.Command{
	Use:   "workspace-sync [RUNNER_ID]",
	Short: "Mount runner workspaces locally using sshfs",
	Long: `Mount runner workspaces locally using sshfs over kubectl port-forward.

If RUNNER_ID is specified, sync only that runner's workspace.
If RUNNER_ID is omitted, sync all running runners' workspaces.

For each runner, this command will:
1. Check that the runner exists and is running
2. Create a local directory at ./runners/RUNNER_ID/workspace
3. Start kubectl port-forward to tunnel SSH traffic
4. Mount the remote /workspace using sshfs
5. Keep the mount active until interrupted (Ctrl+C)

Requirements:
- kubectl must be available and configured for the cluster
- sshfs must be installed on the local machine
- The runner(s) must have been created with SSH public key support
- The runner(s) must be in 'running' status

Examples:
  gractl workspace-sync runner-1    # Sync specific runner
  gractl workspace-sync             # Sync all running runners

The mounted workspace(s) will be available at:
  ./runners/runner-1/workspace/
  ./runners/runner-2/workspace/
  ...

Press Ctrl+C to unmount and clean up.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Initialize gRPC client
		serverAddress, _ := cmd.Flags().GetString("server")
		cfg := &client.Config{
			ServerAddress: serverAddress,
		}
		
		grpcClient, err := client.NewClient(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to connect to server: %v\n", err)
			os.Exit(1)
		}
		defer grpcClient.Close()

		// Check dependencies first
		if err := checkDependencies(); err != nil {
			fmt.Fprintf(os.Stderr, "Dependency check failed: %v\n", err)
			os.Exit(1)
		}

		// Determine which runners to sync
		var runnersToSync []string
		if len(args) == 1 {
			// Single runner specified
			runnersToSync = []string{args[0]}
		} else {
			// Get all running runners
			runningRunners, err := getRunningRunners(grpcClient)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to get running runners: %v\n", err)
				os.Exit(1)
			}
			runnersToSync = runningRunners
		}

		if len(runnersToSync) == 0 {
			fmt.Println("No running runners found to sync.")
			os.Exit(0)
		}

		fmt.Printf("Syncing %d runner(s): %s\n", len(runnersToSync), strings.Join(runnersToSync, ", "))

		// Verify all runners exist and are running
		for _, runnerID := range runnersToSync {
			runner, err := getRunnerStatus(grpcClient, runnerID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to get runner status for %s: %v\n", runnerID, err)
				os.Exit(1)
			}

			if runner.Status != gradv1.RunnerStatus_RUNNER_STATUS_RUNNING {
				fmt.Fprintf(os.Stderr, "Runner %s is not running (status: %s). Skipping.\n", 
					runnerID, runner.Status.String())
				continue
			}
		}

		// Setup workspace syncs for all runners
		type runnerSync struct {
			runnerID       string
			workspaceDir   string
			portForwardCmd *exec.Cmd
			sshfsCmd       *exec.Cmd
			localPort      int
		}

		var activeSyncs []runnerSync
		var syncMutex sync.Mutex

		// Start workspace sync for each runner
		for _, runnerID := range runnersToSync {
			// Create local workspace directory
			workspaceDir := client.GetRunnerWorkspaceDir(runnerID)
			if err := client.CreateLocalDirectory(workspaceDir); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create local workspace directory for %s: %v\n", runnerID, err)
				continue
			}

			fmt.Printf("Created local workspace directory: %s\n", workspaceDir)

			// Start kubectl port-forward
			localPort, portForwardCmd, err := startPortForward(runnerID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to start port forwarding for %s: %v\n", runnerID, err)
				continue
			}

			fmt.Printf("Port forwarding started: localhost:%d -> %s:22\n", localPort, runnerID)

			// Wait a moment for port forwarding to establish
			time.Sleep(2 * time.Second)

			// Mount workspace using sshfs
			sshfsCmd, err := startSSHFSMount(localPort, workspaceDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to mount workspace for %s: %v\n", runnerID, err)
				if portForwardCmd != nil && portForwardCmd.Process != nil {
					portForwardCmd.Process.Kill()
				}
				continue
			}

			fmt.Printf("Workspace mounted: %s:/workspace -> %s\n", runnerID, workspaceDir)

			// Add to active syncs
			syncMutex.Lock()
			activeSyncs = append(activeSyncs, runnerSync{
				runnerID:       runnerID,
				workspaceDir:   workspaceDir,
				portForwardCmd: portForwardCmd,
				sshfsCmd:       sshfsCmd,
				localPort:      localPort,
			})
			syncMutex.Unlock()
		}

		if len(activeSyncs) == 0 {
			fmt.Println("No workspace syncs were successfully established.")
			os.Exit(1)
		}

		fmt.Printf("\nSuccessfully synced %d workspace(s). Press Ctrl+C to unmount and exit...\n", len(activeSyncs))

		// Setup cleanup function
		cleanupAll := func() {
			fmt.Println("\nCleaning up all workspace syncs...")
			syncMutex.Lock()
			defer syncMutex.Unlock()
			
			for _, sync := range activeSyncs {
				fmt.Printf("Cleaning up %s...\n", sync.runnerID)
				
				// Unmount workspace
				unmountWorkspace(sync.workspaceDir)
				
				// Kill sshfs process
				if sync.sshfsCmd != nil && sync.sshfsCmd.Process != nil {
					sync.sshfsCmd.Process.Kill()
				}
				
				// Kill port forwarding process
				if sync.portForwardCmd != nil && sync.portForwardCmd.Process != nil {
					sync.portForwardCmd.Process.Kill()
				}
			}
		}
		defer cleanupAll()

		// Wait for interrupt signal
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
	},
}

// checkDependencies verifies that required external commands are available
func checkDependencies() error {
	if err := client.CheckCommandAvailable("kubectl"); err != nil {
		return fmt.Errorf("kubectl not found: %w", err)
	}

	if err := client.CheckCommandAvailable("sshfs"); err != nil {
		return fmt.Errorf("sshfs not found: %w. Please install sshfs", err)
	}

	return nil
}

// getRunningRunners retrieves all runners with RUNNING status
func getRunningRunners(grpcClient *client.Client) ([]string, error) {
	req := &gradv1.ListRunnersRequest{
		Status: gradv1.RunnerStatus_RUNNER_STATUS_RUNNING,
		Limit:  100, // reasonable limit for workspace sync
	}

	resp, err := grpcClient.RunnerService().ListRunners(context.Background(), req)
	if err != nil {
		return nil, err
	}

	var runnerIDs []string
	for _, runner := range resp.Runners {
		runnerIDs = append(runnerIDs, runner.Id)
	}

	return runnerIDs, nil
}

// getRunnerStatus retrieves the current status of a runner
func getRunnerStatus(grpcClient *client.Client, runnerID string) (*gradv1.Runner, error) {
	req := &gradv1.GetRunnerRequest{
		RunnerId: runnerID,
	}

	resp, err := grpcClient.RunnerService().GetRunner(context.Background(), req)
	if err != nil {
		return nil, err
	}

	return resp.Runner, nil
}

// startPortForward starts kubectl port-forward and returns the local port and process
func startPortForward(runnerID string) (int, *exec.Cmd, error) {
	// Use a high port number to avoid conflicts
	localPort := 2222 + (int(time.Now().Unix()) % 1000)

	// Pod name format matches what's used in kubernetes.go: grad-runner-{runnerID}
	podName := fmt.Sprintf("grad-runner-%s", runnerID)
	portMapping := fmt.Sprintf("%d:22", localPort)

	cmd := exec.Command("kubectl", "port-forward", "pod/"+podName, portMapping)
	
	// Debug: Print the kubectl command for debugging
	fmt.Printf("DEBUG: Executing kubectl command: %s\n", strings.Join(cmd.Args, " "))
	
	// Start the process
	if err := cmd.Start(); err != nil {
		return 0, nil, fmt.Errorf("failed to start kubectl port-forward: %w", err)
	}

	return localPort, cmd, nil
}

// startSSHFSMount mounts the remote workspace using sshfs
func startSSHFSMount(localPort int, mountPoint string) (*exec.Cmd, error) {
	portStr := strconv.Itoa(localPort)
	
	// sshfs command with appropriate options
	cmd := exec.Command("sshfs",
		"root@localhost:/workspace", // remote path - use root user for proper permissions
		mountPoint,                  // local mount point
		"-p", portStr,              // SSH port
		"-o", "reconnect",          // automatically reconnect
		"-o", "UserKnownHostsFile=/dev/null", // skip host key verification
		"-o", "StrictHostKeyChecking=no",     // skip host key checking
		"-o", "PasswordAuthentication=no",    // use key-based auth only
		"-o", "IdentitiesOnly=yes",           // only use specified identity
	)

	// Debug: Print the full sshfs command for debugging
	fmt.Printf("DEBUG: Executing sshfs command: %s %s\n", cmd.Path, strings.Join(cmd.Args[1:], " "))
	fmt.Printf("DEBUG: Full command: %s\n", strings.Join(cmd.Args, " "))

	// Run sshfs in the background
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start sshfs: %w", err)
	}

	// Give sshfs a moment to establish the mount
	time.Sleep(1 * time.Second)

	// Verify mount was successful by checking if directory is accessible
	if _, err := os.Stat(mountPoint); err != nil {
		cmd.Process.Kill() // Kill sshfs if mount failed
		return nil, fmt.Errorf("mount verification failed: %w", err)
	}

	return cmd, nil
}

// unmountWorkspace safely unmounts the sshfs filesystem
func unmountWorkspace(mountPoint string) {
	fmt.Printf("Unmounting workspace: %s\n", mountPoint)

	// Use fusermount to unmount (standard way to unmount FUSE filesystems)
	cmd := exec.Command("fusermount", "-u", mountPoint)
	if err := cmd.Run(); err != nil {
		// If fusermount fails, try umount as fallback
		cmd = exec.Command("umount", mountPoint)
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to unmount %s: %v\n", mountPoint, err)
		}
	}
}

func init() {
	// Add global flags to the workspace-sync command
	WorkspaceSyncCmd.Flags().String("server", "localhost:9090", "gRPC server address")
}

// init() removed from runners.go - WorkspaceSyncCmd is now registered as a top-level command in main.go