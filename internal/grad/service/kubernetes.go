package service

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Well-known constants
const (
	// Default runner image built by skaffold with org/domain prefix
	// In dev mode, skaffold uses dynamic tags (e.g., :v1.17.1-38-g1c6517887)
	// Use RUNNER_IMAGE environment variable to override with actual dynamic tag
	DefaultRunnerImage = "ghcr.io/strrl/grad-runner:latest"
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
	// Use in-cluster configuration when running in a pod
	kubeConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
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

// GetRunnerPod retrieves a runner pod by ID
func (k *KubernetesClient) GetRunnerPod(ctx context.Context, runnerID string) (*corev1.Pod, error) {
	podName := k.getPodName(runnerID)

	pod, err := k.clientset.CoreV1().Pods(k.config.Namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get runner pod: %w", err)
	}

	return pod, nil
}

// ListRunnerPods lists all runner pods
func (k *KubernetesClient) ListRunnerPods(ctx context.Context) (*corev1.PodList, error) {
	labelSelector := labels.Set{
		"app":  "grad-runner",
		"type": "runner",
	}.AsSelector()

	pods, err := k.clientset.CoreV1().Pods(k.config.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector.String(),
	})
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

// ExecuteCommand executes a command in a runner pod
func (k *KubernetesClient) ExecuteCommand(ctx context.Context, runnerID, command string) (*ExecuteCommandResult, error) {
	// For now, we'll return a simulated result
	// In a real implementation, this would use kubectl exec or SSH
	startTime := time.Now()

	// Simulate execution time
	time.Sleep(100 * time.Millisecond)

	executionTime := time.Since(startTime)

	return &ExecuteCommandResult{
		Output:     fmt.Sprintf("Executed command in runner %s: %s", runnerID, command),
		Error:      "",
		ExitCode:   0,
		DurationMS: executionTime.Nanoseconds() / 1000000,
	}, nil
}
