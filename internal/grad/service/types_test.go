package service

import (
	"testing"

	gradv1 "github.com/strrl/gra/gen/grad/v1"
)

func TestResourceRequirementsToProto(t *testing.T) {
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
}

func TestSSHDetailsToProto(t *testing.T) {
	ssh := &SSHDetails{
		Host:      "test-host",
		Port:      22,
		Username:  "test-user",
		PublicKey: "test-key",
	}

	proto := ssh.ToProto()

	if proto.Host != "test-host" {
		t.Errorf("Expected SSH host 'test-host', got '%s'", proto.Host)
	}

	if proto.Port != 22 {
		t.Errorf("Expected SSH port 22, got %d", proto.Port)
	}

	if proto.Username != "test-user" {
		t.Errorf("Expected SSH username 'test-user', got '%s'", proto.Username)
	}

	if proto.PublicKey != "test-key" {
		t.Errorf("Expected SSH public key 'test-key', got '%s'", proto.PublicKey)
	}
}

func TestRunnerToProto(t *testing.T) {
	runner := &Runner{
		ID:     "test-id",
		Name:   "test-name",
		Status: RunnerStatusRunning,
		Resources: &ResourceRequirements{
			CPUMillicores: 500,
			MemoryMB:      1024,
			StorageGB:     5,
		},
		CreatedAt: 1234567890,
		UpdatedAt: 1234567891,
		SSH: &SSHDetails{
			Host:     "192.168.1.1",
			Port:     22,
			Username: "runner",
		},
		IPAddress: "192.168.1.1",
		Env: map[string]string{
			"TEST": "value",
		},
	}

	proto := runner.ToProto()

	if proto.Id != "test-id" {
		t.Errorf("Expected runner ID 'test-id', got '%s'", proto.Id)
	}

	if proto.Name != "test-name" {
		t.Errorf("Expected runner name 'test-name', got '%s'", proto.Name)
	}

	if proto.Status != gradv1.RunnerStatus_RUNNER_STATUS_RUNNING {
		t.Errorf("Expected status RUNNING, got %v", proto.Status)
	}

	if proto.IpAddress != "192.168.1.1" {
		t.Errorf("Expected IP address '192.168.1.1', got '%s'", proto.IpAddress)
	}

	if proto.CreatedAt != 1234567890 {
		t.Errorf("Expected created at 1234567890, got %d", proto.CreatedAt)
	}

	if proto.UpdatedAt != 1234567891 {
		t.Errorf("Expected updated at 1234567891, got %d", proto.UpdatedAt)
	}

	if proto.Env["TEST"] != "value" {
		t.Errorf("Expected env TEST='value', got '%s'", proto.Env["TEST"])
	}

	// Test nested conversions
	if proto.Resources.CpuMillicores != 500 {
		t.Errorf("Expected CPU millicores 500, got %d", proto.Resources.CpuMillicores)
	}

	if proto.Ssh.Host != "192.168.1.1" {
		t.Errorf("Expected SSH host '192.168.1.1', got '%s'", proto.Ssh.Host)
	}
}

func TestFromProtoCreateRunnerRequest(t *testing.T) {
	protoReq := &gradv1.CreateRunnerRequest{
		Name: "test-runner",
		Env: map[string]string{
			"ENV_VAR": "env_value",
		},
	}

	domainReq := FromProtoCreateRunnerRequest(protoReq)

	if domainReq.Name != "test-runner" {
		t.Errorf("Expected name 'test-runner', got '%s'", domainReq.Name)
	}

	// Resources should be nil since they're no longer in the proto request
	if domainReq.Resources != nil {
		t.Errorf("Expected resources to be nil (will use preset), got %+v", domainReq.Resources)
	}

	if domainReq.Env["ENV_VAR"] != "env_value" {
		t.Errorf("Expected env ENV_VAR='env_value', got '%s'", domainReq.Env["ENV_VAR"])
	}
}

func TestFromProtoExecuteCodeRequest(t *testing.T) {
	protoReq := &gradv1.ExecuteCodeRequest{
		RunnerId:   "runner-123",
		Code:       "print('hello')",
		Language:   "python",
		Timeout:    30,
		WorkingDir: "/tmp",
	}

	domainReq := FromProtoExecuteCodeRequest(protoReq)

	if domainReq.RunnerID != "runner-123" {
		t.Errorf("Expected runner ID 'runner-123', got '%s'", domainReq.RunnerID)
	}

	if domainReq.Code != "print('hello')" {
		t.Errorf("Expected code 'print('hello')', got '%s'", domainReq.Code)
	}

	if domainReq.Language != "python" {
		t.Errorf("Expected language 'python', got '%s'", domainReq.Language)
	}

	if domainReq.Timeout != 30 {
		t.Errorf("Expected timeout 30, got %d", domainReq.Timeout)
	}

	if domainReq.WorkingDir != "/tmp" {
		t.Errorf("Expected working dir '/tmp', got '%s'", domainReq.WorkingDir)
	}
}

func TestExecuteCodeResultToProto(t *testing.T) {
	result := &ExecuteCodeResult{
		Output:     "Hello, World!",
		Error:      "",
		ExitCode:   0,
		DurationMS: 150,
	}

	proto := result.ToProto()

	if proto.Output != "Hello, World!" {
		t.Errorf("Expected output 'Hello, World!', got '%s'", proto.Output)
	}

	if proto.Error != "" {
		t.Errorf("Expected empty error, got '%s'", proto.Error)
	}

	if proto.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", proto.ExitCode)
	}

	if proto.DurationMs != 150 {
		t.Errorf("Expected duration 150ms, got %d", proto.DurationMs)
	}
}

func TestFromProtoListOptions(t *testing.T) {
	opts := FromProtoListOptions(gradv1.RunnerStatus_RUNNER_STATUS_RUNNING, 20, 10)

	if opts.Status != RunnerStatusRunning {
		t.Errorf("Expected status RUNNING, got %v", opts.Status)
	}

	if opts.Limit != 20 {
		t.Errorf("Expected limit 20, got %d", opts.Limit)
	}

	if opts.Offset != 10 {
		t.Errorf("Expected offset 10, got %d", opts.Offset)
	}
}

func TestNilHandling(t *testing.T) {
	// Test that nil resource requirements are handled properly
	var nilResources *ResourceRequirements = nil
	proto := nilResources.ToProto()
	if proto != nil {
		t.Error("Expected nil proto for nil resources")
	}

	// Test that nil SSH details are handled properly
	var nilSSH *SSHDetails = nil
	sshProto := nilSSH.ToProto()
	if sshProto != nil {
		t.Error("Expected nil proto for nil SSH details")
	}

	// Test conversion from proto (resources no longer in request)
	protoReq := &gradv1.CreateRunnerRequest{
		Name: "test",
		Env:  map[string]string{},
	}

	domainReq := FromProtoCreateRunnerRequest(protoReq)
	if domainReq.Resources != nil {
		t.Error("Expected nil resources since they're no longer in proto request")
	}
}
