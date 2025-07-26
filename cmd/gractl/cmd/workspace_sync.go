package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	gradv1 "github.com/strrl/gra/gen/grad/v1"
	"github.com/strrl/gra/cmd/gractl/client"
)

// workspaceSyncCmd represents the workspace-sync command
var workspaceSyncCmd = &cobra.Command{
	Use:   "workspace-sync RUNNER_ID",
	Short: "Mount runner workspace locally using sshfs",
	Long: `Mount the runner's /workspace directory to a local directory using sshfs over kubectl port-forward.

This command will:
1. Check that the runner exists and is running
2. Create a local directory at ./runners/RUNNER_ID/workspace
3. Start kubectl port-forward to tunnel SSH traffic
4. Mount the remote /workspace using sshfs
5. Keep the mount active until interrupted (Ctrl+C)

Requirements:
- kubectl must be available and configured for the cluster
- sshfs must be installed on the local machine
- The runner must have been created with SSH public key support
- The runner must be in 'running' status

Example:
  gractl runners workspace-sync runner-1

The mounted workspace will be available at:
  ./runners/runner-1/workspace/

Press Ctrl+C to unmount and clean up.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runnerID := args[0]

		// Check dependencies first
		if err := checkDependencies(); err != nil {
			fmt.Fprintf(os.Stderr, "Dependency check failed: %v\n", err)
			os.Exit(1)
		}

		// Verify runner exists and is running
		runner, err := getRunnerStatus(runnerID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get runner status: %v\n", err)
			os.Exit(1)
		}

		if runner.Status != gradv1.RunnerStatus_RUNNER_STATUS_RUNNING {
			fmt.Fprintf(os.Stderr, "Runner %s is not running (status: %s). Please wait for it to start.\n", 
				runnerID, runner.Status.String())
			os.Exit(1)
		}

		// Create local workspace directory
		workspaceDir := client.GetRunnerWorkspaceDir(runnerID)
		if err := client.CreateLocalDirectory(workspaceDir); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create local workspace directory: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Created local workspace directory: %s\n", workspaceDir)

		// Start kubectl port-forward
		localPort, portForwardCmd, err := startPortForward(runnerID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start port forwarding: %v\n", err)
			os.Exit(1)
		}
		defer func() {
			if portForwardCmd != nil && portForwardCmd.Process != nil {
				portForwardCmd.Process.Kill()
			}
		}()

		fmt.Printf("Port forwarding started: localhost:%d -> %s:22\n", localPort, runnerID)

		// Wait a moment for port forwarding to establish
		time.Sleep(2 * time.Second)

		// Mount workspace using sshfs
		_, err = startSSHFSMount(localPort, workspaceDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to mount workspace: %v\n", err)
			os.Exit(1)
		}
		defer func() {
			unmountWorkspace(workspaceDir)
		}()

		fmt.Printf("Workspace mounted: %s -> %s\n", runnerID+":/workspace", workspaceDir)
		fmt.Println("Press Ctrl+C to unmount and exit...")

		// Wait for interrupt signal
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		fmt.Println("\nReceived interrupt signal, cleaning up...")

		// Cleanup will be handled by defer statements
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

// getRunnerStatus retrieves the current status of a runner
func getRunnerStatus(runnerID string) (*gradv1.Runner, error) {
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

	podName := runnerID // Assuming pod name matches runner ID
	portMapping := fmt.Sprintf("%d:22", localPort)

	cmd := exec.Command("kubectl", "port-forward", "pod/"+podName, portMapping)
	
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
		"runner@localhost:/workspace", // remote path
		mountPoint,                    // local mount point
		"-p", portStr,                // SSH port
		"-o", "reconnect",            // automatically reconnect
		"-o", "UserKnownHostsFile=/dev/null", // skip host key verification
		"-o", "StrictHostKeyChecking=no",     // skip host key checking
		"-o", "PasswordAuthentication=no",    // use key-based auth only
		"-o", "IdentitiesOnly=yes",           // only use specified identity
	)

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
	RunnersCmd.AddCommand(workspaceSyncCmd)
}