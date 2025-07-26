package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
)

// runnerService implements the RunnerService interface
type runnerService struct {
	k8sClient *KubernetesClient
	mu        sync.RWMutex

	// In-memory cache for runner metadata
	// In production, this could be replaced with a database
	runners map[string]*Runner

	// Runner ID counter
	runnerIDCounter int64
}

// NewRunnerService creates a new runner service
func NewRunnerService(k8sClient *KubernetesClient) RunnerService {
	return &runnerService{
		k8sClient: k8sClient,
		runners:   make(map[string]*Runner),
	}
}

// CreateRunner creates a new runner instance
func (s *runnerService) CreateRunner(ctx context.Context, req *CreateRunnerRequest) (*Runner, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate runner ID
	s.runnerIDCounter++
	runnerID := fmt.Sprintf("runner-%d", s.runnerIDCounter)

	// Use provided name or generate one
	name := req.Name
	if name == "" {
		name = runnerID
	}

	// Use hardcoded "small" preset resources: 2c2g40g
	resources := &ResourceRequirements{
		CPUMillicores: RunnerSpecPreset.Small.CPUMillicores,
		MemoryMB:      RunnerSpecPreset.Small.MemoryMB,
		StorageGB:     RunnerSpecPreset.Small.StorageGB,
	}

	// Create runner
	runner := &Runner{
		ID:        runnerID,
		Name:      name,
		Status:    RunnerStatusCreating,
		Resources: resources,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		SSH: &SSHDetails{
			Host:     "localhost", // Will be updated with actual pod IP
			Port:     22,
			Username: "runner",
		},
		IPAddress: "127.0.0.1", // Will be updated with actual pod IP
		Env:       req.Env,
	}

	// Store runner in cache
	s.runners[runnerID] = runner

	// Create Kubernetes pod
	if err := s.k8sClient.CreateRunnerPod(ctx, runner); err != nil {
		// Remove from cache if pod creation fails
		delete(s.runners, runnerID)
		return nil, fmt.Errorf("%w: %v", ErrKubernetesAPI, err)
	}

	// Start async status monitoring
	go s.monitorRunnerStatus(runnerID)

	return runner, nil
}

// DeleteRunner removes a runner instance
func (s *runnerService) DeleteRunner(ctx context.Context, runnerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	runner, exists := s.runners[runnerID]
	if !exists {
		return ErrRunnerNotFound
	}

	// Update status to stopping
	runner.Status = RunnerStatusStopping
	runner.UpdatedAt = time.Now().Unix()

	// Delete Kubernetes pod
	if err := s.k8sClient.DeleteRunnerPod(ctx, runnerID); err != nil {
		// If pod doesn't exist, that's fine (already deleted)
		if !errors.IsNotFound(err) {
			return fmt.Errorf("%w: %v", ErrKubernetesAPI, err)
		}
	}

	// Start async cleanup
	go s.cleanupRunner(runnerID)

	return nil
}

// ListRunners returns all available runners
func (s *runnerService) ListRunners(ctx context.Context, opts *ListOptions) ([]*Runner, int32, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var runners []*Runner

	// Filter by status if specified
	for _, runner := range s.runners {
		if opts != nil && opts.Status != RunnerStatusUnspecified {
			if runner.Status != opts.Status {
				continue
			}
		}
		runners = append(runners, runner)
	}

	// Apply pagination
	total := int32(len(runners))
	if opts != nil {
		offset := opts.Offset
		limit := opts.Limit

		if limit == 0 {
			limit = 50 // Default limit
		}

		if offset >= total {
			runners = []*Runner{}
		} else {
			end := offset + limit
			if end > total {
				end = total
			}
			runners = runners[offset:end]
		}
	}

	return runners, total, nil
}

// GetRunner returns details about a specific runner
func (s *runnerService) GetRunner(ctx context.Context, runnerID string) (*Runner, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	runner, exists := s.runners[runnerID]
	if !exists {
		return nil, ErrRunnerNotFound
	}

	// Update runner status from Kubernetes
	if err := s.updateRunnerStatusFromK8s(ctx, runner); err != nil {
		// Log error but don't fail the request
		// In production, you'd use proper logging
		fmt.Printf("Warning: failed to update runner status: %v\n", err)
	}

	return runner, nil
}

// ExecuteCode executes code in a specific runner
func (s *runnerService) ExecuteCode(ctx context.Context, req *ExecuteCodeRequest) (*ExecuteCodeResult, error) {
	s.mu.RLock()
	runner, exists := s.runners[req.RunnerID]
	s.mu.RUnlock()

	if !exists {
		return nil, ErrRunnerNotFound
	}

	if runner.Status != RunnerStatusRunning {
		return nil, ErrRunnerNotRunning
	}

	// Execute code via Kubernetes client
	result, err := s.k8sClient.ExecuteCommand(ctx, req.RunnerID, req.Code)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCodeExecution, err)
	}

	return result, nil
}

// monitorRunnerStatus monitors runner status in the background
func (s *runnerService) monitorRunnerStatus(runnerID string) {
	// Monitor for up to 5 minutes
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			// Timeout reached, set status to error if still creating
			s.mu.Lock()
			if runner, exists := s.runners[runnerID]; exists && runner.Status == RunnerStatusCreating {
				runner.Status = RunnerStatusError
				runner.UpdatedAt = time.Now().Unix()
			}
			s.mu.Unlock()
			return

		case <-ticker.C:
			s.mu.Lock()
			runner, exists := s.runners[runnerID]
			if !exists {
				s.mu.Unlock()
				return
			}

			// Update status from Kubernetes
			ctx := context.Background()
			if err := s.updateRunnerStatusFromK8s(ctx, runner); err != nil {
				// Continue monitoring on error
				s.mu.Unlock()
				continue
			}

			// Stop monitoring if runner is running or in error state
			if runner.Status == RunnerStatusRunning || runner.Status == RunnerStatusError {
				s.mu.Unlock()
				return
			}
			s.mu.Unlock()
		}
	}
}

// cleanupRunner cleans up runner resources after deletion
func (s *runnerService) cleanupRunner(runnerID string) {
	// Wait a bit for pod deletion to complete
	time.Sleep(2 * time.Second)

	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove from cache
	delete(s.runners, runnerID)
}

// updateRunnerStatusFromK8s updates runner status based on pod status
func (s *runnerService) updateRunnerStatusFromK8s(ctx context.Context, runner *Runner) error {
	pod, err := s.k8sClient.GetRunnerPod(ctx, runner.ID)
	if err != nil {
		if errors.IsNotFound(err) {
			runner.Status = RunnerStatusStopped
			runner.UpdatedAt = time.Now().Unix()
			return nil
		}
		return err
	}

	// Update status based on pod status
	newStatus := s.k8sClient.GetPodStatus(pod)
	if newStatus != runner.Status {
		runner.Status = newStatus
		runner.UpdatedAt = time.Now().Unix()

		// Update IP address if pod is running
		if newStatus == RunnerStatusRunning && pod.Status.PodIP != "" {
			runner.IPAddress = pod.Status.PodIP
			if runner.SSH != nil {
				runner.SSH.Host = pod.Status.PodIP
			}
		}
	}

	return nil
}
