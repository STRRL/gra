package grpc

import (
	"context"
	"fmt"
	"sync"
	"time"

	gradv1 "github.com/strrl/gra/gen/grad/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server implements the gRPC RunnerService
type Server struct {
	gradv1.UnimplementedRunnerServiceServer
	
	// In-memory storage for demo purposes
	// In production, this would be replaced with proper storage/K8s API
	runners map[string]*gradv1.Runner
	mu      sync.RWMutex
	
	// Runner ID counter
	runnerIDCounter int64
}

// NewServer creates a new gRPC server instance
func NewServer() *Server {
	return &Server{
		runners: make(map[string]*gradv1.Runner),
	}
}

// CreateRunner creates a new runner instance
func (s *Server) CreateRunner(ctx context.Context, req *gradv1.CreateRunnerRequest) (*gradv1.CreateRunnerResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Generate runner ID
	s.runnerIDCounter++
	runnerID := fmt.Sprintf("runner-%d", s.runnerIDCounter)
	
	// Use provided name or generate one
	name := req.Name
	if name == "" {
		name = runnerID
	}
	
	// Set default resources if not provided
	resources := req.Resources
	if resources == nil {
		resources = &gradv1.ResourceRequirements{
			CpuMillicores: 1000, // 1 CPU
			MemoryMb:      2048, // 2GB RAM
			StorageGb:     10,   // 10GB storage
		}
	}
	
	// Create runner
	runner := &gradv1.Runner{
		Id:        runnerID,
		Name:      name,
		Status:    gradv1.RunnerStatus_RUNNER_STATUS_CREATING,
		Resources: resources,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		Ssh: &gradv1.SSHDetails{
			Host:     "localhost", // This would be the actual pod IP in K8s
			Port:     22,
			Username: "runner",
		},
		IpAddress: "127.0.0.1", // This would be the actual pod IP in K8s
		Env:       req.Env,
	}
	
	// Store runner
	s.runners[runnerID] = runner
	
	// Simulate async creation process
	go func() {
		time.Sleep(2 * time.Second) // Simulate creation time
		s.mu.Lock()
		if r, exists := s.runners[runnerID]; exists {
			r.Status = gradv1.RunnerStatus_RUNNER_STATUS_RUNNING
			r.UpdatedAt = time.Now().Unix()
		}
		s.mu.Unlock()
	}()
	
	return &gradv1.CreateRunnerResponse{
		Runner: runner,
	}, nil
}

// DeleteRunner removes a runner instance
func (s *Server) DeleteRunner(ctx context.Context, req *gradv1.DeleteRunnerRequest) (*gradv1.DeleteRunnerResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	runner, exists := s.runners[req.RunnerId]
	if !exists {
		return nil, status.Errorf(codes.NotFound, "runner %s not found", req.RunnerId)
	}
	
	// Update status to stopping
	runner.Status = gradv1.RunnerStatus_RUNNER_STATUS_STOPPING
	runner.UpdatedAt = time.Now().Unix()
	
	// Simulate async deletion process
	go func() {
		time.Sleep(1 * time.Second) // Simulate deletion time
		s.mu.Lock()
		delete(s.runners, req.RunnerId)
		s.mu.Unlock()
	}()
	
	return &gradv1.DeleteRunnerResponse{
		Message: fmt.Sprintf("runner %s deletion initiated", req.RunnerId),
	}, nil
}

// ListRunners returns all available runners
func (s *Server) ListRunners(ctx context.Context, req *gradv1.ListRunnersRequest) (*gradv1.ListRunnersResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var runners []*gradv1.Runner
	
	// Filter by status if specified
	for _, runner := range s.runners {
		if req.Status != gradv1.RunnerStatus_RUNNER_STATUS_UNSPECIFIED {
			if runner.Status != req.Status {
				continue
			}
		}
		runners = append(runners, runner)
	}
	
	// Apply pagination
	total := int32(len(runners))
	offset := req.Offset
	limit := req.Limit
	
	if limit == 0 {
		limit = 50 // Default limit
	}
	
	if offset >= total {
		runners = []*gradv1.Runner{}
	} else {
		end := offset + limit
		if end > total {
			end = total
		}
		runners = runners[offset:end]
	}
	
	return &gradv1.ListRunnersResponse{
		Runners: runners,
		Total:   total,
	}, nil
}

// ExecuteCode executes code in a specific runner
func (s *Server) ExecuteCode(ctx context.Context, req *gradv1.ExecuteCodeRequest) (*gradv1.ExecuteCodeResponse, error) {
	s.mu.RLock()
	runner, exists := s.runners[req.RunnerId]
	s.mu.RUnlock()
	
	if !exists {
		return nil, status.Errorf(codes.NotFound, "runner %s not found", req.RunnerId)
	}
	
	if runner.Status != gradv1.RunnerStatus_RUNNER_STATUS_RUNNING {
		return nil, status.Errorf(codes.FailedPrecondition, "runner %s is not running (status: %s)", req.RunnerId, runner.Status.String())
	}
	
	// Simulate code execution
	startTime := time.Now()
	
	// For demo purposes, simulate different execution results based on code content
	var output, errorOutput string
	var exitCode int32
	
	if req.Code == "" {
		errorOutput = "error: no code provided"
		exitCode = 1
	} else if req.Code == "exit 1" {
		errorOutput = "process exited with code 1"
		exitCode = 1
	} else {
		// Simulate successful execution
		output = fmt.Sprintf("Code executed successfully in runner %s\nLanguage: %s\nWorking directory: %s\n", 
			req.RunnerId, req.Language, req.WorkingDir)
		if req.Language == "python" {
			output += "Python 3.11.0\n"
		}
		exitCode = 0
	}
	
	executionTime := time.Since(startTime)
	
	return &gradv1.ExecuteCodeResponse{
		Output:     output,
		Error:      errorOutput,
		ExitCode:   exitCode,
		DurationMs: executionTime.Nanoseconds() / 1000000, // Convert to milliseconds
	}, nil
}

// GetRunner returns details about a specific runner
func (s *Server) GetRunner(ctx context.Context, req *gradv1.GetRunnerRequest) (*gradv1.GetRunnerResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	runner, exists := s.runners[req.RunnerId]
	if !exists {
		return nil, status.Errorf(codes.NotFound, "runner %s not found", req.RunnerId)
	}
	
	return &gradv1.GetRunnerResponse{
		Runner: runner,
	}, nil
}