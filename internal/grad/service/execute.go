package service

import (
	"context"
	"fmt"
	"time"
)

// executeService implements the ExecuteService interface
type executeService struct {
	runnerService RunnerService
}

// NewExecuteService creates a new execute service
func NewExecuteService(runnerService RunnerService) ExecuteService {
	return &executeService{
		runnerService: runnerService,
	}
}

// ExecuteCommand executes a command, creating a runner if needed
func (s *executeService) ExecuteCommand(ctx context.Context, req *ExecuteCommandRequest, stdoutCh, stderrCh chan<- []byte) (int32, error) {
	// First, try to find an available running runner
	runners, _, err := s.runnerService.ListRunners(ctx, &ListOptions{
		Status: RunnerStatusRunning,
		Limit:  10,
	})
	if err != nil {
		return 1, fmt.Errorf("failed to list runners: %w", err)
	}

	var runnerID string
	if len(runners) > 0 {
		// Use the first available running runner
		runnerID = runners[0].ID
	} else {
		// No running runners available, create a new one
		createReq := &CreateRunnerRequest{
			Name: fmt.Sprintf("auto-runner-%d", time.Now().Unix()),
		}

		runner, err := s.runnerService.CreateRunner(ctx, createReq)
		if err != nil {
			return 1, fmt.Errorf("failed to create runner: %w", err)
		}

		runnerID = runner.ID

		// Wait for runner to be ready (with timeout)
		waitCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		runnerReady := false
		for !runnerReady {
			select {
			case <-waitCtx.Done():
				return 1, fmt.Errorf("timeout waiting for runner to be ready")
			case <-ticker.C:
				runner, err := s.runnerService.GetRunner(ctx, runnerID)
				if err != nil {
					return 1, fmt.Errorf("failed to get runner status: %w", err)
				}

				if runner.Status == RunnerStatusRunning {
					// Runner is ready, exit the wait loop
					runnerReady = true
				} else if runner.Status == RunnerStatusError || runner.Status == RunnerStatusStopped {
					return 1, fmt.Errorf("runner failed to start: status=%s", runner.Status)
				}
			}
		}
	}

	// Update the request with the runner ID
	execReq := &ExecuteCommandRequest{
		RunnerID:   runnerID,
		Command:    req.Command,
		Shell:      req.Shell,
		Timeout:    req.Timeout,
		WorkingDir: req.WorkingDir,
	}

	// Execute the command in the runner
	return s.runnerService.ExecuteCommandStream(ctx, execReq, stdoutCh, stderrCh)
}
