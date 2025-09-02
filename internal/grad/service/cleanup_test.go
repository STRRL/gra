package service

import (
	"context"
	"testing"
	"time"
)

// mockRunnerService implements RunnerService for testing
type mockRunnerService struct {
	runners         map[string]*Runner
	deletedRunners  []string
	shouldFailGet   bool
	shouldFailDelete bool
}

func newMockRunnerService() *mockRunnerService {
	return &mockRunnerService{
		runners:        make(map[string]*Runner),
		deletedRunners: make([]string, 0),
	}
}

func (m *mockRunnerService) CreateRunner(ctx context.Context, req *CreateRunnerRequest) (*Runner, error) {
	return nil, nil // Not needed for cleanup tests
}

func (m *mockRunnerService) DeleteRunner(ctx context.Context, runnerID string) error {
	if m.shouldFailDelete {
		return ErrKubernetesAPI
	}
	m.deletedRunners = append(m.deletedRunners, runnerID)
	delete(m.runners, runnerID)
	return nil
}

func (m *mockRunnerService) ListRunners(ctx context.Context, opts *ListOptions) ([]*Runner, int32, error) {
	return nil, 0, nil // Not needed for cleanup tests
}

func (m *mockRunnerService) GetRunner(ctx context.Context, runnerID string) (*Runner, error) {
	if m.shouldFailGet {
		return nil, ErrRunnerNotFound
	}
	if runner, exists := m.runners[runnerID]; exists {
		return runner, nil
	}
	return nil, ErrRunnerNotFound
}

func (m *mockRunnerService) ExecuteCommandStream(ctx context.Context, req *ExecuteCommandRequest, stdoutCh, stderrCh chan<- []byte) (int32, error) {
	return 0, nil // Not needed for cleanup tests
}

func TestCleanupService(t *testing.T) {
	mockService := newMockRunnerService()
	tracker := NewActivityTracker()
	
	// Create cleanup service with short intervals for testing
	cleanupService := NewCleanupService(mockService, tracker)
	cleanupService.cleanupInterval = 100 * time.Millisecond
	cleanupService.inactiveTimeout = 200 * time.Millisecond

	// Add some test runners
	runner1 := &Runner{ID: "runner-1", Status: RunnerStatusRunning}
	runner2 := &Runner{ID: "runner-2", Status: RunnerStatusRunning}
	runner3 := &Runner{ID: "runner-3", Status: RunnerStatusStopped}

	mockService.runners["runner-1"] = runner1
	mockService.runners["runner-2"] = runner2
	mockService.runners["runner-3"] = runner3

	// Set old activity times to simulate inactive runners
	oldTime := time.Now().Add(-5 * time.Minute)
	tracker.lastActiveTimes["runner-1"] = oldTime
	tracker.lastActiveTimes["runner-2"] = oldTime
	tracker.lastActiveTimes["runner-3"] = oldTime

	// Trigger cleanup manually
	ctx := context.Background()
	cleanupService.cleanupInactiveRunners(ctx)

	// Check that only running runners were deleted (stopped runner should be ignored)
	if len(mockService.deletedRunners) != 2 {
		t.Errorf("Expected 2 deleted runners, got %d: %v", len(mockService.deletedRunners), mockService.deletedRunners)
	}

	// Verify the correct runners were deleted
	deleted := make(map[string]bool)
	for _, id := range mockService.deletedRunners {
		deleted[id] = true
	}

	if !deleted["runner-1"] || !deleted["runner-2"] {
		t.Errorf("Expected runner-1 and runner-2 to be deleted, got: %v", mockService.deletedRunners)
	}

	if deleted["runner-3"] {
		t.Error("runner-3 should not have been deleted (already stopped)")
	}

	// Verify that stopped runner was removed from tracker but not deleted
	if _, exists := tracker.lastActiveTimes["runner-3"]; exists {
		t.Error("Expected runner-3 to be removed from activity tracker")
	}
}

func TestCleanupServiceErrorHandling(t *testing.T) {
	mockService := newMockRunnerService()
	tracker := NewActivityTracker()
	
	cleanupService := NewCleanupService(mockService, tracker)

	// Test runner not found (should be handled gracefully)
	tracker.lastActiveTimes["nonexistent-runner"] = time.Now().Add(-10 * time.Minute)
	mockService.shouldFailGet = true

	ctx := context.Background()
	cleanupService.cleanupInactiveRunners(ctx)

	// Should not panic and should remove from tracker
	if _, exists := tracker.lastActiveTimes["nonexistent-runner"]; exists {
		t.Error("Expected nonexistent runner to be removed from tracker")
	}
}

func TestCleanupServiceLifecycle(t *testing.T) {
	mockService := newMockRunnerService()
	tracker := NewActivityTracker()
	
	cleanupService := NewCleanupService(mockService, tracker)
	cleanupService.cleanupInterval = 50 * time.Millisecond

	// Start cleanup service
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	
	go func() {
		cleanupService.Start(ctx)
		done <- struct{}{}
	}()

	// Let it run for a short time
	time.Sleep(150 * time.Millisecond)

	// Stop it
	cancel()
	cleanupService.Stop()

	// Wait for it to finish
	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Error("Cleanup service did not stop within timeout")
	}
}