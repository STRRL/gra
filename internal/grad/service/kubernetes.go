package service

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Well-known constants
const (
	// Default runner image built by skaffold with org/domain prefix
	// In dev mode, skaffold uses dynamic tags (e.g., :v1.17.1-38-g1c6517887)
	// Use RUNNER_IMAGE environment variable to override with actual dynamic tag
	DefaultRunnerImage = "ghcr.io/strrl/grad-runner:latest"
	
	// Default S3FS sidecar image built by skaffold
	// Use S3FS_IMAGE environment variable to override with actual dynamic tag
	DefaultS3FSImage = "ghcr.io/strrl/grad-runner-s3fs:latest"

	// Kubernetes annotations and labels for runner management
	RunnerAnnotationPrefix = "grad.io/"
	RunnerLabelSelector    = "app.kubernetes.io/managed-by=grad"
	RunnerComponentLabel   = "app.kubernetes.io/component=runner"
	RunnerFinalizer        = "grad.io/runner-finalizer"

	// Runner-specific annotations
	RunnerIDAnnotation      = RunnerAnnotationPrefix + "runner-id"
	RunnerNameAnnotation    = RunnerAnnotationPrefix + "runner-name"
	RunnerStatusAnnotation  = RunnerAnnotationPrefix + "status"
	RunnerCreatedAnnotation = RunnerAnnotationPrefix + "created-at"
)

// RunnerSpec holds resource specifications for a runner preset
type RunnerSpec struct {
	// Kubernetes resource string format
	CPU     string
	Memory  string
	Storage string

	// Numeric values for domain objects
	CPUMillicores int32
	MemoryMB      int32
	StorageGB     int32
}

// RunnerSpecPreset holds all available runner presets
var RunnerSpecPreset = struct {
	Small  RunnerSpec
	Medium RunnerSpec
	Large  RunnerSpec
}{
	// Small preset: 2c2g40g (currently used)
	Small: RunnerSpec{
		CPU:           "2000m",
		Memory:        "2Gi",
		Storage:       "40Gi",
		CPUMillicores: 2000,
		MemoryMB:      2048,
		StorageGB:     40,
	},
	// Medium preset: 4c4g40g (future)
	Medium: RunnerSpec{
		CPU:           "4000m",
		Memory:        "4Gi",
		Storage:       "40Gi",
		CPUMillicores: 4000,
		MemoryMB:      4096,
		StorageGB:     40,
	},
	// Large preset: 8c8g40g (future)
	Large: RunnerSpec{
		CPU:           "8000m",
		Memory:        "8Gi",
		Storage:       "40Gi",
		CPUMillicores: 8000,
		MemoryMB:      8192,
		StorageGB:     40,
	},
}

// GetCurrentRunnerSpec returns the currently used runner specification
// Currently hardcoded to Small preset, but can be made configurable in the future
func GetCurrentRunnerSpec() RunnerSpec {
	return RunnerSpecPreset.Small
}

// GetEffectiveRunnerImage returns the runner image that will be used
// Takes into account environment variable overrides for skaffold dynamic tags
func GetEffectiveRunnerImage() string {
	config := loadKubernetesConfig()
	return config.RunnerImage
}

// KubernetesConfig holds configuration for Kubernetes operations
type KubernetesConfig struct {
	Namespace      string
	RunnerImage    string
	S3FSImage      string
	DefaultCPU     string
	DefaultMemory  string
	DefaultStorage string
	SSHPort        int32
}

// DefaultKubernetesConfig returns default configuration with hardcoded "small" preset
func DefaultKubernetesConfig() *KubernetesConfig {
	return &KubernetesConfig{
		Namespace: "default",
		// Default runner image - can be overridden by RUNNER_IMAGE env var for skaffold dynamic tags
		RunnerImage: DefaultRunnerImage,
		// Default S3FS sidecar image - can be overridden by S3FS_IMAGE env var for skaffold dynamic tags
		S3FSImage:   DefaultS3FSImage,
		// Small preset configuration
		DefaultCPU:     RunnerSpecPreset.Small.CPU,
		DefaultMemory:  RunnerSpecPreset.Small.Memory,
		DefaultStorage: RunnerSpecPreset.Small.Storage,
		SSHPort:        22,
	}
}

// KubernetesClient wraps the Kubernetes client with runner-specific operations
type KubernetesClient struct {
	clientset *kubernetes.Clientset
	config    *KubernetesConfig
}

