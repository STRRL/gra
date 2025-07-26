package service

import (
	"os"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildPodCreationRequest(t *testing.T) {
	config := &KubernetesConfig{
		Namespace:      "test-namespace",
		RunnerImage:    "test-image:latest",
		DefaultCPU:     RunnerSpecPreset.Small.CPU,
		DefaultMemory:  RunnerSpecPreset.Small.Memory,
		DefaultStorage: RunnerSpecPreset.Small.Storage,
		SSHPort:        22,
	}

	runner := &Runner{
		ID:   "test-runner-123",
		Name: "my-test-runner",
		Resources: &ResourceRequirements{
			CPUMillicores: RunnerSpecPreset.Small.CPUMillicores,
			MemoryMB:      RunnerSpecPreset.Small.MemoryMB,
			StorageGB:     RunnerSpecPreset.Small.StorageGB,
		},
		Env: map[string]string{
			"TEST_VAR": "test_value",
			"DEBUG":    "true",
		},
	}

	req := BuildPodCreationRequest(runner, config)

	// Test basic fields
	if req.PodName != "grad-runner-test-runner-123" {
		t.Errorf("Expected pod name 'grad-runner-test-runner-123', got '%s'", req.PodName)
	}

	if req.Namespace != "test-namespace" {
		t.Errorf("Expected namespace 'test-namespace', got '%s'", req.Namespace)
	}

	if req.RunnerID != "test-runner-123" {
		t.Errorf("Expected runner ID 'test-runner-123', got '%s'", req.RunnerID)
	}

	if req.RunnerName != "my-test-runner" {
		t.Errorf("Expected runner name 'my-test-runner', got '%s'", req.RunnerName)
	}

	// Test preset resource configuration (ignores runner.Resources)
	if req.CPURequest != "2000m" {
		t.Errorf("Expected CPU request '2000m' (preset), got '%s'", req.CPURequest)
	}

	if req.MemoryRequest != "2Gi" {
		t.Errorf("Expected memory request '2Gi' (preset), got '%s'", req.MemoryRequest)
	}

	// Test environment variables
	if req.Env["TEST_VAR"] != "test_value" {
		t.Errorf("Expected env var TEST_VAR='test_value', got '%s'", req.Env["TEST_VAR"])
	}

	if req.Env["DEBUG"] != "true" {
		t.Errorf("Expected env var DEBUG='true', got '%s'", req.Env["DEBUG"])
	}
}

func TestBuildPodCreationRequestWithDefaults(t *testing.T) {
	config := &KubernetesConfig{
		Namespace:      "default",
		RunnerImage:    DefaultRunnerImage,
		DefaultCPU:     RunnerSpecPreset.Small.CPU,
		DefaultMemory:  RunnerSpecPreset.Small.Memory,
		DefaultStorage: RunnerSpecPreset.Small.Storage,
		SSHPort:        22,
	}

	// Runner with no resource requirements
	runner := &Runner{
		ID:        "simple-runner",
		Name:      "simple",
		Resources: nil, // No resources specified
		Env:       map[string]string{},
	}

	req := BuildPodCreationRequest(runner, config)

	// Should use preset configuration regardless of runner.Resources
	if req.CPURequest != "2000m" {
		t.Errorf("Expected preset CPU request '2000m', got '%s'", req.CPURequest)
	}

	if req.MemoryRequest != "2Gi" {
		t.Errorf("Expected preset memory request '2Gi', got '%s'", req.MemoryRequest)
	}
}

func TestPodCreationRequestToPodSpec(t *testing.T) {
	req := &PodCreationRequest{
		PodName:       "test-pod",
		Namespace:     "test-ns",
		RunnerID:      "runner-123",
		RunnerName:    "test-runner",
		Image:         "ghcr.io/strrl/grad-runner:latest",
		CPURequest:    "500m",
		MemoryRequest: "1Gi",
		SSHPort:       22,
		Env: map[string]string{
			"TEST": "value",
		},
	}

	pod := req.ToPodSpec()

	// Test metadata
	if pod.Name != "test-pod" {
		t.Errorf("Expected pod name 'test-pod', got '%s'", pod.Name)
	}

	if pod.Namespace != "test-ns" {
		t.Errorf("Expected namespace 'test-ns', got '%s'", pod.Namespace)
	}

	// Test labels
	if pod.Labels["app"] != "grad-runner" {
		t.Errorf("Expected label app='grad-runner', got '%s'", pod.Labels["app"])
	}

	if pod.Labels["runner-id"] != "runner-123" {
		t.Errorf("Expected label runner-id='runner-123', got '%s'", pod.Labels["runner-id"])
	}

	// Test annotations
	if pod.Annotations["grad.io/runner-name"] != "test-runner" {
		t.Errorf("Expected annotation grad.io/runner-name='test-runner', got '%s'", pod.Annotations["grad.io/runner-name"])
	}

	// Test container spec
	if len(pod.Spec.Containers) != 1 {
		t.Fatalf("Expected 1 container, got %d", len(pod.Spec.Containers))
	}

	container := pod.Spec.Containers[0]
	if container.Name != "runner" {
		t.Errorf("Expected container name 'runner', got '%s'", container.Name)
	}

	if container.Image != "ghcr.io/strrl/grad-runner:latest" {
		t.Errorf("Expected container image 'ghcr.io/strrl/grad-runner:latest', got '%s'", container.Image)
	}

	// Test environment variables
	envMap := make(map[string]string)
	for _, env := range container.Env {
		envMap[env.Name] = env.Value
	}

	if envMap["RUNNER_ID"] != "runner-123" {
		t.Errorf("Expected env RUNNER_ID='runner-123', got '%s'", envMap["RUNNER_ID"])
	}

	if envMap["RUNNER_NAME"] != "test-runner" {
		t.Errorf("Expected env RUNNER_NAME='test-runner', got '%s'", envMap["RUNNER_NAME"])
	}

	if envMap["TEST"] != "value" {
		t.Errorf("Expected env TEST='value', got '%s'", envMap["TEST"])
	}

	// Test resource requirements (basic check)
	if container.Resources.Requests == nil {
		t.Error("Expected resource requests to be set")
	}

	if container.Resources.Limits == nil {
		t.Error("Expected resource limits to be set")
	}
}

func TestMapPodStatusToRunnerStatus(t *testing.T) {
	tests := []struct {
		name           string
		podPhase       corev1.PodPhase
		conditions     []corev1.PodCondition
		expectedStatus RunnerStatus
	}{
		{
			name:           "Pending pod",
			podPhase:       corev1.PodPending,
			conditions:     []corev1.PodCondition{},
			expectedStatus: RunnerStatusCreating,
		},
		{
			name:     "Running pod with ready condition",
			podPhase: corev1.PodRunning,
			conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
			expectedStatus: RunnerStatusRunning,
		},
		{
			name:     "Running pod without ready condition",
			podPhase: corev1.PodRunning,
			conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionFalse,
				},
			},
			expectedStatus: RunnerStatusCreating,
		},
		{
			name:           "Succeeded pod",
			podPhase:       corev1.PodSucceeded,
			conditions:     []corev1.PodCondition{},
			expectedStatus: RunnerStatusStopped,
		},
		{
			name:           "Failed pod",
			podPhase:       corev1.PodFailed,
			conditions:     []corev1.PodCondition{},
			expectedStatus: RunnerStatusError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &corev1.Pod{
				Status: corev1.PodStatus{
					Phase:      tt.podPhase,
					Conditions: tt.conditions,
				},
			}

			status := MapPodStatusToRunnerStatus(pod)
			if status != tt.expectedStatus {
				t.Errorf("Expected status %v, got %v", tt.expectedStatus, status)
			}
		})
	}
}

