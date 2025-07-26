package service

import (
	"context"
	"errors"

	gradv1 "github.com/strrl/gra/gen/grad/v1"
)

// Domain errors
var (
	ErrRunnerNotFound   = errors.New("runner not found")
	ErrRunnerNotRunning = errors.New("runner is not running")
	ErrInvalidRequest   = errors.New("invalid request")
	ErrKubernetesAPI    = errors.New("kubernetes API error")
	ErrCodeExecution    = errors.New("code execution failed")
	ErrResourceConflict = errors.New("resource conflict")
)

// CreateRunnerRequest represents the domain request to create a runner
type CreateRunnerRequest struct {
	Name      string
	Resources *ResourceRequirements
	Env       map[string]string
}

// ResourceRequirements represents resource allocation for a runner
type ResourceRequirements struct {
	CPUMillicores int32
	MemoryMB      int32
	StorageGB     int32
}

// Runner represents a runner instance in the domain
type Runner struct {
	ID        string
	Name      string
	Status    RunnerStatus
	Resources *ResourceRequirements
	CreatedAt int64
	UpdatedAt int64
	SSH       *SSHDetails
	IPAddress string
	Env       map[string]string
}

// RunnerStatus represents the status of a runner
type RunnerStatus int

const (
	RunnerStatusUnspecified RunnerStatus = iota
	RunnerStatusCreating
	RunnerStatusRunning
	RunnerStatusStopping
	RunnerStatusStopped
	RunnerStatusError
)

// SSHDetails contains SSH connection information
type SSHDetails struct {
	Host      string
	Port      int32
	Username  string
	PublicKey string
}

// ExecuteCodeRequest represents a code execution request
type ExecuteCodeRequest struct {
	RunnerID   string
	Code       string
	Language   string
	Timeout    int32
	WorkingDir string
}

// ExecuteCodeResult represents the result of code execution
type ExecuteCodeResult struct {
	Output     string
	Error      string
	ExitCode   int32
	DurationMS int64
}

// ListOptions represents options for listing runners
type ListOptions struct {
	Status RunnerStatus
	Limit  int32
	Offset int32
}

// RunnerService defines the interface for runner management
type RunnerService interface {
	CreateRunner(ctx context.Context, req *CreateRunnerRequest) (*Runner, error)
	DeleteRunner(ctx context.Context, runnerID string) error
	ListRunners(ctx context.Context, opts *ListOptions) ([]*Runner, int32, error)
	GetRunner(ctx context.Context, runnerID string) (*Runner, error)
	ExecuteCode(ctx context.Context, req *ExecuteCodeRequest) (*ExecuteCodeResult, error)
}

// Conversion functions between domain and proto types

// ToProtoRunner converts domain Runner to proto Runner
func (r *Runner) ToProto() *gradv1.Runner {
	return &gradv1.Runner{
		Id:        r.ID,
		Name:      r.Name,
		Status:    gradv1.RunnerStatus(r.Status),
		Resources: r.Resources.ToProto(),
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
		Ssh:       r.SSH.ToProto(),
		IpAddress: r.IPAddress,
		Env:       r.Env,
	}
}

// ToProto converts domain ResourceRequirements to proto ResourceRequirements
func (rr *ResourceRequirements) ToProto() *gradv1.ResourceRequirements {
	if rr == nil {
		return nil
	}
	return &gradv1.ResourceRequirements{
		CpuMillicores: rr.CPUMillicores,
		MemoryMb:      rr.MemoryMB,
		StorageGb:     rr.StorageGB,
	}
}

// ToProto converts domain SSHDetails to proto SSHDetails
func (ssh *SSHDetails) ToProto() *gradv1.SSHDetails {
	if ssh == nil {
		return nil
	}
	return &gradv1.SSHDetails{
		Host:      ssh.Host,
		Port:      ssh.Port,
		Username:  ssh.Username,
		PublicKey: ssh.PublicKey,
	}
}

// FromProtoCreateRunnerRequest converts proto request to domain request
func FromProtoCreateRunnerRequest(req *gradv1.CreateRunnerRequest) *CreateRunnerRequest {
	return &CreateRunnerRequest{
		Name:      req.Name,
		Resources: nil, // Resources are no longer in the request - will use preset
		Env:       req.Env,
	}
}

// FromProtoResourceRequirements converts proto ResourceRequirements to domain
func FromProtoResourceRequirements(rr *gradv1.ResourceRequirements) *ResourceRequirements {
	if rr == nil {
		return nil
	}
	return &ResourceRequirements{
		CPUMillicores: rr.CpuMillicores,
		MemoryMB:      rr.MemoryMb,
		StorageGB:     rr.StorageGb,
	}
}

// FromProtoExecuteCodeRequest converts proto request to domain request
func FromProtoExecuteCodeRequest(req *gradv1.ExecuteCodeRequest) *ExecuteCodeRequest {
	return &ExecuteCodeRequest{
		RunnerID:   req.RunnerId,
		Code:       req.Code,
		Language:   req.Language,
		Timeout:    req.Timeout,
		WorkingDir: req.WorkingDir,
	}
}

// ToProtoExecuteCodeResponse converts domain result to proto response
func (ecr *ExecuteCodeResult) ToProto() *gradv1.ExecuteCodeResponse {
	return &gradv1.ExecuteCodeResponse{
		Output:     ecr.Output,
		Error:      ecr.Error,
		ExitCode:   ecr.ExitCode,
		DurationMs: ecr.DurationMS,
	}
}

// FromProtoListOptions converts proto list options to domain
func FromProtoListOptions(status gradv1.RunnerStatus, limit, offset int32) *ListOptions {
	return &ListOptions{
		Status: RunnerStatus(status),
		Limit:  limit,
		Offset: offset,
	}
}
