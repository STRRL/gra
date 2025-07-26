package service

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
)

// runnerService implements the RunnerService interface using Kubernetes API
type runnerService struct {
	k8sClient *KubernetesClient
}

// NewRunnerService creates a new runner service
func NewRunnerService(k8sClient *KubernetesClient) RunnerService {
	return &runnerService{
		k8sClient: k8sClient,
	}
}

// CreateRunner creates a new runner instance
func (s *runnerService) CreateRunner(ctx context.Context, req *CreateRunnerRequest) (*Runner, error) {
	// Generate simple runner ID by counting existing runners
	runnerID, err := s.generateRunnerID(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to generate runner ID: %v", ErrKubernetesAPI, err)
	}

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

	// Create runner object for pod creation
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

	// Create Kubernetes pod with proper annotations and finalizers
	if err := s.k8sClient.CreateRunnerPod(ctx, runner); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrKubernetesAPI, err)
	}

	// Get the created pod to return accurate information from Kubernetes
	pod, err := s.k8sClient.GetRunnerPod(ctx, runnerID)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get created pod: %v", ErrKubernetesAPI, err)
	}

	return PodToRunner(pod), nil
}

// DeleteRunner removes a runner instance with proper finalizer cleanup
func (s *runnerService) DeleteRunner(ctx context.Context, runnerID string) error {
	// Check if runner pod exists
	pod, err := s.k8sClient.GetRunnerPod(ctx, runnerID)
	if err != nil {
		return ErrRunnerNotFound
	}

	// Remove finalizer to allow Kubernetes to delete the pod
	if err := s.k8sClient.RemoveRunnerFinalizer(ctx, pod.Name); err != nil {
		return fmt.Errorf("%w: failed to remove finalizer: %v", ErrKubernetesAPI, err)
	}

	// Delete Kubernetes pod
	if err := s.k8sClient.DeleteRunnerPod(ctx, runnerID); err != nil {
		// If pod doesn't exist, that's fine (already deleted)
		if !errors.IsNotFound(err) {
			return fmt.Errorf("%w: %v", ErrKubernetesAPI, err)
		}
	}

	return nil
}

// ListRunners returns all available runners by querying Kubernetes API
func (s *runnerService) ListRunners(ctx context.Context, opts *ListOptions) ([]*Runner, int32, error) {
	// Determine status filter
	status := RunnerStatusUnspecified
	if opts != nil {
		status = opts.Status
	}

	// List runner pods from Kubernetes
	podList, err := s.k8sClient.ListRunnerPods(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %v", ErrKubernetesAPI, err)
	}

	// Convert pods to runners and filter by status
	runners := make([]*Runner, 0, len(podList.Items))
	for _, pod := range podList.Items {
		runner := PodToRunner(&pod)

		// Filter by status if specified
		if status != RunnerStatusUnspecified && runner.Status != status {
			continue
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

// GetRunner returns details about a specific runner by querying Kubernetes API
func (s *runnerService) GetRunner(ctx context.Context, runnerID string) (*Runner, error) {
	// Get runner pod from Kubernetes
	pod, err := s.k8sClient.GetRunnerPod(ctx, runnerID)
	if err != nil {
		return nil, ErrRunnerNotFound
	}

	return PodToRunner(pod), nil
}

// ExecuteCommandStream executes a command in a specific runner with streaming output
func (s *runnerService) ExecuteCommandStream(ctx context.Context, req *ExecuteCommandRequest, stdoutCh, stderrCh chan<- []byte) (int32, error) {
	// Check if runner exists and is running
	pod, err := s.k8sClient.GetRunnerPod(ctx, req.RunnerID)
	if err != nil {
		return 1, ErrRunnerNotFound
	}

	runner := PodToRunner(pod)
	if runner.Status != RunnerStatusRunning {
		return 1, ErrRunnerNotRunning
	}

	// Execute command via Kubernetes client with streaming
	exitCode, err := s.k8sClient.ExecuteCommandStream(ctx, req.RunnerID, req.Command, stdoutCh, stderrCh)
	if err != nil {
		return 1, fmt.Errorf("%w: %v", ErrCommandExecution, err)
	}

	return exitCode, nil
}

// generateRunnerID generates a simple incrementing runner ID (runner-1, runner-2, etc.)
func (s *runnerService) generateRunnerID(ctx context.Context) (string, error) {
	// List existing runners to find the next available ID
	podList, err := s.k8sClient.ListRunnerPods(ctx)
	if err != nil {
		return "", err
	}

	maxID := 0
	for _, pod := range podList.Items {
		if runnerIDStr, ok := pod.Annotations[RunnerIDAnnotation]; ok {
			// Extract number from runner-N format
			if len(runnerIDStr) > 7 && runnerIDStr[:7] == "runner-" {
				var currentID int
				if n, parseErr := fmt.Sscanf(runnerIDStr, "runner-%d", &currentID); parseErr == nil && n == 1 {
					if currentID > maxID {
						maxID = currentID
					}
				}
			}
		}
	}

	return fmt.Sprintf("runner-%d", maxID+1), nil
}
