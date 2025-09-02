package service

import (
	"context"
	"log/slog"
	"time"
)

// CleanupService manages inactive runner cleanup
type CleanupService struct {
	runnerService   RunnerService
	activityTracker *ActivityTracker
	cleanupInterval time.Duration
	inactiveTimeout time.Duration
	stopCh          chan struct{}
}

// NewCleanupService creates a new cleanup service
func NewCleanupService(runnerService RunnerService, activityTracker *ActivityTracker) *CleanupService {
	return &CleanupService{
		runnerService:   runnerService,
		activityTracker: activityTracker,
		cleanupInterval: 1 * time.Minute,  // Check every 1 minute
		inactiveTimeout: 5 * time.Minute,  // Delete runners inactive for >5 minutes
		stopCh:          make(chan struct{}),
	}
}

// Start begins the cleanup background task
func (cs *CleanupService) Start(ctx context.Context) {
	ticker := time.NewTicker(cs.cleanupInterval)
	defer ticker.Stop()

	slog.Info("Starting cleanup service", 
		"cleanup_interval", cs.cleanupInterval.String(), 
		"inactive_timeout", cs.inactiveTimeout.String())

	for {
		select {
		case <-ticker.C:
			cs.cleanupInactiveRunners(ctx)
		case <-cs.stopCh:
			slog.Info("Cleanup service stopped")
			return
		case <-ctx.Done():
			slog.Info("Cleanup service stopping due to context cancellation")
			return
		}
	}
}

// Stop stops the cleanup service
func (cs *CleanupService) Stop() {
	close(cs.stopCh)
}

// cleanupInactiveRunners performs the actual cleanup of inactive runners
func (cs *CleanupService) cleanupInactiveRunners(ctx context.Context) {
	// Get summary of tracked runners before cleanup
	allTracked := cs.activityTracker.GetAllTrackedRunners()
	totalTrackedCount := len(allTracked)
	
	slog.Info("Starting cleanup cycle", 
		"total_tracked_runners", totalTrackedCount,
		"inactive_timeout", cs.inactiveTimeout.String())

	// Get list of inactive runners
	inactiveRunners := cs.activityTracker.GetInactiveRunners(cs.inactiveTimeout)
	
	if len(inactiveRunners) == 0 {
		slog.Info("Cleanup cycle completed - no inactive runners found",
			"total_tracked_runners", totalTrackedCount)
		return
	}

	slog.Info("Beginning cleanup of inactive runners", 
		"total_runners", totalTrackedCount,
		"inactive_runners_count", len(inactiveRunners), 
		"runners_to_cleanup", inactiveRunners)

	// Track cleanup results
	var (
		successfulDeletes = 0
		alreadyStopped    = 0
		failedDeletes     = 0
	)

	// Delete each inactive runner
	for _, runnerID := range inactiveRunners {
		deleted, err := cs.deleteInactiveRunner(ctx, runnerID)
		if err != nil {
			failedDeletes++
			slog.Error("Failed to delete inactive runner", 
				"runner_id", runnerID, 
				"error", err)
		} else if deleted {
			successfulDeletes++
			slog.Info("Successfully deleted inactive runner", "runner_id", runnerID)
			// Remove from activity tracker
			cs.activityTracker.RemoveRunner(runnerID)
		} else {
			alreadyStopped++
			slog.Info("Removed inactive runner from tracking (already stopped)", "runner_id", runnerID)
			// Remove from activity tracker
			cs.activityTracker.RemoveRunner(runnerID)
		}
	}

	// Final cleanup summary
	remainingTracked := len(cs.activityTracker.GetAllTrackedRunners())
	slog.Info("Cleanup cycle completed",
		"initial_tracked_runners", totalTrackedCount,
		"inactive_runners_processed", len(inactiveRunners),
		"successful_deletes", successfulDeletes,
		"already_stopped", alreadyStopped,
		"failed_deletes", failedDeletes,
		"remaining_tracked_runners", remainingTracked)
}

// deleteInactiveRunner deletes a specific inactive runner
// Returns (deleted, error) where deleted indicates if the runner was actually deleted
func (cs *CleanupService) deleteInactiveRunner(ctx context.Context, runnerID string) (bool, error) {
	slog.Debug("Attempting to delete inactive runner", "runner_id", runnerID)
	
	// First verify the runner still exists and get its current state
	runner, err := cs.runnerService.GetRunner(ctx, runnerID)
	if err != nil {
		// If runner doesn't exist, remove from tracker and return success
		if err == ErrRunnerNotFound {
			slog.Info("Runner no longer exists, removing from tracking", "runner_id", runnerID)
			cs.activityTracker.RemoveRunner(runnerID)
			return false, nil
		}
		slog.Error("Failed to get runner for cleanup", "runner_id", runnerID, "error", err)
		return false, err
	}

	slog.Debug("Runner found for cleanup evaluation", 
		"runner_id", runnerID, 
		"status", runner.Status,
		"created_at", runner.CreatedAt)

	// Only delete running or creating runners - don't delete already stopped/error runners
	if runner.Status == RunnerStatusStopped || runner.Status == RunnerStatusError {
		slog.Info("Skipping deletion of already stopped/error runner", 
			"runner_id", runnerID, 
			"status", runner.Status)
		cs.activityTracker.RemoveRunner(runnerID)
		return false, nil
	}

	// Delete the runner
	slog.Info("Deleting inactive runner", 
		"runner_id", runnerID, 
		"status", runner.Status,
		"last_active", cs.activityTracker.GetLastActiveTime(runnerID))
	
	err = cs.runnerService.DeleteRunner(ctx, runnerID)
	if err != nil {
		slog.Error("Failed to delete runner", "runner_id", runnerID, "error", err)
		return false, err
	}

	slog.Info("Successfully initiated deletion of inactive runner", "runner_id", runnerID)
	return true, nil
}