func TestExtractPodInfo(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"runner-id": "test-runner-123",
			},
			Annotations: map[string]string{
				"grad.io/runner-name": "my-test-runner",
			},
		},
		Status: corev1.PodStatus{
			PodIP: "192.168.1.100",
		},
	}

	runnerID, runnerName, ipAddress := ExtractPodInfo(pod)

	if runnerID != "test-runner-123" {
		t.Errorf("Expected runner ID 'test-runner-123', got '%s'", runnerID)
	}

	if runnerName != "my-test-runner" {
		t.Errorf("Expected runner name 'my-test-runner', got '%s'", runnerName)
	}

	if ipAddress != "192.168.1.100" {
		t.Errorf("Expected IP address '192.168.1.100', got '%s'", ipAddress)
	}
}

func TestBuildPodDeletionRequest(t *testing.T) {
	config := &KubernetesConfig{
		Namespace: "test-namespace",
	}

	req := BuildPodDeletionRequest("runner-456", config)

	if req.PodName != "grad-runner-runner-456" {
		t.Errorf("Expected pod name 'grad-runner-runner-456', got '%s'", req.PodName)
	}

	if req.Namespace != "test-namespace" {
		t.Errorf("Expected namespace 'test-namespace', got '%s'", req.Namespace)
	}

	if req.RunnerID != "runner-456" {
		t.Errorf("Expected runner ID 'runner-456', got '%s'", req.RunnerID)
	}
}

