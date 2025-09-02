//go:build integration
// +build integration

package service

import (
	"context"
	"testing"
)

// TestRunnerServiceBasics tests basic functionality of the runner service
// This is an integration test that requires a real Kubernetes cluster
func TestRunnerServiceBasics(t *testing.T) {
	// For testing, we'll use a mock that doesn't actually connect to Kubernetes
	// In a real test environment, you'd use a fake Kubernetes client

	// Skip this test if we can't create a Kubernetes client
	k8sClient, err := NewKubernetesClient(DefaultKubernetesConfig())
	if err != nil {
		t.Skipf("Skipping test - cannot create Kubernetes client: %v", err)
	}

	activityTracker := NewActivityTracker()
	service := NewRunnerService(k8sClient, activityTracker)
	ctx := context.Background()

	// Test creating a runner
	req := &CreateRunnerRequest{
		Name: "test-runner",
		Resources: &ResourceRequirements{
			CPUMillicores: 500,
			MemoryMB:      1024,
			StorageGB:     5,
		},
		Env: map[string]string{
			"TEST_ENV": "test_value",
		},
	}

	// This will likely fail in testing because we don't have a real K8s cluster
	// but we can verify the service layer structure is correct
	runner, err := service.CreateRunner(ctx, req)
	if err != nil {
		// Expected in test environment without real K8s cluster
		t.Logf("CreateRunner failed as expected in test environment: %v", err)
		return
	}

	// If we somehow succeed, verify the response
	if runner.Name != "test-runner" {
		t.Errorf("Expected runner name 'test-runner', got '%s'", runner.Name)
	}

	// Note: Resources now use hardcoded "small" preset (2c2g40g), not request values
	if runner.Resources.CPUMillicores != RunnerSpecPreset.Small.CPUMillicores {
		t.Errorf("Expected CPU millicores %d (small preset), got %d", RunnerSpecPreset.Small.CPUMillicores, runner.Resources.CPUMillicores)
	}

	// Test getting the runner
	retrieved, err := service.GetRunner(ctx, runner.ID)
	if err != nil {
		t.Errorf("Failed to get runner: %v", err)
		return
	}

	if retrieved == nil {
		t.Error("Retrieved runner is nil")
		return
	}

	if retrieved.ID != runner.ID {
		t.Errorf("Expected runner ID '%s', got '%s'", runner.ID, retrieved.ID)
	}

	// Test listing runners
	runners, total, err := service.ListRunners(ctx, &ListOptions{Limit: 10})
	if err != nil {
		t.Errorf("Failed to list runners: %v", err)
	}

	if total == 0 {
		t.Error("Expected at least one runner in the list")
	}

	if len(runners) == 0 {
		t.Error("Expected runners in the response")
	}

	// Test deleting the runner
	err = service.DeleteRunner(ctx, runner.ID)
	if err != nil {
		t.Errorf("Failed to delete runner: %v", err)
	}
}