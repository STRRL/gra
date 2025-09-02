package service

import (
	"testing"
)

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
