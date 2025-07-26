package grpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	gradv1 "github.com/strrl/gra/gen/grad/v1"
	"github.com/strrl/gra/internal/grad/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server implements the gRPC RunnerService and ExecuteService as a thin controller
type Server struct {
	gradv1.UnimplementedRunnerServiceServer
	gradv1.UnimplementedExecuteServiceServer
	runnerService service.RunnerService
	executeService service.ExecuteService
}

// NewServer creates a new gRPC server instance
func NewServer(runnerService service.RunnerService, executeService service.ExecuteService) *Server {
	return &Server{
		runnerService:  runnerService,
		executeService: executeService,
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

// ExecuteCommandStream executes a command in a specific runner with streaming output
func (s *Server) ExecuteCommandStream(req *gradv1.ExecuteCommandRequest, stream gradv1.RunnerService_ExecuteCommandStreamServer) error {
	// Validate request
	if err := s.validateExecuteCommandRequest(req); err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	// Convert proto request to domain request
	domainReq := service.FromProtoExecuteCommandRequest(req)

	// Create channels for streaming
	// Note: stdoutCh and stderrCh will be closed by the sender (Kubernetes layer)
	stdoutCh := make(chan []byte, 100)
	stderrCh := make(chan []byte, 100)
	
	// exitCh and errCh are owned by this gRPC layer
	exitCh := make(chan int32, 1)
	errCh := make(chan error, 1)

	// Start command execution in a goroutine
	go func() {
		// Only close channels that this goroutine owns/sends to
		defer close(exitCh)
		defer close(errCh)

		exitCode, err := s.runnerService.ExecuteCommandStream(stream.Context(), domainReq, stdoutCh, stderrCh)
		if err != nil {
			errCh <- err
			return
		}
		exitCh <- exitCode
	}()

	// Stream the output
	for {
		select {
		case data, ok := <-stdoutCh:
			if !ok {
				stdoutCh = nil
				continue
			}
			if len(data) > 0 {
				if err := stream.Send(&gradv1.ExecuteCommandStreamResponse{
					Type: gradv1.StreamType_STREAM_TYPE_STDOUT,
					Data: data,
				}); err != nil {
					return err
				}
			}

		case data, ok := <-stderrCh:
			if !ok {
				stderrCh = nil
				continue
			}
			if len(data) > 0 {
				if err := stream.Send(&gradv1.ExecuteCommandStreamResponse{
					Type: gradv1.StreamType_STREAM_TYPE_STDERR,
					Data: data,
				}); err != nil {
					return err
				}
			}

		case exitCode := <-exitCh:
			// Send final exit message
			return stream.Send(&gradv1.ExecuteCommandStreamResponse{
				Type:     gradv1.StreamType_STREAM_TYPE_EXIT,
				ExitCode: exitCode,
			})

		case err := <-errCh:
			return s.mapServiceError(err)

		case <-stream.Context().Done():
			return stream.Context().Err()
		}

		// If both stdout and stderr channels are closed, wait for exit
		if stdoutCh == nil && stderrCh == nil {
			select {
			case exitCode := <-exitCh:
				return stream.Send(&gradv1.ExecuteCommandStreamResponse{
					Type:     gradv1.StreamType_STREAM_TYPE_EXIT,
					ExitCode: exitCode,
				})
			case err := <-errCh:
				return s.mapServiceError(err)
			case <-stream.Context().Done():
				return stream.Context().Err()
			}
		}
	}
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

// validateExecuteCommandRequest validates the execute command request
func (s *Server) validateExecuteCommandRequest(req *gradv1.ExecuteCommandRequest) error {
	if req.RunnerId == "" {
		return errors.New("runner_id is required")
	}

	if req.Command == "" {
		return errors.New("command is required")
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

// validateExecuteServiceCommandRequest validates the execute command request for ExecuteService
// This is similar to validateExecuteCommandRequest but doesn't require runner_id
func (s *Server) validateExecuteServiceCommandRequest(req *gradv1.ExecuteCommandRequest) error {
	if req.Command == "" {
		return errors.New("command is required")
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

// ExecuteCommand executes a command with automatic runner provisioning
func (s *Server) ExecuteCommand(req *gradv1.ExecuteCommandRequest, stream gradv1.ExecuteService_ExecuteCommandServer) error {
	// Validate request (without runner_id requirement)
	if err := s.validateExecuteServiceCommandRequest(req); err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	// Convert proto request to domain request
	domainReq := service.FromProtoExecuteCommandRequest(req)

	// Create channels for streaming
	// Note: stdoutCh and stderrCh will be closed by the sender (service layer)
	stdoutCh := make(chan []byte, 100)
	stderrCh := make(chan []byte, 100)
	
	// exitCh and errCh are owned by this gRPC layer
	exitCh := make(chan int32, 1)
	errCh := make(chan error, 1)

	// Start command execution in a goroutine
	go func() {
		// Only close channels that this goroutine owns/sends to
		defer close(exitCh)
		defer close(errCh)

		exitCode, err := s.executeService.ExecuteCommand(stream.Context(), domainReq, stdoutCh, stderrCh)
		if err != nil {
			errCh <- err
			return
		}
		exitCh <- exitCode
	}()

	// Stream the output (same logic as ExecuteCommandStream)
	for {
		select {
		case data, ok := <-stdoutCh:
			if !ok {
				stdoutCh = nil
				continue
			}
			if len(data) > 0 {
				if err := stream.Send(&gradv1.ExecuteCommandStreamResponse{
					Type: gradv1.StreamType_STREAM_TYPE_STDOUT,
					Data: data,
				}); err != nil {
					return err
				}
			}

		case data, ok := <-stderrCh:
			if !ok {
				stderrCh = nil
				continue
			}
			if len(data) > 0 {
				if err := stream.Send(&gradv1.ExecuteCommandStreamResponse{
					Type: gradv1.StreamType_STREAM_TYPE_STDERR,
					Data: data,
				}); err != nil {
					return err
				}
			}

		case exitCode := <-exitCh:
			// Send final exit message
			return stream.Send(&gradv1.ExecuteCommandStreamResponse{
				Type:     gradv1.StreamType_STREAM_TYPE_EXIT,
				ExitCode: exitCode,
			})

		case err := <-errCh:
			return s.mapServiceError(err)

		case <-stream.Context().Done():
			return stream.Context().Err()
		}

		// If both stdout and stderr channels are closed, wait for exit
		if stdoutCh == nil && stderrCh == nil {
			select {
			case exitCode := <-exitCh:
				return stream.Send(&gradv1.ExecuteCommandStreamResponse{
					Type:     gradv1.StreamType_STREAM_TYPE_EXIT,
					ExitCode: exitCode,
				})
			case err := <-errCh:
				return s.mapServiceError(err)
			case <-stream.Context().Done():
				return stream.Context().Err()
			}
		}
	}
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
		slog.Error("Kubernetes API error", "error", err)
		return status.Errorf(codes.Internal, "kubernetes API error: %v", err)
	case errors.Is(err, service.ErrCommandExecution):
		slog.Error("Command execution error", "error", err)
		return status.Errorf(codes.Internal, "command execution failed: %v", err)
	default:
		// Log unknown errors for debugging
		slog.Error("Unknown service error", "error", err)
		return status.Errorf(codes.Internal, "internal server error: %v", err)
	}
}
