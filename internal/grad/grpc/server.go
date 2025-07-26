package grpc

import (
	"context"
	"errors"
	"fmt"

	gradv1 "github.com/strrl/gra/gen/grad/v1"
	"github.com/strrl/gra/internal/grad/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server implements the gRPC RunnerService as a thin controller
type Server struct {
	gradv1.UnimplementedRunnerServiceServer
	runnerService service.RunnerService
}

// NewServer creates a new gRPC server instance
func NewServer(runnerService service.RunnerService) *Server {
	return &Server{
		runnerService: runnerService,
	}
}

// CreateRunner creates a new runner instance
func (s *Server) CreateRunner(ctx context.Context, req *gradv1.CreateRunnerRequest) (*gradv1.CreateRunnerResponse, error) {
	// Validate request
	if err := s.validateCreateRunnerRequest(req); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	// Convert proto request to domain request
	domainReq := service.FromProtoCreateRunnerRequest(req)

	// Call service layer
	runner, err := s.runnerService.CreateRunner(ctx, domainReq)
	if err != nil {
		return nil, s.mapServiceError(err)
	}

	// Convert domain response to proto response
	return &gradv1.CreateRunnerResponse{
		Runner: runner.ToProto(),
	}, nil
}

// DeleteRunner removes a runner instance
func (s *Server) DeleteRunner(ctx context.Context, req *gradv1.DeleteRunnerRequest) (*gradv1.DeleteRunnerResponse, error) {
	// Validate request
	if req.RunnerId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "runner_id is required")
	}

	// Call service layer
	err := s.runnerService.DeleteRunner(ctx, req.RunnerId)
	if err != nil {
		return nil, s.mapServiceError(err)
	}

	return &gradv1.DeleteRunnerResponse{
		Message: fmt.Sprintf("runner %s deletion initiated", req.RunnerId),
	}, nil
}

// ListRunners returns all available runners
func (s *Server) ListRunners(ctx context.Context, req *gradv1.ListRunnersRequest) (*gradv1.ListRunnersResponse, error) {
	// Validate request
	if req.Limit < 0 || req.Offset < 0 {
		return nil, status.Errorf(codes.InvalidArgument, "limit and offset must be non-negative")
	}

	// Convert proto request to domain options
	opts := service.FromProtoListOptions(req.Status, req.Limit, req.Offset)

	// Call service layer
	runners, total, err := s.runnerService.ListRunners(ctx, opts)
	if err != nil {
		return nil, s.mapServiceError(err)
	}

	// Convert domain runners to proto runners
	protoRunners := make([]*gradv1.Runner, len(runners))
	for i, runner := range runners {
		protoRunners[i] = runner.ToProto()
	}

	return &gradv1.ListRunnersResponse{
		Runners: protoRunners,
		Total:   total,
	}, nil
}

// ExecuteCode executes code in a specific runner
func (s *Server) ExecuteCode(ctx context.Context, req *gradv1.ExecuteCodeRequest) (*gradv1.ExecuteCodeResponse, error) {
	// Validate request
	if err := s.validateExecuteCodeRequest(req); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	// Convert proto request to domain request
	domainReq := service.FromProtoExecuteCodeRequest(req)

	// Call service layer
	result, err := s.runnerService.ExecuteCode(ctx, domainReq)
	if err != nil {
		return nil, s.mapServiceError(err)
	}

	// Convert domain result to proto response
	return result.ToProto(), nil
}

// GetRunner returns details about a specific runner
func (s *Server) GetRunner(ctx context.Context, req *gradv1.GetRunnerRequest) (*gradv1.GetRunnerResponse, error) {
	// Validate request
	if req.RunnerId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "runner_id is required")
	}

	// Call service layer
	runner, err := s.runnerService.GetRunner(ctx, req.RunnerId)
	if err != nil {
		return nil, s.mapServiceError(err)
	}

	return &gradv1.GetRunnerResponse{
		Runner: runner.ToProto(),
	}, nil
}

// validateCreateRunnerRequest validates the create runner request
func (s *Server) validateCreateRunnerRequest(req *gradv1.CreateRunnerRequest) error {
	// Name validation (optional but if provided, must be valid)
	if req.Name != "" && len(req.Name) > 100 {
		return errors.New("name must be less than 100 characters")
	}

	// Note: Resource requirements are ignored - preset configuration (2c2g40g) is always used

	return nil
}

// validateExecuteCodeRequest validates the execute code request
func (s *Server) validateExecuteCodeRequest(req *gradv1.ExecuteCodeRequest) error {
	if req.RunnerId == "" {
		return errors.New("runner_id is required")
	}

	if req.Code == "" {
		return errors.New("code is required")
	}

	if req.Timeout < 0 {
		return errors.New("timeout must be non-negative")
	}

	// Set default timeout if not provided
	if req.Timeout == 0 {
		req.Timeout = 30 // 30 seconds default
	}

	return nil
}

// mapServiceError maps domain errors to gRPC status errors
func (s *Server) mapServiceError(err error) error {
	switch {
	case errors.Is(err, service.ErrRunnerNotFound):
		return status.Errorf(codes.NotFound, "runner not found")
	case errors.Is(err, service.ErrRunnerNotRunning):
		return status.Errorf(codes.FailedPrecondition, "runner is not running")
	case errors.Is(err, service.ErrInvalidRequest):
		return status.Errorf(codes.InvalidArgument, "invalid request")
	case errors.Is(err, service.ErrResourceConflict):
		return status.Errorf(codes.AlreadyExists, "resource conflict")
	case errors.Is(err, service.ErrKubernetesAPI):
		return status.Errorf(codes.Internal, "kubernetes API error")
	case errors.Is(err, service.ErrCodeExecution):
		return status.Errorf(codes.Internal, "code execution failed")
	default:
		// Log the error for debugging in production
		return status.Errorf(codes.Internal, "internal server error")
	}
}
