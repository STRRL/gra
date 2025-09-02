package service

import (
	"log/slog"
	"sync"
	"time"
)

// ActivityTracker manages the last active time for runners in memory
type ActivityTracker struct {
	mu             sync.RWMutex
	lastActiveTimes map[string]time.Time
}

// NewActivityTracker creates a new activity tracker
func NewActivityTracker() *ActivityTracker {
	return &ActivityTracker{
		lastActiveTimes: make(map[string]time.Time),
	}
}

// UpdateLastActiveTime records the last active time for a runner
func (at *ActivityTracker) UpdateLastActiveTime(runnerID string) {
	at.mu.Lock()
	defer at.mu.Unlock()
	now := time.Now()
	at.lastActiveTimes[runnerID] = now
	slog.Debug("Updated runner activity", 
		"runner_id", runnerID, 
		"last_active", now,
		"total_tracked", len(at.lastActiveTimes))
}

// GetLastActiveTime retrieves the last active time for a runner
// Returns zero time if runner has no recorded activity
func (at *ActivityTracker) GetLastActiveTime(runnerID string) time.Time {
	at.mu.RLock()
	defer at.mu.RUnlock()
	return at.lastActiveTimes[runnerID]
}

// GetInactiveRunners returns runners that have been inactive for longer than the specified duration
// Runners with no recorded activity are ignored (returns empty slice if no runners are inactive)
func (at *ActivityTracker) GetInactiveRunners(inactiveDuration time.Duration) []string {
	at.mu.RLock()
	defer at.mu.RUnlock()

	var inactiveRunners []string
	now := time.Now()
	totalTracked := len(at.lastActiveTimes)

	for runnerID, lastActive := range at.lastActiveTimes {
		inactiveFor := now.Sub(lastActive)
		if inactiveFor > inactiveDuration {
			inactiveRunners = append(inactiveRunners, runnerID)
			slog.Debug("Found inactive runner", 
				"runner_id", runnerID,
				"last_active", lastActive,
				"inactive_for", inactiveFor.String(),
				"threshold", inactiveDuration.String())
		}
	}

	slog.Info("Activity tracker scan completed",
		"total_tracked_runners", totalTracked,
		"inactive_runners_found", len(inactiveRunners),
		"inactive_threshold", inactiveDuration.String())

	return inactiveRunners
}

// RemoveRunner removes a runner from activity tracking
func (at *ActivityTracker) RemoveRunner(runnerID string) {
	at.mu.Lock()
	defer at.mu.Unlock()
	
	_, existed := at.lastActiveTimes[runnerID]
	delete(at.lastActiveTimes, runnerID)
	
	if existed {
		slog.Info("Removed runner from activity tracking", 
			"runner_id", runnerID,
			"remaining_tracked", len(at.lastActiveTimes))
	} else {
		slog.Debug("Attempted to remove non-tracked runner", 
			"runner_id", runnerID,
			"total_tracked", len(at.lastActiveTimes))
	}
}

// GetAllTrackedRunners returns all runner IDs currently being tracked
func (at *ActivityTracker) GetAllTrackedRunners() []string {
	at.mu.RLock()
	defer at.mu.RUnlock()

	runners := make([]string, 0, len(at.lastActiveTimes))
	for runnerID := range at.lastActiveTimes {
		runners = append(runners, runnerID)
	}
	return runners
}