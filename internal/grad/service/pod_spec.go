package service

import (
	"fmt"
	"time"

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
	S3FSImage     string
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
		S3FSImage:  config.S3FSImage,
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
	// Build environment variables for main container
	mainEnv := []corev1.EnvVar{
		{
			Name:  "RUNNER_ID",
			Value: req.RunnerID,
		},
		{
			Name:  "RUNNER_NAME",
			Value: req.RunnerName,
		},
	}

	// Add custom environment variables to main container
	for key, value := range req.Env {
		mainEnv = append(mainEnv, corev1.EnvVar{
			Name:  key,
			Value: value,
		})
	}

	// Build environment variables for S3FS sidecar
	s3fsEnv := []corev1.EnvVar{
		{
			Name:  "RUNNER_ID",
			Value: req.RunnerID,
		},
		{
			Name:  "RUNNER_NAME",
			Value: req.RunnerName,
		},
	}

	// Add S3 configuration if present in custom environment variables
	for key, value := range req.Env {
		if key == "AWS_ACCESS_KEY_ID" || key == "AWS_SECRET_ACCESS_KEY" || 
		   key == "S3_BUCKET" || key == "S3_ENDPOINT" {
			s3fsEnv = append(s3fsEnv, corev1.EnvVar{
				Name:  key,
				Value: value,
			})
		}
	}

	// Create shared volume for workspace
	workspaceVolume := corev1.Volume{
		Name: "workspace",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.PodName,
			Namespace: req.Namespace,
			Labels: map[string]string{
				"app":                             "grad-runner",
				"app.kubernetes.io/managed-by":    "grad",
				"app.kubernetes.io/component":     "runner",
				"app.kubernetes.io/name":          "grad-runner",
				"app.kubernetes.io/instance":      req.RunnerID,
				"type":      "runner",
				"runner-id": req.RunnerID,
			},
			Annotations: map[string]string{
				"grad.io/runner-id":     req.RunnerID,
				"grad.io/runner-name":   req.RunnerName,
				"grad.io/status":        "creating",
				"grad.io/created-at":    time.Now().Format(time.RFC3339),
			},
			Finalizers: []string{
				"grad.io/runner-finalizer",
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Volumes: []corev1.Volume{workspaceVolume},
			// S3FS sidecar as init container with OnFailure restart policy
			InitContainers: []corev1.Container{
				{
					Name:          "s3fs-sidecar",
					Image:         req.S3FSImage,
					RestartPolicy: func() *corev1.ContainerRestartPolicy { 
						policy := corev1.ContainerRestartPolicyAlways
						return &policy
					}(),
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("50m"),
							corev1.ResourceMemory: resource.MustParse("64Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
					},
					Env: s3fsEnv,
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "workspace",
							MountPath: "/workspace",
						},
					},
					SecurityContext: &corev1.SecurityContext{
						Privileged: &[]bool{true}[0],
					},
				},
			},
			// Main runner container
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
					Env: mainEnv,
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "workspace",
							MountPath: "/workspace",
						},
					},
					Command: []string{"/usr/local/bin/entrypoint.sh"},
					Args:    []string{"sleep", "infinity"},
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
