package service

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodCreationRequest represents a request to create a pod
type PodCreationRequest struct {
	PodName       string
	Namespace     string
	RunnerID      string
	RunnerName    string
	Image         string
	CPURequest    string
	MemoryRequest string
	SSHPort       int32
	Env           map[string]string
}

// PodDeletionRequest represents a request to delete a pod
type PodDeletionRequest struct {
	PodName   string
	Namespace string
	RunnerID  string
}

// BuildPodCreationRequest creates a pod creation request from a runner
func BuildPodCreationRequest(runner *Runner, config *KubernetesConfig) *PodCreationRequest {
	podName := fmt.Sprintf("grad-runner-%s", runner.ID)

	// Use hardcoded "small" preset configuration: 2c2g40g
	return &PodCreationRequest{
		PodName:    podName,
		Namespace:  config.Namespace,
		RunnerID:   runner.ID,
		RunnerName: runner.Name,
		Image:      config.RunnerImage,
		// Small preset: 2000m (2 cores)
		CPURequest: config.DefaultCPU,
		// Small preset: 2Gi
		MemoryRequest: config.DefaultMemory,
		SSHPort:       config.SSHPort,
		Env:           runner.Env,
	}
}

// BuildPodDeletionRequest creates a pod deletion request from a runner ID
func BuildPodDeletionRequest(runnerID string, config *KubernetesConfig) *PodDeletionRequest {
	podName := fmt.Sprintf("grad-runner-%s", runnerID)

	return &PodDeletionRequest{
		PodName:   podName,
		Namespace: config.Namespace,
		RunnerID:  runnerID,
	}
}

// ToPodSpec converts a PodCreationRequest to a Kubernetes Pod specification
func (req *PodCreationRequest) ToPodSpec() *corev1.Pod {
	// Build environment variables
	env := []corev1.EnvVar{
		{
			Name:  "RUNNER_ID",
			Value: req.RunnerID,
		},
		{
			Name:  "RUNNER_NAME",
			Value: req.RunnerName,
		},
	}

	// Add custom environment variables
	for key, value := range req.Env {
		env = append(env, corev1.EnvVar{
			Name:  key,
			Value: value,
		})
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.PodName,
			Namespace: req.Namespace,
			Labels: map[string]string{
				"app":       "grad-runner",
				"type":      "runner",
				"runner-id": req.RunnerID,
			},
			Annotations: map[string]string{
				"grad.io/runner-name": req.RunnerName,
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:  "runner",
					Image: req.Image,
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: req.SSHPort,
							Name:          "ssh",
							Protocol:      corev1.ProtocolTCP,
						},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse(req.CPURequest),
							corev1.ResourceMemory: resource.MustParse(req.MemoryRequest),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse(req.CPURequest),
							corev1.ResourceMemory: resource.MustParse(req.MemoryRequest),
						},
					},
					Env: env,
					// This will be enhanced with SSH setup and development tools
					Command: []string{"/bin/bash"},
					Args:    []string{"-c", "while true; do sleep 30; done"},
				},
			},
		},
	}
}

// MapPodStatusToRunnerStatus maps Kubernetes pod status to runner status (pure function)
func MapPodStatusToRunnerStatus(pod *corev1.Pod) RunnerStatus {
	switch pod.Status.Phase {
	case corev1.PodPending:
		return RunnerStatusCreating
	case corev1.PodRunning:
		// Check if all containers are ready
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
				return RunnerStatusRunning
			}
		}
		return RunnerStatusCreating
	case corev1.PodSucceeded:
		return RunnerStatusStopped
	case corev1.PodFailed:
		return RunnerStatusError
	default:
		return RunnerStatusError
	}
}

// ExtractPodInfo extracts runner information from a pod (pure function)
func ExtractPodInfo(pod *corev1.Pod) (runnerID, runnerName, ipAddress string) {
	runnerID = pod.Labels["runner-id"]
	runnerName = pod.Annotations["grad.io/runner-name"]
	ipAddress = pod.Status.PodIP

	return runnerID, runnerName, ipAddress
}