func TestDefaultRunnerImageConstant(t *testing.T) {
	// Test that the default configuration uses the well-known runner image
	config := DefaultKubernetesConfig()

	if config.RunnerImage != DefaultRunnerImage {
		t.Errorf("Expected default runner image to be '%s', got '%s'", DefaultRunnerImage, config.RunnerImage)
	}

	if config.RunnerImage != "ghcr.io/strrl/grad-runner:latest" {
		t.Errorf("Expected DefaultRunnerImage constant to be 'ghcr.io/strrl/grad-runner:latest', got '%s'", config.RunnerImage)
	}
}

func TestRunnerSpecPresets(t *testing.T) {
	// Test Small preset
	small := RunnerSpecPreset.Small
	if small.CPU != "2000m" {
		t.Errorf("Expected RunnerSpecPreset.Small.CPU to be '2000m', got '%s'", small.CPU)
	}
	if small.Memory != "2Gi" {
		t.Errorf("Expected RunnerSpecPreset.Small.Memory to be '2Gi', got '%s'", small.Memory)
	}
	if small.Storage != "40Gi" {
		t.Errorf("Expected RunnerSpecPreset.Small.Storage to be '40Gi', got '%s'", small.Storage)
	}
	if small.CPUMillicores != 2000 {
		t.Errorf("Expected RunnerSpecPreset.Small.CPUMillicores to be 2000, got %d", small.CPUMillicores)
	}
	if small.MemoryMB != 2048 {
		t.Errorf("Expected RunnerSpecPreset.Small.MemoryMB to be 2048, got %d", small.MemoryMB)
	}
	if small.StorageGB != 40 {
		t.Errorf("Expected RunnerSpecPreset.Small.StorageGB to be 40, got %d", small.StorageGB)
	}

	// Test Medium preset (future)
	medium := RunnerSpecPreset.Medium
	if medium.CPU != "4000m" {
		t.Errorf("Expected RunnerSpecPreset.Medium.CPU to be '4000m', got '%s'", medium.CPU)
	}
	if medium.CPUMillicores != 4000 {
		t.Errorf("Expected RunnerSpecPreset.Medium.CPUMillicores to be 4000, got %d", medium.CPUMillicores)
	}

	// Test Large preset (future)
	large := RunnerSpecPreset.Large
	if large.CPU != "8000m" {
		t.Errorf("Expected RunnerSpecPreset.Large.CPU to be '8000m', got '%s'", large.CPU)
	}
	if large.CPUMillicores != 8000 {
		t.Errorf("Expected RunnerSpecPreset.Large.CPUMillicores to be 8000, got %d", large.CPUMillicores)
	}
}

func TestRunnerImageEnvironmentOverride(t *testing.T) {
	// Test that RUNNER_IMAGE environment variable overrides the default
	originalEnv := os.Getenv("RUNNER_IMAGE")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("RUNNER_IMAGE")
		} else {
			os.Setenv("RUNNER_IMAGE", originalEnv)
		}
	}()

	// Set dynamic tag that skaffold would generate
	dynamicTag := "ghcr.io/strrl/grad-runner:v1.17.1-38-g1c6517887"
	os.Setenv("RUNNER_IMAGE", dynamicTag)

	config := LoadConfig()

	if config.Kubernetes.RunnerImage != dynamicTag {
		t.Errorf("Expected runner image to be overridden to '%s', got '%s'", dynamicTag, config.Kubernetes.RunnerImage)
	}

	// Test default behavior when env var is not set
	os.Unsetenv("RUNNER_IMAGE")
	config = LoadConfig()

	if config.Kubernetes.RunnerImage != DefaultRunnerImage {
		t.Errorf("Expected default runner image '%s', got '%s'", DefaultRunnerImage, config.Kubernetes.RunnerImage)
	}
}