// NewKubernetesClient creates a new Kubernetes client for runner management
func NewKubernetesClient(config *KubernetesConfig) (*KubernetesClient, error) {
	var kubeConfig *rest.Config
	var err error

	// Try in-cluster configuration first (when running in a pod)
	kubeConfig, err = rest.InClusterConfig()
	if err != nil {
		// Fall back to local kubeconfig for development
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
		if err != nil {
			return nil, fmt.Errorf("failed to get kubernetes config (tried in-cluster and local kubeconfig): %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	if config == nil {
		config = DefaultKubernetesConfig()
	}

	return &KubernetesClient{
		clientset: clientset,
		config:    config,
	}, nil
}

// CreateRunnerPod creates a new pod for a runner
func (k *KubernetesClient) CreateRunnerPod(ctx context.Context, runner *Runner) error {
	req := BuildPodCreationRequest(runner, k.config)
	pod := req.ToPodSpec()

	_, err := k.clientset.CoreV1().Pods(k.config.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create runner pod: %w", err)
	}

	return nil
}

// DeleteRunnerPod deletes a runner pod
func (k *KubernetesClient) DeleteRunnerPod(ctx context.Context, runnerID string) error {
	req := BuildPodDeletionRequest(runnerID, k.config)

	err := k.clientset.CoreV1().Pods(req.Namespace).Delete(ctx, req.PodName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete runner pod: %w", err)
	}

	return nil
}

// GetRunnerPod gets a specific runner pod by ID
func (k *KubernetesClient) GetRunnerPod(ctx context.Context, runnerID string) (*corev1.Pod, error) {
	podName := k.getPodName(runnerID)

	pod, err := k.clientset.CoreV1().Pods(k.config.Namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get runner pod: %w", err)
	}

	return pod, nil
}

// ListRunnerPods lists all runner pods using label selectors with optional status filtering
func (k *KubernetesClient) ListRunnerPods(ctx context.Context) (*corev1.PodList, error) {
	labelSelector := RunnerLabelSelector + "," + RunnerComponentLabel

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	pods, err := k.clientset.CoreV1().Pods(k.config.Namespace).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list runner pods: %w", err)
	}

	return pods, nil
}

// GetPodStatus maps Kubernetes pod status to runner status (uses pure function)
func (k *KubernetesClient) GetPodStatus(pod *corev1.Pod) RunnerStatus {
	return MapPodStatusToRunnerStatus(pod)
}

// getPodName generates a consistent pod name for a runner
func (k *KubernetesClient) getPodName(runnerID string) string {
	return fmt.Sprintf("grad-runner-%s", runnerID)
}

// ExecuteCommandStream executes a command in a runner pod with streaming output
func (k *KubernetesClient) ExecuteCommandStream(ctx context.Context, runnerID, command string, stdoutCh, stderrCh chan<- []byte) (int32, error) {
	slog.Info("ExecuteCommandStream called",
		"runnerID", runnerID,
		"command", command)

	// For this demo, we'll execute the command locally since we don't have real K8s runners yet
	// In production, this would use kubectl exec with streaming to the actual pod
	cmd := exec.CommandContext(ctx, "bash", "-c", command)

	slog.Info("Created command", "cmd", cmd.String())

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		slog.Error("Failed to create stdout pipe", "error", err)
		return 1, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		slog.Error("Failed to create stderr pipe", "error", err)
		return 1, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	slog.Info("Starting command execution")
	if err := cmd.Start(); err != nil {
		slog.Error("Failed to start command", "error", err)
		return 1, fmt.Errorf("failed to start command: %w", err)
	}

	slog.Info("Command started successfully, setting up streaming")

	// Stream stdout in a goroutine
	go func() {
		defer func() {
			slog.Info("Closing stdout channel")
			close(stdoutCh)
		}()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) > 0 {
				// Copy the line since scanner reuses the buffer
				lineCopy := make([]byte, len(line)+1)
				copy(lineCopy, line)
				lineCopy[len(line)] = '\n'

				select {
				case <-ctx.Done():
					slog.Info("Context cancelled, stopping stdout streaming")
					return
				case stdoutCh <- lineCopy:
					slog.Debug("Sent stdout line", "line", string(lineCopy))
				}
			}
		}
		if err := scanner.Err(); err != nil {
			slog.Error("Error reading stdout", "error", err)
		}
	}()

	// Stream stderr in a goroutine
	go func() {
		defer func() {
			slog.Info("Closing stderr channel")
			close(stderrCh)
		}()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) > 0 {
				// Copy the line since scanner reuses the buffer
				lineCopy := make([]byte, len(line)+1)
				copy(lineCopy, line)
				lineCopy[len(line)] = '\n'

				select {
				case <-ctx.Done():
					slog.Info("Context cancelled, stopping stderr streaming")
					return
				case stderrCh <- lineCopy:
					slog.Debug("Sent stderr line", "line", string(lineCopy))
				}
			}
		}
		if err := scanner.Err(); err != nil {
			slog.Error("Error reading stderr", "error", err)
		}
	}()

	// Wait for command to complete
	slog.Info("Waiting for command to complete")
	err = cmd.Wait()
	if err != nil {
		slog.Error("Command execution failed", "error", err)
		if exitError, ok := err.(*exec.ExitError); ok {
			slog.Info("Command exited with non-zero code", "exit_code", exitError.ExitCode())
			return int32(exitError.ExitCode()), nil
		}
		return 1, err
	}

	slog.Info("Command completed successfully")
	return 0, nil
}

