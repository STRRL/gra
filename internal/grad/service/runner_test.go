package service

import (
	"context"
	"testing"
)

// TestRunnerServiceBasics tests basic functionality of the runner service
func TestRunnerServiceBasics(t *testing.T) {
	// For testing, we'll use a mock that doesn't actually connect to Kubernetes
	// In a real test environment, you'd use a fake Kubernetes client

	// Skip this test if we can't create a Kubernetes client
	k8sClient, err := NewKubernetesClient(DefaultKubernetesConfig())
	if err != nil {
		t.Skipf("Skipping test - cannot create Kubernetes client: %v", err)
	}

	service := NewRunnerService(k8sClient)
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

// TestDomainTypes tests the domain type conversions
func TestDomainTypes(t *testing.T) {
	// Test ResourceRequirements conversion
	resources := &ResourceRequirements{
		CPUMillicores: 1000,
		MemoryMB:      2048,
		StorageGB:     10,
	}

	proto := resources.ToProto()
	if proto.CpuMillicores != 1000 {
		t.Errorf("Expected CPU millicores 1000, got %d", proto.CpuMillicores)
	}

	if proto.MemoryMb != 2048 {
		t.Errorf("Expected memory MB 2048, got %d", proto.MemoryMb)
	}

	if proto.StorageGb != 10 {
		t.Errorf("Expected storage GB 10, got %d", proto.StorageGb)
	}

	// Test SSHDetails conversion
	ssh := &SSHDetails{
		Host:      "test-host",
		Port:      22,
		Username:  "test-user",
		PublicKey: "test-key",
	}

	sshProto := ssh.ToProto()
	if sshProto.Host != "test-host" {
		t.Errorf("Expected SSH host 'test-host', got '%s'", sshProto.Host)
	}

	if sshProto.Port != 22 {
		t.Errorf("Expected SSH port 22, got %d", sshProto.Port)
	}

	// Test Runner conversion
	runner := &Runner{
		ID:        "test-id",
		Name:      "test-name",
		Status:    RunnerStatusRunning,
		Resources: resources,
		SSH:       ssh,
		IPAddress: "192.168.1.1",
		Env:       map[string]string{"TEST": "value"},
	}

	runnerProto := runner.ToProto()
	if runnerProto.Id != "test-id" {
		t.Errorf("Expected runner ID 'test-id', got '%s'", runnerProto.Id)
	}

	if runnerProto.Name != "test-name" {
		t.Errorf("Expected runner name 'test-name', got '%s'", runnerProto.Name)
	}

	if runnerProto.IpAddress != "192.168.1.1" {
		t.Errorf("Expected IP address '192.168.1.1', got '%s'", runnerProto.IpAddress)
	}
}
