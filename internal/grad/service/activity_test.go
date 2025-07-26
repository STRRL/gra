package service

import (
	"testing"
	"time"
)

func TestActivityTracker(t *testing.T) {
	tracker := NewActivityTracker()

	// Test UpdateLastActiveTime and GetLastActiveTime
	runnerID := "test-runner-1"
	
	// Initially, runner should have zero time
	lastActive := tracker.GetLastActiveTime(runnerID)
	if !lastActive.IsZero() {
		t.Errorf("Expected zero time for new runner, got %v", lastActive)
	}

	// Update active time
	before := time.Now()
	tracker.UpdateLastActiveTime(runnerID)
	after := time.Now()
	
	lastActive = tracker.GetLastActiveTime(runnerID)
	if lastActive.Before(before) || lastActive.After(after) {
		t.Errorf("Expected last active time between %v and %v, got %v", before, after, lastActive)
	}

	// Test GetInactiveRunners
	// Set an old time for the runner
	oldTime := time.Now().Add(-10 * time.Minute)
	tracker.lastActiveTimes[runnerID] = oldTime

	// Should be inactive for 5 minute threshold
	inactiveRunners := tracker.GetInactiveRunners(5 * time.Minute)
	if len(inactiveRunners) != 1 || inactiveRunners[0] != runnerID {
		t.Errorf("Expected 1 inactive runner %s, got %v", runnerID, inactiveRunners)
	}

	// Should not be inactive for 15 minute threshold
	inactiveRunners = tracker.GetInactiveRunners(15 * time.Minute)
	if len(inactiveRunners) != 0 {
		t.Errorf("Expected 0 inactive runners, got %v", inactiveRunners)
	}

	// Test RemoveRunner
	tracker.RemoveRunner(runnerID)
	lastActive = tracker.GetLastActiveTime(runnerID)
	if !lastActive.IsZero() {
		t.Errorf("Expected zero time after removal, got %v", lastActive)
	}

	// Test GetAllTrackedRunners
	tracker.UpdateLastActiveTime("runner-1")
	tracker.UpdateLastActiveTime("runner-2")
	tracked := tracker.GetAllTrackedRunners()
	if len(tracked) != 2 {
		t.Errorf("Expected 2 tracked runners, got %d", len(tracked))
	}
}

func TestActivityTrackerConcurrency(t *testing.T) {
	tracker := NewActivityTracker()
	done := make(chan struct{})

	// Test concurrent updates
	go func() {
		for i := 0; i < 100; i++ {
			tracker.UpdateLastActiveTime("runner-1")
		}
		done <- struct{}{}
	}()

	go func() {
		for i := 0; i < 100; i++ {
			tracker.GetLastActiveTime("runner-1")
		}
		done <- struct{}{}
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Verify final state
	lastActive := tracker.GetLastActiveTime("runner-1")
	if lastActive.IsZero() {
		t.Error("Expected non-zero time after concurrent updates")
	}
}