// PodToRunner converts a Kubernetes pod to a domain Runner object
func PodToRunner(pod *corev1.Pod) *Runner {
	runner := &Runner{
		ID:   pod.Annotations[RunnerIDAnnotation],
		Name: pod.Annotations[RunnerNameAnnotation],
	}

	// Always derive status from actual pod state (pod phase and conditions)
	// This ensures we get the real-time status rather than stale annotations
	runner.Status = MapPodStatusToRunnerStatus(pod)

	// Parse timestamps
	if createdStr, ok := pod.Annotations[RunnerCreatedAnnotation]; ok {
		if createdAt, err := time.Parse(time.RFC3339, createdStr); err == nil {
			runner.CreatedAt = createdAt.Unix()
		}
	} else {
		runner.CreatedAt = pod.CreationTimestamp.Unix()
	}

	runner.UpdatedAt = runner.CreatedAt
	if pod.Status.StartTime != nil {
		runner.UpdatedAt = pod.Status.StartTime.Unix()
	}

	// Get IP address
	runner.IPAddress = pod.Status.PodIP

	// Extract resource requirements
	if len(pod.Spec.Containers) > 0 {
		container := pod.Spec.Containers[0]
		if requests := container.Resources.Requests; requests != nil {
			runner.Resources = &ResourceRequirements{}

			if cpu := requests.Cpu(); cpu != nil {
				runner.Resources.CPUMillicores = int32(cpu.MilliValue())
			}
			if memory := requests.Memory(); memory != nil {
				runner.Resources.MemoryMB = int32(memory.Value() / (1024 * 1024))
			}
			if storage := requests.StorageEphemeral(); storage != nil {
				runner.Resources.StorageGB = int32(storage.Value() / (1024 * 1024 * 1024))
			}
		}
	}

	// Extract environment variables
	runner.Env = make(map[string]string)
	if len(pod.Spec.Containers) > 0 {
		for _, envVar := range pod.Spec.Containers[0].Env {
			runner.Env[envVar.Name] = envVar.Value
		}
	}

	return runner
}


// AddRunnerFinalizer adds the runner finalizer to a pod
func (k *KubernetesClient) AddRunnerFinalizer(ctx context.Context, podName string) error {
	pod, err := k.clientset.CoreV1().Pods(k.config.Namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get pod for finalizer: %w", err)
	}

	// Check if finalizer already exists
	for _, finalizer := range pod.Finalizers {
		if finalizer == RunnerFinalizer {
			return nil // Already has finalizer
		}
	}

	// Add finalizer
	pod.Finalizers = append(pod.Finalizers, RunnerFinalizer)

	_, err = k.clientset.CoreV1().Pods(k.config.Namespace).Update(ctx, pod, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	return nil
}

// RemoveRunnerFinalizer removes the runner finalizer from a pod
func (k *KubernetesClient) RemoveRunnerFinalizer(ctx context.Context, podName string) error {
	pod, err := k.clientset.CoreV1().Pods(k.config.Namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get pod for finalizer removal: %w", err)
	}

	// Remove finalizer
	finalizers := make([]string, 0)
	for _, finalizer := range pod.Finalizers {
		if finalizer != RunnerFinalizer {
			finalizers = append(finalizers, finalizer)
		}
	}
	pod.Finalizers = finalizers

	_, err = k.clientset.CoreV1().Pods(k.config.Namespace).Update(ctx, pod, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to remove finalizer: %w", err)
	}

	return nil
}
