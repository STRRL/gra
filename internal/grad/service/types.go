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
	ErrCommandExecution = errors.New("command execution failed")
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
type RunnerStatus string

const (
	RunnerStatusUnspecified RunnerStatus = ""
	RunnerStatusCreating    RunnerStatus = "creating"
	RunnerStatusRunning     RunnerStatus = "running"
	RunnerStatusStopping    RunnerStatus = "stopping"
	RunnerStatusStopped     RunnerStatus = "stopped"
	RunnerStatusError       RunnerStatus = "error"
)

// SSHDetails contains SSH connection information
type SSHDetails struct {
	Host      string
	Port      int32
	Username  string
	PublicKey string
}

// ExecuteCommandRequest represents a command execution request
type ExecuteCommandRequest struct {
	RunnerID   string
	Command    string
	Shell      string
	Timeout    int32
	WorkingDir string
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
	ExecuteCommandStream(ctx context.Context, req *ExecuteCommandRequest, stdoutCh, stderrCh chan<- []byte) (int32, error)
}

// Conversion functions between domain and proto types

// ToProtoRunner converts domain Runner to proto Runner
func (r *Runner) ToProto() *gradv1.Runner {
	return &gradv1.Runner{
		Id:        r.ID,
		Name:      r.Name,
		Status:    r.Status.ToProto(),
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

// FromProtoExecuteCommandRequest converts proto request to domain request
func FromProtoExecuteCommandRequest(req *gradv1.ExecuteCommandRequest) *ExecuteCommandRequest {
	return &ExecuteCommandRequest{
		RunnerID:   req.RunnerId,
		Command:    req.Command,
		Shell:      req.Shell,
		Timeout:    req.Timeout,
		WorkingDir: req.WorkingDir,
	}
}


// FromProtoListOptions converts proto list options to domain
func FromProtoListOptions(status gradv1.RunnerStatus, limit, offset int32) *ListOptions {
	return &ListOptions{
		Status: RunnerStatusFromProto(status),
		Limit:  limit,
		Offset: offset,
	}
}

// ToProto converts domain RunnerStatus to proto RunnerStatus
func (rs RunnerStatus) ToProto() gradv1.RunnerStatus {
	switch rs {
	case RunnerStatusUnspecified:
		return gradv1.RunnerStatus_RUNNER_STATUS_UNSPECIFIED
	case RunnerStatusCreating:
		return gradv1.RunnerStatus_RUNNER_STATUS_CREATING
	case RunnerStatusRunning:
		return gradv1.RunnerStatus_RUNNER_STATUS_RUNNING
	case RunnerStatusStopping:
		return gradv1.RunnerStatus_RUNNER_STATUS_STOPPING
	case RunnerStatusStopped:
		return gradv1.RunnerStatus_RUNNER_STATUS_STOPPED
	case RunnerStatusError:
		return gradv1.RunnerStatus_RUNNER_STATUS_ERROR
	default:
		return gradv1.RunnerStatus_RUNNER_STATUS_UNSPECIFIED
	}
}

// RunnerStatusFromProto converts proto RunnerStatus to domain RunnerStatus
func RunnerStatusFromProto(status gradv1.RunnerStatus) RunnerStatus {
	switch status {
	case gradv1.RunnerStatus_RUNNER_STATUS_UNSPECIFIED:
		return RunnerStatusUnspecified
	case gradv1.RunnerStatus_RUNNER_STATUS_CREATING:
		return RunnerStatusCreating
	case gradv1.RunnerStatus_RUNNER_STATUS_RUNNING:
		return RunnerStatusRunning
	case gradv1.RunnerStatus_RUNNER_STATUS_STOPPING:
		return RunnerStatusStopping
	case gradv1.RunnerStatus_RUNNER_STATUS_STOPPED:
		return RunnerStatusStopped
	case gradv1.RunnerStatus_RUNNER_STATUS_ERROR:
		return RunnerStatusError
	default:
		return RunnerStatusUnspecified
	}
}
